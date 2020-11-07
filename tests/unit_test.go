package tests

import (
	"io"
	"testing"
	"time"

	"github.com/cs489-team11/server"
	"github.com/stretchr/testify/require"
)

var testConfig = server.NewGameConfig(300, 200, 400, 30, 20, 15, 15)

func TestJoinAndLeave(t *testing.T) {
	s := server.NewServer(testConfig)
	serverAddr, err := s.Listen("localhost:0")
	require.NoError(t, err)
	go func() {
		s.Launch()
		t.Log("Server stopped.")
	}()

	client1 := server.NewSampleClient()
	err = client1.Connect(serverAddr)
	require.NoError(t, err)
	client2 := server.NewSampleClient()
	err = client2.Connect(serverAddr)
	require.NoError(t, err)
	client3 := server.NewSampleClient()
	err = client3.Connect(serverAddr)
	require.NoError(t, err)

	joinRes1, err := client1.JoinGame()
	require.NoError(t, err)
	joinRes2, err := client2.JoinGame()
	require.NoError(t, err)
	joinRes3, err := client3.JoinGame()
	require.NoError(t, err)

	err = client2.LeaveGame()
	require.NoError(t, err)
	err = client3.LeaveGame()
	require.NoError(t, err)

	client4 := server.NewSampleClient()
	err = client4.Connect(serverAddr)
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
	s := server.NewServer(testConfig)
	serverAddr, err := s.Listen("localhost:0")
	require.NoError(t, err)
	go func() {
		s.Launch()
		t.Log("Server stopped.")
	}()

	client1 := server.NewSampleClient()
	err = client1.Connect(serverAddr)
	require.NoError(t, err)
	client2 := server.NewSampleClient()
	err = client2.Connect(serverAddr)
	require.NoError(t, err)
	client3 := server.NewSampleClient()
	err = client3.Connect(serverAddr)
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
}

func TestCredit(t *testing.T) {
	s := server.NewServer(testConfig)
	serverAddr, err := s.Listen("localhost:0")
	require.NoError(t, err)
	go func() {
		s.Launch()
		t.Log("Server stopped.")
	}()

	client := server.NewSampleClient()
	err = client.Connect(serverAddr)
	require.NoError(t, err)

	_, err = client.JoinGame()
	require.NoError(t, err)

	go func() {
		streamErr := client.OpenStream()
		require.NoError(t, streamErr)
		for {
			streamRes, streamErr := client.Stream.Recv()
			if streamErr == io.EOF {
				t.Log("Catched EOF error in stream")
				break
			}
			require.NoError(t, streamErr)
			t.Logf("Stream event: %v", streamRes)
		}
	}()

	err = client.StartGame()
	require.NoError(t, err)

	// this is needed, since after start is handled, the stream
	// goroutine will be abruptly finished. so I'm giving it time
	// to process events.
	time.Sleep(2 * time.Second)

	// TODO: finish it. I think I'll need
	// a better way of checking which events are
	// arriving at stream.
}
