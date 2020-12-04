package server

import (
	"fmt"
	"log"
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/cs489-team11/server/pb"
	"github.com/google/uuid"
)

type gameID string
type gameState int

const (
	waitingState gameState = iota
	activeState
	// finished - a virtual state, which exists in a state machine,
	// but is not needed for implementation.
)

// GameConfig contains game configuration variables, which
// can be subject to change.
type GameConfig struct {
	duration              int32 // total game time in seconds
	playerPoints          int32
	bankPointsPerPlayer   int32
	creditInterest        int32
	depositInterest       int32
	creditTime            int32
	depositTime           int32
	theftTime             int32
	theftPercentage       int32
	lotteryTime           int32
	lotteryMaxWin         int32
	questionWinPercentage int32
}

// NewGameConfig returns pointer to a newly created
// instance of a GameConfig type.
func NewGameConfig(
	duration int32,
	playerPoints int32,
	bankPointsPerPlayer int32,
	creditInterest int32,
	depositInterest int32,
	creditTime int32,
	depositTime int32,
	theftTime int32,
	theftPercentage int32,
	lotteryTime int32,
	lotteryMaxWin int32,
	questionWinPercentage int32,
) GameConfig {
	return GameConfig{
		duration:              duration,
		playerPoints:          playerPoints,
		bankPointsPerPlayer:   bankPointsPerPlayer,
		creditInterest:        creditInterest,
		depositInterest:       depositInterest,
		creditTime:            creditTime,
		depositTime:           depositTime,
		theftTime:             theftTime,
		theftPercentage:       theftPercentage,
		lotteryTime:           lotteryTime,
		lotteryMaxWin:         lotteryMaxWin,
		questionWinPercentage: questionWinPercentage,
	}
}

// Struct representing a single game.
// Since there is only single [secondary] bank, its info
// is also contained in this struct.
type game struct {
	mutex             sync.RWMutex
	gameID            gameID
	state             gameState
	config            GameConfig
	players           map[userID]*player
	bankPoints        int32
	lotteryCellValues []int32
}

func getNumberProportion(num int32, percentage int32) int32 {
	floatRes := float64(num) * float64(percentage) / 100.0
	res := int32(math.Ceil(floatRes))
	return res
}

func generateLotteryCellValues(maxWin int32) []int32 {
	// TODO: put cellCount to game config
	cellCount := 9

	res := make([]int32, cellCount)

	winPoints1 := int32(0)
	res[0] = winPoints1
	res[1] = winPoints1
	winPoints2 := getNumberProportion(maxWin, 20)
	res[2] = winPoints2
	res[3] = winPoints2
	winPoints3 := getNumberProportion(maxWin, 30)
	res[4] = winPoints3
	res[5] = winPoints3
	winPoints4 := getNumberProportion(maxWin, 60)
	res[6] = winPoints4
	res[7] = winPoints4
	winPoints5 := maxWin
	res[8] = winPoints5

	return res
}

// Creates new game in waiting state.
func newGame(config GameConfig) *game {
	gameID := gameID(uuid.New().String())
	lotteryCellValues := generateLotteryCellValues(config.lotteryMaxWin)
	return &game{
		gameID:            gameID,
		state:             waitingState,
		config:            config,
		players:           make(map[userID]*player),
		bankPoints:        0, // to be calculated in "start" function
		lotteryCellValues: lotteryCellValues,
	}
}

// Creates a new player with a provided username
// and adds it to the game.
// NOTE: only should be called on game in waiting state.
func (g *game) addPlayer(username username) userID {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	player := newPlayer(username, g.config.playerPoints)
	g.players[player.userID] = player

	// broadcasting player joining
	go func() {
		msg := g.getJoinMessage(player)
		g.broadcast(msg)
	}()

	return player.userID
}

// Deletes player from the game.
// NOTE: only should be called on game in waiting state.
func (g *game) deletePlayer(userID userID) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	delete(g.players, userID)

	// broadcasting player leaving
	go func() {
		msg := g.getLeaveMessage(userID)
		g.broadcast(msg)
	}()
}

// NOTE: This function uses readlock, so it has to be used carefully.
func (g *game) getWinnerID() userID {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	noUserID := userID("")
	winnerID := noUserID
	for _, player := range g.players {
		if winnerID == noUserID || player.points > g.players[winnerID].points {
			winnerID = player.userID
		}
	}
	return winnerID
}

func (g *game) start() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.state = activeState
	// bank points are calculated
	g.bankPoints = int32(len(g.players)) * g.config.bankPointsPerPlayer

	// marking each player as if he has just played the lottery
	// users can play their first lottery after g.config.lotteryTime seconds.
	for _, player := range g.players {
		player.updateLastLotteryTime()
	}

	// broadcasting game start
	go func() {
		msg := g.getStartMessage()
		g.broadcast(msg)
	}()

	// launch theft timer
	time.AfterFunc(time.Duration(g.config.theftTime)*time.Second, func() {
		g.doTheft()
	})
}

