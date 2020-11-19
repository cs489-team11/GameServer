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
	duration            int32 // total game time in seconds
	playerPoints        int32
	bankPointsPerPlayer int32
	creditInterest      int32
	depositInterest     int32
	creditTime          int32
	depositTime         int32
	theftTime           int32
	theftPercentage     int32
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
) GameConfig {
	return GameConfig{
		duration:            duration,
		playerPoints:        playerPoints,
		bankPointsPerPlayer: bankPointsPerPlayer,
		creditInterest:      creditInterest,
		depositInterest:     depositInterest,
		creditTime:          creditTime,
		depositTime:         depositTime,
		theftTime:           theftTime,
		theftPercentage:     theftPercentage,
	}
}

// Struct representing a single game.
// Since there is only single [secondary] bank, its info
// is also contained in this struct.
type game struct {
	mutex      sync.RWMutex
	gameID     gameID
	state      gameState
	config     GameConfig
	players    map[userID]*player
	bankPoints int32
	// credits - probably some channel through which expiration of credit time will be notified.
	// deposits - same as credits.
}

// Creates new game in waiting state.
func newGame(config GameConfig) *game {
	gameID := gameID(uuid.New().String())
	return &game{
		gameID:     gameID,
		state:      waitingState,
		config:     config,
		players:    make(map[userID]*player),
		bankPoints: 0, // to be calculated in "start" function
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
	return player.userID
}

// Deletes player from the game.
// NOTE: only should be called on game in waiting state.
func (g *game) deletePlayer(userID userID) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	delete(g.players, userID)
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

// Bank points are calculated.
func (g *game) start() {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.state = activeState
	g.bankPoints = int32(len(g.players)) * g.config.bankPointsPerPlayer

	// broadcasting game start
	go func() {
		msg := g.getStartMessage()
		g.broadcast(msg)
	}()

	// TODO: launch theft timer
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
		} else {
			if reflect.TypeOf(response.Event) == reflect.TypeOf(pb.StreamResponse_Start_{}) {
				player.gameStartNotified = true
			} else if !player.gameStartNotified {
				stream.Send(g.getStartMessage())
			}
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
