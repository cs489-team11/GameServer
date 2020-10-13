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
)

type Server struct {
	listener net.Listener
	mutex    sync.RWMutex
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) SayHello(_ context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{
		Message: "Hello Ethics Project!!!",
	}, nil
}

func (s *Server) SayRepeatHello(req *pb.RepeatHelloRequest, srv pb.Greeter_SayRepeatHelloServer) error {
	for i := 1; i <= 5; i++ {
		srv.Send(&pb.HelloReply{
			Message: fmt.Sprintf("Streaming with Ethics Project %d\n", i),
		})
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

func (s *Server) Listen(serv_addr string) (string, error) {
	listener, err := net.Listen("tcp", serv_addr)
	if err != nil {
		log.Print("Failed to init listener:", err)
		return "", err
	}
	log.Print("Initialized listener:", listener.Addr().String())

	s.listener = listener
	return s.listener.Addr().String(), nil
}

func (s *Server) Start() {
	srv := grpc.NewServer()
	pb.RegisterGreeterServer(srv, s)
	srv.Serve(s.listener)
}
