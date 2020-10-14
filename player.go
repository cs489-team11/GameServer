package server

import "github.com/cs489-team11/server/pb"

type username string

type player struct {
	username username
	points   int64
	stream   pb.Game_StreamServer
	// time until lottery ??
}
