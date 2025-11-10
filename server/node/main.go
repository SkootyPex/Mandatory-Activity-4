package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	pb "github.com/SkootyPex/mandatory-activity-4/proto"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedMutexServer
	id    string
	clock *LamportClock
}

func (s *server) RequestAccess(ctx context.Context, req *pb.Request) (*pb.Reply, error) {
	s.clock.Receive(req.Timestamp)
	log.Printf("%s got a request from %s Timestamp: %d", s.id, req.From, req.Timestamp)
	return &pb.Reply{From: s.id, Ok: true}, nil //always grant it for now, temp
}

func main() {
	id := flag.String("id", "node1", "unique id")
	port := flag.Int("port", 50051, "listening port")
	peer := flag.String("peer", "", "optional address (host:port)")
	flag.Parse()

	clock := &LamportClock{}

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
		if err != nil {
			log.Fatalf("Failed to listeon on %v", err)
		}
		s := grpc.NewServer()
		pb.RegisterMutexServer(s, &server{id: *id, clock: clock})

		log.Printf("%s Listening on: %d", *id, *port)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("server err: %v, err")
		}
	}()

	time.Sleep(500 * time.Millisecond)

	if *peer != "" {
		conn, err := grpc.Dial(*peer, grpc.WithInsecure())
		if err != nil {
			log.Fatalf("could not connect to peer %s: %v", *peer, err)
		}
		defer conn.Close()
		client := pb.NewMutexClient(conn)
		clock.Tick()
		ts := clock.Time()

		log.Printf("%s (%d) is sending requestAccess to %s", *id, ts, *peer)
		resp, err := client.RequestAccess(context.Background(),
			&pb.Request{From: *id, Timestamp: ts})
		if err != nil {
			log.Fatalf("error %v", err)
		}
		log.Printf("%s (%d) got a reply from %s, %v", *id, clock.Time(), resp.From, resp.Ok)
	}
	select {}
}
