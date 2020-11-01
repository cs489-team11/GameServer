package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

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
	mutex       sync.RWMutex
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

// Join adds a player to the game.
func (s *Server) Join(_ context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	reqUsername := username(req.GetUsername())
	userID := s.waitingGame.addPlayer(reqUsername)

	res := s.getJoinResponseMessage(userID, s.waitingGame)
	return res, nil
}

// Leave deleted player from the waiting game.
func (s *Server) Leave(_ context.Context, req *pb.LeaveRequest) (*pb.LeaveResponse, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	reqGameID := gameID(req.GetGameId())
	reqUserID := userID(req.GetUserId())

	if s.waitingGame.gameID != reqGameID {
		err := fmt.Errorf(
			"game with id %v doesn't exist or has been already started (can't join active game)",
			reqGameID,
		)
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	s.waitingGame.deletePlayer(reqUserID)
	return &pb.LeaveResponse{}, nil
}

// Start will start the game, i.e. change it from "waiting" to "active".
// New waiting game will be created for other users to join.
func (s *Server) Start(_ context.Context, req *pb.StartRequest) (*pb.StartResponse, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	reqGameID := gameID(req.GetGameId())

	if s.waitingGame.gameID != reqGameID {
		log.Printf(
			"attempt to start game with id different from waiting game id; have: %v, want %v\n",
			reqGameID,
			s.waitingGame.gameID,
		)
		// ignore the error
		return &pb.StartResponse{}, nil
	}

	game := s.waitingGame
	game.start()
	s.activeGames[game.gameID] = game
	// count down until game finishes
	time.AfterFunc(time.Duration(game.config.duration)*time.Second, func() {
		game.finish()
	})

	// create a new waiting game
	s.waitingGame = newGame(s.gameConfig)

	return &pb.StartResponse{}, nil
}

// Credit will check if the credit can be granted. It will return "True" for success, if
// credit has been granted. If "success == False", "explanation" will contain the relevant
// explanation about why it hasn't been granted.
// Requesting client has to make sure that provided game_id and user_id are vaild.
func (s *Server) Credit(_ context.Context, req *pb.CreditRequest) (*pb.CreditResponse, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	reqGameID := gameID(req.GetGameId())
	reqUserID := userID(req.GetUserId())
	reqVal := req.GetValue()

	game, ok := s.activeGames[reqGameID]
	if !ok {
		err := fmt.Errorf("there is no active game with id %v", reqGameID)
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	if reqVal <= 0 {
		err := fmt.Errorf("requested value has to be positive value (received: %d)", reqVal)
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	success, explanation, err := game.useCredit(reqUserID, reqVal)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	return s.getCreditResponseMessage(success, explanation), nil
}

func (s *Server) Deposit(_ context.Context, req *pb.DepositRequest) (*pb.DepositResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "Unimplemented")
}

// Stream opens the server stream with the user.
func (s *Server) Stream(req *pb.StreamRequest, srv pb.Game_StreamServer) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var game *game = nil
	reqGameID := gameID(req.GetGameId())
	reqUserID := userID(req.GetUserId())

	if reqGameID == s.waitingGame.gameID {
		game = s.waitingGame
	} else if g, ok := s.activeGames[reqGameID]; ok {
		game = g
	}

	if game == nil {
		return status.Errorf(codes.InvalidArgument, "game with id %v doesn't exist or is finished", reqGameID)
	}

	game.setPlayerStream(reqUserID, srv)
	return nil
}

func (s *Server) getJoinResponseMessage(
	userID userID, game *game,
) *pb.JoinResponse {
	game.mutex.RLock()
	defer game.mutex.RUnlock()
	return &pb.JoinResponse{
		UserId:              string(userID),
		GameId:              string(game.gameID),
		Duration:            game.config.duration,
		PlayerPoints:        game.config.playerPoints,
		BankPointsPerPlayer: game.config.bankPointsPerPlayer,
		CreditInterest:      game.config.creditInterest,
		DepositInterest:     game.config.depositInterest,
		CreditTime:          game.config.creditTime,
		DepositTime:         game.config.depositTime,
		Players:             game.getPBPlayersWithBank(),
	}
}

func (s *Server) getCreditResponseMessage(success bool, explanation string) *pb.CreditResponse {
	return &pb.CreditResponse{
		Success:     success,
		Explanation: explanation,
	}
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
