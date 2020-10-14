package server

import (
	"log"

	"github.com/cs489-team11/server/pb"
)

type gameConfig struct {
	duration        int32 // total game time in seconds
	totalPoints     int64
	creditInterest  float32
	depositInterest float32
}

type game struct {
	gameID      int64
	players     map[username]*player
	totalPoints int64
	// credits - probably some channel through which expiration of credit time will be notified.
	// deposits - same as credits.
}

// Broadcast sends some event to all users in the game.
func (g *game) broadcast(response *pb.StreamResponse) {
	for username, player := range g.players {
		stream := player.stream
		if err := stream.Send(response); err != nil {
			log.Printf("Could not send event to %v in game %d\n", username, g.gameID)
		}
	}
}
