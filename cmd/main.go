package main

import (
	"log"

	"github.com/cs489-team11/server"
)

func main() {
	s := server.NewServer()
	if _, err := s.Listen("0.0.0.0:9090"); err != nil {
		log.Fatalf("Server failed to listen: %v", err)
	}
	s.Launch()
}
