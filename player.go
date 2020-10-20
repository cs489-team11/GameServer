package server

import (
	"github.com/cs489-team11/server/pb"
	"github.com/google/uuid"
)

type userID string
type username string

type player struct {
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
	}
}
