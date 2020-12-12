package tests

import (
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/cs489-team11/server"
	"github.com/cs489-team11/server/pb"
	"github.com/stretchr/testify/require"
)

const testServAddr = "localhost:9090" //"178.128.85.78:9090" //"localhost:9090"//

//const testServAddr = "localhost:0"

func TestJoinAndLeave(t *testing.T) {
	var err error

	client1 := server.NewSampleClient()
	err = client1.Connect(testServAddr)
	require.NoError(t, err)
	client2 := server.NewSampleClient()
	err = client2.Connect(testServAddr)
	require.NoError(t, err)
	client3 := server.NewSampleClient()
	err = client3.Connect(testServAddr)
	require.NoError(t, err)
	t.Log("All clients are connected")

	joinRes1, err := client1.JoinGame()
	require.NoError(t, err)
	joinRes2, err := client2.JoinGame()
	require.NoError(t, err)
	joinRes3, err := client3.JoinGame()
	require.NoError(t, err)
	t.Log("Checkpoint 1")

	err = client2.LeaveGame()
	require.NoError(t, err)
	err = client3.LeaveGame()
	require.NoError(t, err)

	client4 := server.NewSampleClient()
	err = client4.Connect(testServAddr)
	require.NoError(t, err)

	joinRes4, err := client4.JoinGame()
	require.NoError(t, err)

	// check that all of the players are joining the same game
	require.Equal(t, joinRes1.GameId, joinRes2.GameId)
	require.Equal(t, joinRes2.GameId, joinRes3.GameId)
	require.Equal(t, joinRes3.GameId, joinRes4.GameId)

	// TODO: check that there is only
	// client1, client4, and bank returned
	// for Join response of client4.
	// For now, I just checked it manually by looking at logs.

	err = client4.LeaveGame()
	require.NoError(t, err)
	err = client1.LeaveGame()
	require.NoError(t, err)

	// check that leaving again just silently
	// returns no error
	err = client1.LeaveGame()
	require.NoError(t, err)
}

func TestStart(t *testing.T) {
	var err error

	client1 := server.NewSampleClient()
	err = client1.Connect(testServAddr)
	require.NoError(t, err)
	client2 := server.NewSampleClient()
	err = client2.Connect(testServAddr)
	require.NoError(t, err)
	client3 := server.NewSampleClient()
	err = client3.Connect(testServAddr)
	require.NoError(t, err)

	joinRes1, err := client1.JoinGame()
	require.NoError(t, err)
	joinRes2, err := client2.JoinGame()
	require.NoError(t, err)
	require.Equal(t, joinRes1.GameId, joinRes2.GameId)

	err = client2.StartGame()
	require.NoError(t, err)

	joinRes3, err := client3.JoinGame()
	require.NoError(t, err)
	// check that after starting, next user joins into different game.
	// test passes, however, it may happen that client3.JoinGame is handled
	// before client2.StartGame. In that case, the test may fail.
	require.NotEqual(t, joinRes1.GameId, joinRes3.GameId)

	// check that one player can start a game
	err = client3.StartGame()
	require.NoError(t, err)
}

func runTestCreditClientStream(t *testing.T, client *server.SampleClient, debugName string) {
	streamErr := client.OpenStream()
	require.NoError(t, streamErr)
	for i := 1; i < 10000; i++ {
		streamRes, streamErr := client.Stream.Recv()
		if streamErr == io.EOF {
			t.Logf("%s, catched EOF error in stream", debugName)
			break
		}
		require.NoError(t, streamErr)
		switch i {
		case 1:
			require.IsType(
				t, reflect.TypeOf(streamRes.Event), reflect.TypeOf(pb.StreamResponse_Start_{}),
			)
		case 2:
			// check that the second event is credit
			require.IsType(
				t, reflect.TypeOf(streamRes.Event), reflect.TypeOf(pb.StreamResponse_Transaction_{}),
			)
		// next events are returnCredit, useDeposit, and returnDeposit
		// the order is non-deterministic, so it would be difficult to reason here
		// about what happens next
		default:
		}
		t.Logf("%s, stream event: %v\n", debugName, streamRes)
	}
}

func TestCreditAndDeposit(t *testing.T) {
	var err error

	client1 := server.NewSampleClient()
	err = client1.Connect(testServAddr)
	require.NoError(t, err)

	client2 := server.NewSampleClient()
	err = client2.Connect(testServAddr)
	require.NoError(t, err)

	joinRes1, err := client1.JoinGame()
	require.NoError(t, err)
	go runTestCreditClientStream(t, client1, "client1")
	time.Sleep(1 * time.Second) // needed so that 1st player gets event that 2nd joined

	joinRes2, err := client2.JoinGame()
	require.NoError(t, err)
	require.Equal(t, joinRes1.GameId, joinRes2.GameId)
	go runTestCreditClientStream(t, client2, "client2")

	err = client1.StartGame()
	require.NoError(t, err)

	res1, err := client2.TakeCredit(75)
	require.NoError(t, err)
	require.True(t, res1.Success)

	_, err = client2.TakeCredit(-234)
	require.NotNil(t, err)

	res2, err := client1.TakeDeposit(83)
	require.NoError(t, err)
	require.True(t, res2.Success)

	_, err = client1.TakeDeposit(0)
	require.NotNil(t, err)

	// requesting too much money so that the bank cannot
	// grant the credit.
	res3, err := client1.TakeCredit(100000)
	require.NoError(t, err)
	require.False(t, res3.Success)

	// this is needed, since after this goroutine finishes, the stream
	// goroutines will be abruptly finished. so I'm giving it time
	// to process events.
	time.Sleep(2 * time.Second) // sleep time needs to be increased to see theft events
}

