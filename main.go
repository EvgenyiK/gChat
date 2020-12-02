package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/EvgenyiK/gChat/proto"
	"google.golang.org/grpc"
	
)

type Connection struct {
	stream proto.Broadcast_CreateStreamServer
	id     string
	active bool
	error  chan error
}

type Server struct {
	Connection []*Connection
}

//CreateStream Создание соединения и добавление его в список соединений
func (s *Server) CreateStream(pconn *proto.Connect, stream proto.Broadcast_CreateStreamServer) error {
	conn := &Connection{
		stream: stream,
		id:     pconn.User.Id,
		active: true,
		error:  make(chan error),
	}

	s.Connection = append(s.Connection, conn)
	return <-conn.error
}

//BroadcastMessage  отправляет сообщения всем активным пользователям
func (s *Server) BroadcastMessage(ctx context.Context, msg *proto.Message) (*proto.Close, error) {
	wait := sync.WaitGroup{}
	done := make(chan int)

	for _, conn := range s.Connection {
		wait.Add(1)

		go func(msg *proto.Message, conn *Connection) {
			defer wait.Done()

			if conn.active {
				err := conn.stream.Send(msg)
				log.Println("Sending message to: ", conn.stream)
				if err != nil {
					log.Printf("Error with stream %v. Error: %v", conn.stream, err)
					conn.active = false
					conn.error <- err
				}
			}
		}(msg, conn)
	}

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done
	return &proto.Close{}, nil
}

func main() {
	var connections []*Connection

	server := &Server{connections}

	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("error creating the server %v", err)
	}
	fmt.Println("Starting server at port:8080")

	proto.RegisterBroadcastServer(grpcServer, server)
	grpcServer.Serve(listener)
}
