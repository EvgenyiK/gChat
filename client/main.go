package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/EvgenyiK/gChat/proto"
	"google.golang.org/grpc"
)

var client proto.BroadcastClient
var wait *sync.WaitGroup

func init() {
	wait = &sync.WaitGroup{}
}

//Подключение к потоковому серверу при помощи CreateStream  
func connect(user *proto.User) error {
	var streamerror error

	stream,err:= client.CreateStream(context.Background(), &proto.Connect{
		User: user,
		Active: true,
	})

	if err != nil {
		return fmt.Errorf("connection failed: %v", err)
	}

// Создание горутины которая получает сообщение	
	wait.Add(1)
	go func(str proto.Broadcast_CreateStreamClient) {
		defer wait.Done()

		for{
			msg,err:= str.Recv()
			if err != nil {
				streamerror = fmt.Errorf("Error reading message: %v", err)
				break
			}

			fmt.Printf("%v : %s/n", msg.Id, msg.Message)
		}
	}(stream)

	return streamerror
}

func main(){
	timestamp:= time.Now()
	done:= make(chan int)

	//Получаем имя из коммандной строки
	name:= flag.String("N", "Anonimus", "The name of user")
	flag.Parse()

	id:= sha256.Sum256([]byte(timestamp.String()+ *name))

	conn,err:= grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Couldnt connect to service: %v", err)
	}

	client = proto.NewBroadcastClient(conn)
	user:= &proto.User{
		Id: hex.EncodeToString(id[:]),
		Name: *name,
	}

	//Подключаемся к серверу
	connect(user)

	//Другая горутина для отправки асинхронного сообщения
	wait.Add(1)
	go func ()  {
		defer wait.Done()
		scanner:= bufio.NewScanner(os.Stdin)
		for scanner.Scan(){
			msg:= &proto.Message{
				Id: user.Id,
				Message: scanner.Text(),
				Timestamp: timestamp.String(),
			}

			_,err:= client.BroadcastMessage(context.Background(), msg)
			if err != nil {
				fmt.Printf("Error Sending Message: %v", err)
				break
			}
		}
	}()

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done
}