func (g *game) finish() {
	go func() {
		winnerUserID := g.getWinnerID()
		msg := g.getFinishMessage(winnerUserID)
		g.broadcast(msg)
	}()
}

// useCredit returns "True" and empty string, if credit can be granted.
// Otherwise, it will return "False" and explanation why credit has not
// been granted.
func (g *game) useCredit(userID userID, val int32) (bool, string, error) {
	player, ok := g.players[userID]
	if !ok {
		return false, "", fmt.Errorf("there is no player with id %v in the game", userID)
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	// bank doesn't have enough points to give the credit
	// NOTE: this check can be deleted to allow bank to go down a bit
	// but in that case, we would need to check that the user doesn't borrow too much
	if g.bankPoints < val {
		return false, fmt.Sprintf("bank cannot grant the credit due to bank's undisclosed policies"), nil
	}
	g.bankPoints -= val
	player.points += val

	time.AfterFunc(time.Duration(g.config.creditTime)*time.Second, func() {
		g.returnCredit(userID, val)
	})

	go func() {
		msg := g.getUseCreditMessage(userID, val)
		g.broadcast(msg)
	}()

	return true, "", nil
}

// useDeposit returns "True" and empty string, if deposit can be granted.
// Otherwise, it will return "False" and explanation why deposit has not
// been granted.
func (g *game) useDeposit(userID userID, val int32) (bool, string, error) {
	player, ok := g.players[userID]
	if !ok {
		return false, "", fmt.Errorf("there is no player with id %v in the game", userID)
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	// We will grant all deposit requests
	// However, later, if the person puts too much money for deposit and we
	// think we cannot return back money with interest, we could reject
	// the request.
	g.bankPoints += val
	player.points -= val

	time.AfterFunc(time.Duration(g.config.depositTime)*time.Second, func() {
		g.returnDeposit(userID, val)
	})

	go func() {
		msg := g.getUseDepositMessage(userID, val)
		g.broadcast(msg)
	}()

	return true, "", nil
}

func (g *game) returnCredit(userID userID, val int32) {
	player, ok := g.players[userID]
	if !ok {
		log.Printf("returnCredit has been called with user %v, who is not in this game", userID)
		return
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	floatInterest := float64(val) * float64(g.config.creditInterest) / 100.0
	interest := int32(math.Ceil(floatInterest))
	valWithInterest := val + interest

	g.bankPoints += valWithInterest
	player.points -= valWithInterest

	go func() {
		msg := g.getReturnCreditMessage(userID, valWithInterest)
		g.broadcast(msg)
	}()
}

func (g *game) returnDeposit(userID userID, val int32) {
	player, ok := g.players[userID]
	if !ok {
		log.Printf("returnDeposit has been called with user %v, who is not in this game", userID)
		return
	}

	g.mutex.Lock()
	defer g.mutex.Unlock()

	floatInterest := float64(val) * float64(g.config.depositInterest) / 100.0
	interest := int32(math.Ceil(floatInterest))
	valWithInterest := val + interest

	g.bankPoints -= valWithInterest
	player.points += valWithInterest

	go func() {
		msg := g.getReturnDepositMessage(userID, valWithInterest)
		g.broadcast(msg)
	}()
}

func (g *game) playLottery(userID userID, cellIndex int32) (bool, []int32, int32, error) {
	success := false
	cellValues := []int32{}
	winPoints := int32(0)

	player, ok := g.players[userID]
	if !ok {
		errMsg := fmt.Sprintf("playLottery has been called with user %v, who is not in this game", userID)
		log.Printf(errMsg)
		return success, cellValues, winPoints, fmt.Errorf(errMsg)
	}

	// locking for reads and writes
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !player.canPlayLottery(g.config.lotteryTime) {
		timePassed := time.Since(player.lastLotteryTime).Seconds()
		errMsg := fmt.Sprintf(
			"please wait until next lottery time, only %f out of %d seconds have passed",
			timePassed,
			g.config.lotteryTime,
		)
		log.Println(errMsg)
		// err is nil, but success is false according to game logic
		return success, cellValues, winPoints, nil
	}

	// all conditions for lottery are correct
	// first, calculate lottery values
	cellValues = RandShuffle(g.lotteryCellValues)
	winPoints = cellValues[cellIndex-1]
	success = true

	// record that player have just played lottery
	player.updateLastLotteryTime()

	// only if player won some amount
	if success && winPoints >= 0 {
		// add points to player
		player.points += winPoints
		g.bankPoints -= winPoints

		go func() {
			msg := g.getLotteryMessage(player.userID, winPoints)
			g.broadcast(msg)
		}()
	}

	return success, cellValues, winPoints, nil
}

// The calling function has to acquire at least read lock
// for accurate reading of player points.
func (g *game) printPlayersPoints(preMsg string) {
	log.Println(preMsg)
	for _, player := range g.players {
		log.Printf("%s: %d points, ", player.userID, player.points)
	}
}

func (g *game) doTheft() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	var userIDs []userID
	var theftAmounts []int32

	g.printPlayersPoints("Players' points BEFORE theft")
	for userID, player := range g.players {
		floatTheftAmount := float64(player.points) * float64(g.config.theftPercentage) / 100.0
		theftAmount := int32(math.Ceil(floatTheftAmount))

		// send only if theft amount is positive number
		// if the theft amount is negative or zero, then we won't do the theft
		// and we won't send a redundant or meaningless message about it
		if theftAmount > 0 {
			player.points -= theftAmount // point deduction from player
			g.bankPoints += theftAmount  // add them to bank

			userIDs = append(userIDs, userID)
			theftAmounts = append(theftAmounts, theftAmount)
		}
	}
	g.printPlayersPoints("Players' points AFTER theft")

	go func() {
		msg := g.getTheftMessage(userIDs, theftAmounts)
		g.broadcast(msg)
		log.Printf("Theft happened as follows:\n%v", msg)
	}()

	time.AfterFunc(time.Duration(g.config.theftTime)*time.Second, func() {
		g.doTheft()
	})
}

func (g *game) setPlayerStream(userID userID, stream pb.Game_StreamServer) error {
	g.mutex.Lock() /* WRITE lock for player.setStream */
	defer g.mutex.Unlock()

	player, ok := g.players[userID]
	if !ok {
		return fmt.Errorf("setPlayerStream: invalid user id %v", userID)
	}

	player.setStream(stream)
	return nil
}

// Broadcast sends some event to all users in the game.
func (g *game) broadcast(response *pb.StreamResponse) {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	for userID, player := range g.players {
		stream := player.stream
		// WARNING: this is a dirty workaround around the problem
		// that start/deposit/etc handlers may be called before
		if stream == nil {
			continue
		}
		if err := stream.Send(response); err != nil {
			log.Printf("Could not send event to %v in game %v: %v\n", userID, g.gameID, err)
			continue
		}

		// if sent message is Start, then player marked as notified about start
		if reflect.TypeOf(response.Event) == reflect.TypeOf(pb.StreamResponse_Start_{}) {
			player.gameStartNotified = true
		}

		// if game is in active state and the player has not been notified about start,
		// then notify player about start and mark player as notified
		if g.state == activeState && !player.gameStartNotified {
			stream.Send(g.getStartMessage())
			player.gameStartNotified = true
		}
	}
}

func (g *game) getBankAsPBPlayer() *pb.Player {
	return &pb.Player{
		UserId:   "bank",
		Username: "bank",
		Points:   g.bankPoints,
	}
}

// WARNING: This function doesn't use any locks (in order not to spawn
// another goroutine). So make sure that goroutine, which calls this function
// uses at least read-lock.
// Read-lock is needed since g.players are read and the state of each player
// is read.
func (g *game) getPBPlayersWithBank() []*pb.Player {
	var players []*pb.Player
	for _, player := range g.players {
		players = append(players, player.toPBPlayer())
	}
	players = append(players, g.getBankAsPBPlayer())
	return players
}

// As this function uses Readlock, it has to be spawned in a separate goroutine.
func (g *game) getJoinMessage(player *player) *pb.StreamResponse {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	pbPlayer := player.toPBPlayer()
	res := &pb.StreamResponse{
		Event: &pb.StreamResponse_Join_{
			Join: &pb.StreamResponse_Join{
				Player: pbPlayer,
			},
		},
	}
	return res
}

// This function can be called from anywhere, as it doesn't
// refer to the state of the game and does not use any locks.
func (g *game) getLeaveMessage(userID userID) *pb.StreamResponse {
	res := &pb.StreamResponse{
		Event: &pb.StreamResponse_Leave_{
			Leave: &pb.StreamResponse_Leave{
				UserId: string(userID),
			},
		},
	}
	return res
}

// This function can be called from anywhere, as it doesn't
// refer to the state of the game and does not use any locks.
func (g *game) getStartMessage() *pb.StreamResponse {
	res := &pb.StreamResponse{
		Event: &pb.StreamResponse_Start_{
			Start: &pb.StreamResponse_Start{},
		},
	}
	return res
}

// As this function uses Readlock, it has to be spawned in a separate goroutine.
func (g *game) getFinishMessage(winnerUserID userID) *pb.StreamResponse {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	players := g.getPBPlayersWithBank()
	res := &pb.StreamResponse{
		Event: &pb.StreamResponse_Finish_{
			Finish: &pb.StreamResponse_Finish{
				Players:      players,
				WinnerUserId: string(winnerUserID),
			},
		},
	}
	return res
}

// As this function uses Readlock, it has to be spawned in a separate goroutine.
func (g *game) getUseCreditMessage(userID userID, val int32) *pb.StreamResponse {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	players := g.getPBPlayersWithBank()
	res := &pb.StreamResponse{
		Event: &pb.StreamResponse_Transaction_{
			Transaction: &pb.StreamResponse_Transaction{
				Players: players,
				Event: &pb.StreamResponse_Transaction_UseCredit_{
					UseCredit: &pb.StreamResponse_Transaction_UseCredit{
						UserId: string(userID),
						Value:  val,
					},
				},
			},
		},
	}
	return res
}

// As this function uses Readlock, it has to be spawned in a separate goroutine.
func (g *game) getUseDepositMessage(userID userID, val int32) *pb.StreamResponse {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	players := g.getPBPlayersWithBank()
	res := &pb.StreamResponse{
		Event: &pb.StreamResponse_Transaction_{
			Transaction: &pb.StreamResponse_Transaction{
				Players: players,
				Event: &pb.StreamResponse_Transaction_UseDeposit_{
					UseDeposit: &pb.StreamResponse_Transaction_UseDeposit{
						UserId: string(userID),
						Value:  val,
					},
				},
			},
		},
	}
	return res
}

// As this function uses Readlock, it has to be spawned in a separate goroutine.
func (g *game) getReturnCreditMessage(userID userID, valWithInterest int32) *pb.StreamResponse {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	players := g.getPBPlayersWithBank()
	res := &pb.StreamResponse{
		Event: &pb.StreamResponse_Transaction_{
			Transaction: &pb.StreamResponse_Transaction{
				Players: players,
				Event: &pb.StreamResponse_Transaction_ReturnCredit_{
					ReturnCredit: &pb.StreamResponse_Transaction_ReturnCredit{
						UserId: string(userID),
						Value:  valWithInterest,
					},
				},
			},
		},
	}
	return res
}

// As this function uses Readlock, it has to be spawned in a separate goroutine.
func (g *game) getReturnDepositMessage(userID userID, valWithInterest int32) *pb.StreamResponse {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	players := g.getPBPlayersWithBank()
	res := &pb.StreamResponse{
		Event: &pb.StreamResponse_Transaction_{
			Transaction: &pb.StreamResponse_Transaction{
				Players: players,
				Event: &pb.StreamResponse_Transaction_ReturnDeposit_{
					ReturnDeposit: &pb.StreamResponse_Transaction_ReturnDeposit{
						UserId: string(userID),
						Value:  valWithInterest,
					},
				},
			},
		},
	}
	return res
}

// As this function uses Readlock, it has to be spawned in a separate goroutine.
func (g *game) getTheftMessage(userIDs []userID, theftAmounts []int32) *pb.StreamResponse {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	var robbedPlayers []*pb.StreamResponse_Transaction_Theft_RobbedPlayer
	for ind := range userIDs {
		robbedPlayer := &pb.StreamResponse_Transaction_Theft_RobbedPlayer{
			UserId: string(userIDs[ind]),
			Value:  theftAmounts[ind],
		}
		robbedPlayers = append(robbedPlayers, robbedPlayer)
	}

	players := g.getPBPlayersWithBank()
	res := &pb.StreamResponse{
		Event: &pb.StreamResponse_Transaction_{
			Transaction: &pb.StreamResponse_Transaction{
				Players: players,
				Event: &pb.StreamResponse_Transaction_Theft_{
					Theft: &pb.StreamResponse_Transaction_Theft{
						RobbedPlayers: robbedPlayers,
					},
				},
			},
		},
	}
	return res
}

// As this function uses Readlock, it has to be spawned in a separate goroutine.
func (g *game) getLotteryMessage(userID userID, val int32) *pb.StreamResponse {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	players := g.getPBPlayersWithBank()
	res := &pb.StreamResponse{
		Event: &pb.StreamResponse_Transaction_{
			Transaction: &pb.StreamResponse_Transaction{
				Players: players,
				Event: &pb.StreamResponse_Transaction_Lottery_{
					Lottery: &pb.StreamResponse_Transaction_Lottery{
						UserId: string(userID),
						Value:  val,
					},
				},
			},
		},
	}
	return res
}
