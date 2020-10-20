package server

import (
	"fmt"
	"log"
	"sync"

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
}

// NewGameConfig returns pointer to a newly created
// instance of a GameConfig type.
func NewGameConfig(
	duration int32,
	playerPoints int32,
	bankPointsPerPlayer int32,
	creditInterest int32,
	depositInterest int32,
) GameConfig {
	return GameConfig{
		duration:            duration,
		playerPoints:        playerPoints,
		bankPointsPerPlayer: bankPointsPerPlayer,
		creditInterest:      creditInterest,
		depositInterest:     depositInterest,
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
func (g *game) addPlayer(username username) error {
	if g.state != waitingState {
		return fmt.Errorf("you cannot enter game after it has been started")
	}
	player := newPlayer(username, g.config.playerPoints)
	g.players[player.userID] = player
	return nil
}

// Deletes player from the game
func (g *game) deletePlayer(userID userID) error {
	if g.state != waitingState {
		return fmt.Errorf("you cannot leave game after it has been started")
	}
	delete(g.players, userID)
	return nil
}

// Bank points are calculated.
func (g *game) start() {
	g.bankPoints = int32(len(g.players)) * g.config.bankPointsPerPlayer
}

// Broadcast sends some event to all users in the game.
func (g *game) broadcast(response *pb.StreamResponse) {
	for userID, player := range g.players {
		stream := player.stream
		if err := stream.Send(response); err != nil {
			log.Printf("Could not send event to %v in game %v\n", userID, g.gameID)
		}
	}
}
