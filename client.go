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
		res.TheftTime, res.TheftPercentage,
		res.LotteryTime, res.LotteryMaxWin,
	)
}

func (c *SampleClient) JoinGame() (*pb.JoinResponse, error) {
	if c.GameClient == nil {
		return nil, fmt.Errorf("Client is not connected to server")
	}

	req := c.GetJoinRequest()
	res, err := c.GameClient.Join(context.Background(), req)
	log.Printf("Join response: %v\n", res)
	if err != nil {
		return nil, fmt.Errorf("failed to join game: %v", err)
	}
	c.ProcessJoinResponse(res)
	return res, nil
}

func (c *SampleClient) LeaveGame() error {
	if c.GameClient == nil {
		return fmt.Errorf("Client is not connected to server")
	}

	req := c.GetLeaveRequest()
	res, err := c.GameClient.Leave(context.Background(), req)
	log.Printf("Leave response: %v.\n", res)
	if err != nil {
		return fmt.Errorf("failed to leave game: %v", err)
	}
	return nil
}

func (c *SampleClient) OpenStream() error {
	if c.GameClient == nil {
		return fmt.Errorf("Client is not connected to server")
	}

	req := c.GetStreamRequest()
	stream, err := c.GameClient.Stream(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to open stream with server: %v", err)
	}
	c.Stream = stream
	log.Printf("Player %v opened stream successfully.\n", c.UserID)
	return nil
}

func (c *SampleClient) StartGame() error {
	if c.GameClient == nil {
		return fmt.Errorf("client is not connected to server")
	}

	req := c.GetStartRequest()
	_, err := c.GameClient.Start(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to start the game: %v", err)
	}
	log.Printf("The game with id %v has been started by %v.\n", c.GameID, c.UserID)
	return nil
}

func (c *SampleClient) TakeCredit(val int32) (*pb.CreditResponse, error) {
	if c.GameClient == nil {
		return nil, fmt.Errorf("client is not connected to server")
	}

	req := c.GetCreditRequest(val)
	res, err := c.GameClient.Credit(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to take credit: %v", err)
	}
	log.Printf(
		"user %v, credit amount: %v, success: %v, explanation: %v\n",
		c.UserID, val, res.Success, res.Explanation,
	)
	return res, nil
}

func (c *SampleClient) TakeDeposit(val int32) (*pb.DepositResponse, error) {
	if c.GameClient == nil {
		return nil, fmt.Errorf("client is not connected to server")
	}

	req := c.GetDepositRequest(val)
	res, err := c.GameClient.Deposit(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to take deposit: %v", err)
	}
	log.Printf(
		"user %v, deposit amount: %v, success: %v, explanation: %v\n",
		c.UserID, val, res.Success, res.Explanation,
	)
	return res, nil
}

func (c *SampleClient) PlayLottery(cellIndex int32) (*pb.LotteryResponse, error) {
	if c.GameClient == nil {
		return nil, fmt.Errorf("client is not connected to server")
	}

	req := c.GetLotteryRequest(cellIndex)
	res, err := c.GameClient.Lottery(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to play lottery: %v", err)
	}
	log.Printf(
		"user %v, cell index: %v, success: %v, cell values: %v, win points: %v\n",
		c.UserID, cellIndex, res.Success, res.CellValues, res.WinPoints,
	)
	return res, nil
}

func (c *SampleClient) GetJoinRequest() *pb.JoinRequest {
	return &pb.JoinRequest{
		Username: string(c.Username),
	}
}

func (c *SampleClient) GetLeaveRequest() *pb.LeaveRequest {
	return &pb.LeaveRequest{
		UserId: string(c.UserID),
		GameId: string(c.GameID),
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

func (c *SampleClient) GetCreditRequest(val int32) *pb.CreditRequest {
	return &pb.CreditRequest{
		UserId: string(c.UserID),
		GameId: string(c.GameID),
		Value:  val,
	}
}

func (c *SampleClient) GetDepositRequest(val int32) *pb.DepositRequest {
	return &pb.DepositRequest{
		UserId: string(c.UserID),
		GameId: string(c.GameID),
		Value:  val,
	}
}

func (c *SampleClient) GetLotteryRequest(cellIndex int32) *pb.LotteryRequest {
	return &pb.LotteryRequest{
		UserId:    string(c.UserID),
		GameId:    string(c.GameID),
		CellIndex: cellIndex,
	}
}
