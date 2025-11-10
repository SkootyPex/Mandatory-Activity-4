package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	pb "github.com/SkootyPex/mandatory-activity-4/proto"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedMutexServer
	id string
}

func main() {
	id := flag.String("id", "node1", "unique id")
	port := flag.Int("port", 50051, "listening port")
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listeon on %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterMutexServer(s, &server{id: *id})

	log.Printf("%s Listening on: %d", *id, *port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("server err: %v, err")
	}
}
