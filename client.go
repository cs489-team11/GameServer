package server

import (
	"context"
	"fmt"
	"log"

	"github.com/cs489-team11/server/pb"
	"google.golang.org/grpc"
)

// SampleClient is a simple client for testing purposes
type SampleClient struct {
	GameClient pb.GameClient
	Username   username
	UserID     userID
	GameID     gameID
	Config     GameConfig
	Stream     pb.Game_StreamClient
}

func NewSampleClient() *SampleClient {
	return &SampleClient{
		Username: RandUsername(),
	}
}

func (c *SampleClient) Connect(addr string) error {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("Could not connect to server at %s", addr)
	}
	c.GameClient = pb.NewGameClient(conn)
	return nil
}

func (c *SampleClient) ProcessJoinResponse(res *pb.JoinResponse) {
	c.UserID = userID(res.UserId)
	c.GameID = gameID(res.GameId)
	c.Config = NewGameConfig(
		res.Duration, res.PlayerPoints, res.BankPointsPerPlayer,
		res.CreditInterest, res.DepositInterest,
		res.CreditTime, res.DepositTime,
	)
}

func (c *SampleClient) JoinGame() error {
	if c.GameClient == nil {
		return fmt.Errorf("Client is not connected to server")
	}

	req := c.GetJoinRequest()
	res, err := c.GameClient.Join(context.Background(), req)
	log.Printf("Join response: %v", res)
	if err != nil {
		return fmt.Errorf("Failed to join game: %v\n", err)
	}
	c.ProcessJoinResponse(res)
	return nil
}

func (c *SampleClient) OpenStream() error {
	if c.GameClient == nil {
		return fmt.Errorf("Client is not connected to server")
	}

	req := c.GetStreamRequest()
	stream, err := c.GameClient.Stream(context.Background(), req)
	if err != nil {
		return fmt.Errorf("Failed to open stream with server: %v\n", err)
	}
	c.Stream = stream
	log.Printf("Player %v opened stream successfully.\n", c.UserID)
	return nil
}

func (c *SampleClient) StartGame() error {
	if c.GameClient == nil {
		return fmt.Errorf("Client is not connected to server")
	}

	req := c.GetStartRequest()
	_, err := c.GameClient.Start(context.Background(), req)
	if err != nil {
		return fmt.Errorf("Failed to start the game: %v", err)
	}
	log.Printf("The game with id %v has been started by %v.\n", c.GameID, c.UserID)
	return nil
}

func (c *SampleClient) GetJoinRequest() *pb.JoinRequest {
	return &pb.JoinRequest{
		Username: string(c.Username),
	}
}

func (c *SampleClient) GetStreamRequest() *pb.StreamRequest {
	return &pb.StreamRequest{
		UserId: string(c.UserID),
		GameId: string(c.GameID),
	}
}

func (c *SampleClient) GetStartRequest() *pb.StartRequest {
	return &pb.StartRequest{
		GameId: string(c.GameID),
	}
}
