package tests

import (
	"io"
	"testing"
	"time"

	"github.com/cs489-team11/server"
	"github.com/stretchr/testify/require"
)

var testConfig = server.NewGameConfig(300, 200, 400, 30, 20, 15, 15)

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

	err = client.JoinGame()
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
	time.Sleep(2 * time.Second)

	// TODO: finish it. I think I'll need
	// a better way of checking which events are
	// arriving at stream.
}
