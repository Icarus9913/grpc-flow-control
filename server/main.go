/*
 *
 * Copyright 2023 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Binary server is an example server.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"time"

	"google.golang.org/grpc"

	"grpc-flow-control/proto/echo"
)

var port = flag.Int("port", 50052, "port number")

var payload = string(make([]byte, 8*1024)) // 8KB

// server is used to implement EchoServer.
type server struct {
	echo.UnimplementedEchoServer
}

func (s *server) BidirectionalStreamingEcho(stream echo.Echo_BidirectionalStreamingEchoServer) error {
	log.Printf("New stream began.")
	// First, we wait 2 seconds before reading from the stream, to give the
	// client an opportunity to block while sending its requests.
	time.Sleep(2 * time.Second)

	for i := 0; true; i++ {
		if _, err := stream.Recv(); err != nil {
			if err == io.EOF {
				log.Printf("Recv EOF")
				return nil
			}
			log.Printf("Error receiving data: %v", err)
			return err
		}
		log.Printf("Read %v messages.", i)

		if err := stream.Send(&echo.EchoResponse{Message: payload}); err != nil {
			log.Printf("Error sending data: %v", err)
			return err
		}
		log.Printf("Sent %v messages.", i)
	}

	return nil
}

func (s *server) BidirectionalStreamingEcho1(stream echo.Echo_BidirectionalStreamingEchoServer) error {
	log.Printf("New stream began.")
	// First, we wait 2 seconds before reading from the stream, to give the
	// client an opportunity to block while sending its requests.
	time.Sleep(2 * time.Second)

	// Next, read all the data sent by the client to allow it to unblock.
	errChReceive := make(chan error, 1)
	errChSend := make(chan error, 1)

	go func() {
		for i := 0; true; i++ {
			if _, err := stream.Recv(); err != nil {
				if err == io.EOF {
					log.Printf("Recv EOF")
					return
				}
				log.Printf("Error receiving data: %v", err)
				errChReceive <- err
			}

			log.Printf("Read %v messages.", i)
			time.Sleep(time.Second * 3)
		}
	}()

	go func() {
		for i := 0; true; i++ {
			if err := stream.Send(&echo.EchoResponse{Message: payload}); err != nil {
				log.Printf("Error sending data: %v", err)
				errChSend <- err
			}
			log.Printf("Sent %v messages.", i)
		}
	}()

	select {
	case err := <-errChReceive:
		if err != nil {
			return err
		}
	case err := <-errChSend:
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	flag.Parse()

	address := fmt.Sprintf(":%v", *port)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	grpcServer := grpc.NewServer(
	//grpc.InitialWindowSize(512*1024),
	//grpc.InitialConnWindowSize(512*1024),
	)
	echo.RegisterEchoServer(grpcServer, &server{})

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
