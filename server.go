package server

import (
	"context"
	"log"
	"net"

	"github.com/cs489-team11/server/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server is a type for the server, which will
// track the games, serve the user requests, maintain
// money invariant, and broadcast events to users.
type Server struct {
	listener    net.Listener
	gameConfig  GameConfig
	waitingGame *game
	activeGames map[gameID]*game
}

// NewServer will return a new instance of the server.
func NewServer(gameConfig GameConfig) *Server {
	return &Server{
		gameConfig:  gameConfig,
		waitingGame: newGame(gameConfig),
		activeGames: make(map[gameID]*game),
	}
}

func (s *Server) Join(_ context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Unimplemented")
}

func (s *Server) Leave(_ context.Context, req *pb.LeaveRequest) (*pb.LeaveResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Unimplemented")
}

func (s *Server) Start(_ context.Context, req *pb.StartRequest) (*pb.StartResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Unimplemented")
}

func (s *Server) Credit(_ context.Context, req *pb.CreditRequest) (*pb.CreditResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Unimplemented")
}

func (s *Server) Deposit(_ context.Context, req *pb.DepositRequest) (*pb.DepositResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Unimplemented")
}

func (s *Server) Stream(req *pb.StreamRequest, srv pb.Game_StreamServer) error {
	return status.Errorf(codes.Unimplemented, "Unimplemented")
}

// Listen makes server listen for tcp connections on specified
// server address.
func (s *Server) Listen(servAddr string) (string, error) {
	listener, err := net.Listen("tcp", servAddr)
	if err != nil {
		log.Print("Failed to init listener:", err)
		return "", err
	}
	log.Print("Initialized listener:", listener.Addr().String())

	s.listener = listener
	return s.listener.Addr().String(), nil
}

// Launch will register the server for Game service
// and make it serve requests.
func (s *Server) Launch() {
	srv := grpc.NewServer()
	pb.RegisterGameServer(srv, s)
	srv.Serve(s.listener)
}
