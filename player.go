package server

import (
	"log"
	"sync"

	"github.com/cs489-team11/server/pb"
	"github.com/google/uuid"
)

type userID string
type username string

type player struct {
	mutex    sync.RWMutex
	userID   userID
	username username
	points   int32
	stream   pb.Game_StreamServer
}

func newPlayer(username username, points int32) *player {
	userID := userID(uuid.New().String())
	return &player{
		userID:   userID,
		username: username,
		points:   points,
		stream:   nil,
	}
}

func (p *player) setStream(stream pb.Game_StreamServer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.stream = stream
	log.Printf("Stream for user %v has been set.\n", p.userID)
}

func (p *player) toPBPlayer() *pb.Player {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return &pb.Player{
		UserId:   string(p.userID),
		Username: string(p.username),
		Points:   p.points,
	}
}
