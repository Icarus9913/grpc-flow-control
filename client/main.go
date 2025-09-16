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

// Binary client is an example client.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"grpc-flow-control/proto/echo"
)

var addr = flag.String("addr", "localhost:50052", "the address to connect to")

var payload = string(make([]byte, 8*1024)) // 8KB

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6061", nil))
	}()

	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		//grpc.WithInitialWindowSize(512*1024),
		//grpc.WithInitialConnWindowSize(512*1024),
	)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := echo.NewEchoClient(conn)

	stream, err := c.BidirectionalStreamingEcho(ctx)
	if err != nil {
		log.Fatalf("Error creating stream: %v", err)
	}
	log.Printf("New stream began.")

	// First we will send data on the stream until we cannot send any more.  We
	// detect this by not seeing a message sent 1s after the last sent message.
	for i := 0; i < 100; i++ {
		if err := stream.Send(&echo.EchoRequest{Message: payload}); err != nil {
			log.Fatalf("Error sending data: %v", err)
		}
		log.Printf("Sent %v messages.", i)
		//stream.CloseSend()
	}

	// Next, we wait 2 seconds before reading from the stream, to give the
	// server an opportunity to block while sending its responses.
	//time.Sleep(2 * time.Second)

	// Finally, read all the data sent by the server to allow it to unblock.
	//for i := 0; true; i++ {
	//	if _, err := stream.Recv(); err != nil {
	//		log.Printf("Read %v messages.", i)
	//		if err == io.EOF {
	//			log.Printf("Stream ended successfully.")
	//			return
	//		}
	//		log.Fatalf("Error receiving data: %v", err)
	//	}
	//}
}
