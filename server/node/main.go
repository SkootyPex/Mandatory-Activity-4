package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	pb "github.com/SkootyPex/mandatory-activity-4/proto"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedMutexServer
	id           string
	clock        *LamportClock
	mu           sync.Mutex
	requestingCS bool
	requestTime  int64
	deferred     map[string]bool
	peers        []string
}

func (s *server) RequestAccess(ctx context.Context, req *pb.Request) (*pb.Reply, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.clock.Receive(req.Timestamp)
	shouldDefer := false

	if s.requestingCS {
		if s.requestTime < req.Timestamp || (s.requestTime == req.Timestamp && s.id < req.From) {
			shouldDefer = true
		}
	}

	if shouldDefer {
		s.deferred[req.From] = true
		log.Printf("[%s at %d] Deferring reply to %s (timestamp = %d)",
			s.id, s.clock.Time(), req.From, req.Timestamp)
		return &pb.Reply{From: s.id, Ok: false}, nil
	}

	log.Printf("[%s at %d] giving reply at %s (timestamp = %d)",
		s.id, s.clock.Time(), req.From, req.Timestamp)
	return &pb.Reply{From: s.id, Ok: true}, nil
}

func (s *server) enterCriticalSection() {
	s.mu.Lock()
	s.requestingCS = true
	s.clock.Tick()
	s.requestTime = s.clock.Time()
	s.mu.Unlock()

	log.Printf("[%s at %d] requesting to enter critical section", s.id, s.requestTime)

	okCount := 0

	for _, peer := range s.peers {
		conn, err := grpc.Dial(peer, grpc.WithInsecure())
		if err != nil {
			log.Printf("[%s] could not reach %s, %v", s.id, peer, err)
			continue
		}
		client := pb.NewMutexClient(conn)
		resp, err := client.RequestAccess(context.Background(),
			&pb.Request{From: s.id, Timestamp: s.requestTime})
		conn.Close()

		if err == nil && resp.Ok {
			okCount++
		}
	}

	if okCount == len(s.peers) {
		log.Printf("[%s] Entering Critical Section", s.id)
		time.Sleep(3 * time.Second)
		log.Printf("[%s] leaving Critical Section", s.id)
	}

	s.mu.Lock()
	s.requestingCS = false
	for peer := range s.deferred {
		log.Printf("[%s] sending deferred reply to %s", s.id, peer)
		delete(s.deferred, peer)
	}
	s.mu.Unlock()
}

func main() {
	id := flag.String("id", "node1", "unique id")
	port := flag.Int("port", 50051, "listening port")
	peersFlag := flag.String("peers", "", "seperated list of addresses")
	flag.Parse()

	clock := &LamportClock{}
	s := &server{
		id:       *id,
		clock:    clock,
		deferred: make(map[string]bool),
	}

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
		if err != nil {
			log.Fatalf("listening err: %v", err)
		}
		grpcServer := grpc.NewServer()
		pb.RegisterMutexServer(grpcServer, s)
		log.Printf("[%s] listening on %d", *id, *port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	if *peersFlag != "" {
		for _, p := range strings.Split(*peersFlag, ",") {
			s.peers = append(s.peers, strings.TrimSpace(p))
		}
	}

	time.Sleep(1 * time.Second)

	if len(s.peers) > 0 {
		time.Sleep(2 * time.Second)
		s.enterCriticalSection()
	}

	select {}
}