func runSimpleTestClientStream(t *testing.T, client *server.SampleClient, debugName string) {
	streamErr := client.OpenStream()
	require.NoError(t, streamErr)
	for i := 1; i < 10000; i++ {
		streamRes, streamErr := client.Stream.Recv()
		if streamErr == io.EOF {
			t.Logf("%s, catched EOF error in stream", debugName)
			break
		}
		require.NoError(t, streamErr)
		switch i {
		case 1:
			require.IsType(
				t, reflect.TypeOf(streamRes.Event), reflect.TypeOf(pb.StreamResponse_Start_{}),
			)
		default:
		}
		t.Logf("%s, stream event: %v\n", debugName, streamRes)
	}
}

func TestLottery(t *testing.T) {
	var err error

	client1 := server.NewSampleClient()
	err = client1.Connect(testServAddr)
	require.NoError(t, err)

	client2 := server.NewSampleClient()
	err = client2.Connect(testServAddr)
	require.NoError(t, err)

	joinRes1, err := client1.JoinGame()
	require.NoError(t, err)
	go runSimpleTestClientStream(t, client1, "client1")
	time.Sleep(1 * time.Second) // needed so that 1st player gets event that 2nd joined

	joinRes2, err := client2.JoinGame()
	require.NoError(t, err)
	require.Equal(t, joinRes1.GameId, joinRes2.GameId)
	go runSimpleTestClientStream(t, client2, "client2")

	err = client1.StartGame()
	require.NoError(t, err)

	_, err = client1.PlayLottery(123)
	require.NotNil(t, err)

	_, err = client2.PlayLottery(0)
	require.NotNil(t, err)

	_, err = client1.PlayLottery(-9)
	require.NotNil(t, err)

	// valid cell index, but calling too early
	res1, err := client2.PlayLottery(4)
	require.NoError(t, err)
	require.False(t, res1.Success)

	time.Sleep(3 * time.Second)

	res2, err := client2.PlayLottery(3)
	require.NoError(t, err)
	require.True(t, res2.Success)
	// expect 9 cell values
	require.Len(t, res2.CellValues, 9)
	// win points >= 0
	require.LessOrEqual(t, int32(0), res2.WinPoints)

	// valid cell index, but calling too early
	res3, err := client2.PlayLottery(1)
	require.NoError(t, err)
	require.False(t, res3.Success)

	time.Sleep(3 * time.Second)

	res4, err := client2.PlayLottery(9)
	require.NoError(t, err)
	require.True(t, res4.Success)
	require.Len(t, res4.CellValues, 9)
	require.LessOrEqual(t, int32(0), res4.WinPoints)

	// this is needed, since after this goroutine finishes, the stream
	// goroutines will be abruptly finished. so I'm giving it time
	// to process events.
	time.Sleep(2 * time.Second) // sleep time needs to be increased to see theft events
}

func TestQuestionGenerateAndAnswer(t *testing.T) {
	var err error

	client1 := server.NewSampleClient()
	err = client1.Connect(testServAddr)
	require.NoError(t, err)

	client2 := server.NewSampleClient()
	err = client2.Connect(testServAddr)
	require.NoError(t, err)

	joinRes1, err := client1.JoinGame()
	require.NoError(t, err)
	go runSimpleTestClientStream(t, client1, "client1")
	time.Sleep(1 * time.Second) // needed so that 1st player gets event that 2nd joined

	joinRes2, err := client2.JoinGame()
	require.NoError(t, err)
	require.Equal(t, joinRes1.GameId, joinRes2.GameId)
	go runSimpleTestClientStream(t, client2, "client2")

	err = client1.StartGame()
	require.NoError(t, err)

	_, err = client1.DoGenerateQuestion(-100)
	require.NotNil(t, err)

	_, err = client2.DoGenerateQuestion(0)
	require.NotNil(t, err)

	res1, err := client1.DoGenerateQuestion(123)
	require.NoError(t, err)

	// 0 is invalid user answer - user answer has to be from 1 to 4
	_, err = client1.DoAnswerQuestion(res1.QuestionId, 0)
	require.NotNil(t, err)

	res2, err := client1.DoAnswerQuestion(res1.QuestionId, 2)
	require.NoError(t, err)
	require.LessOrEqual(t, int32(0), res2.WinPoints)

	// trying to answer the question from different client
	_, err = client2.DoAnswerQuestion(res1.QuestionId, 1)
	require.NotNil(t, err)

	// requesting more money than player has
	_, err = client2.DoGenerateQuestion(100000)
	require.NotNil(t, err)

	res3, err := client1.DoGenerateQuestion(20)
	require.NoError(t, err)

	res4, err := client1.DoAnswerQuestion(res3.QuestionId, 4)
	require.NoError(t, err)
	require.LessOrEqual(t, int32(0), res4.WinPoints)

	// this is needed, since after this goroutine finishes, the stream
	// goroutines will be abruptly finished. so I'm giving it time
	// to process events.
	time.Sleep(2 * time.Second) // sleep time needs to be increased to see theft events
}
