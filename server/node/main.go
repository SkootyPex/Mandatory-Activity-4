package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	pb "github.com/SkootyPex/mandatory-activity-4/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	deferTs := s.clock.Time()

	shouldDefer := s.requestingCS &&
		(s.requestTime < req.Timestamp ||
			(s.requestTime == req.Timestamp && s.id < req.From))

	if shouldDefer {
		s.deferred[req.From] = true
		log.Printf("[%s at %d] deferring reply to %s (timestamp = %d)", s.id, deferTs, req.From, req.Timestamp)
		return &pb.Reply{From: s.id, Ok: false}, nil
	}

	log.Printf("[%s at %d] giving reply at %s (timestamp = %d)",
		s.id, s.clock.Time(), req.From, req.Timestamp)
	return &pb.Reply{From: s.id, Ok: true}, nil
}

func (s *server) waitForPeers() {
	for {
		allUp := true
		for _, peer := range s.peers {
			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			conn, err := grpc.DialContext(
				ctx,
				peer,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			)

			if err != nil {
				allUp = false
				log.Printf("[%s] waiting for %s.", s.id, peer)
				continue
			}
			conn.Close()
		}
		if allUp {
			log.Printf("[%s] all peers reachable", s.id)
			return
		}
		time.Sleep(1 * time.Second)
	}
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

func generatePeers(total, self int) []string {
	var peers []string
	for i := 1; i <= total; i++ {
		if i == self {
			continue
		}
		peers = append(peers, fmt.Sprintf("localhost:%d", 50050+i))
	}
	return peers
}

func main() {
	nodeNum := flag.Int("node", 1, "node number (1..N)")
	totalNodes := flag.Int("total", 3, "total number of nodes")
	flag.Parse()

	id := fmt.Sprintf("node%d", *nodeNum)
	port := 50050 + *nodeNum
	logFile, _ := os.Create(fmt.Sprintf("%s.log", id))
	log.SetOutput(logFile)

	clock := &LamportClock{}
	s := &server{
		id:       id,
		clock:    clock,
		deferred: make(map[string]bool),
		peers:    generatePeers(*totalNodes, *nodeNum),
	}

	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			log.Fatalf("listening err: %v", err)
		}
		grpcServer := grpc.NewServer()
		pb.RegisterMutexServer(grpcServer, s)
		log.Printf("[%s] listening on :%d, peers: %s", id, port, strings.Join(s.peers, ", "))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	s.waitForPeers()

	go func() {
		for {
			delay := time.Duration(3+rand.Intn(5)) * time.Second
			time.Sleep(delay)
			s.enterCriticalSection()
		}
	}()

	select {}
}
