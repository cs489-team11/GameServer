package server

import (
	"log"

	"github.com/cs489-team11/server/pb"
	"github.com/google/uuid"
)

type userID string
type username string

// this struct does not have RWMutex as its member
// this is done to avoid deadlocks between goroutines
type player struct {
	userID            userID
	username          username
	points            int32
	stream            pb.Game_StreamServer
	gameStartNotified bool
}

func newPlayer(username username, points int32) *player {
	userID := userID(uuid.New().String())
	return &player{
		userID:            userID,
		username:          username,
		points:            points,
		stream:            nil,
		gameStartNotified: false,
	}
}

// when game calls this function on player, make sure to grab
// WRITE lock on game
func (p *player) setStream(stream pb.Game_StreamServer) {
	p.stream = stream
	log.Printf("Stream for user %v has been set.\n", p.userID)
}

// when game calls this function on player, make sure to grab
// READ lock on game
func (p *player) toPBPlayer() *pb.Player {
	return &pb.Player{
		UserId:   string(p.userID),
		Username: string(p.username),
		Points:   p.points,
	}
}
