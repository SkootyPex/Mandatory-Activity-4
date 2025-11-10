# Mandatory-Activity-4
Implementation of Ricart-Argawala

Missing: 
- Mutual Exclusion
- Request Queue
- Lamport Clocks

How to run:

In 3 seperate terminals:

Node 1 - Server:

go run ./cmd/node --id=node1 --port=50051

Node 2 - Client + Server:

go run ./cmd/node --id=node2 --port=50052 --peer=localhost:50051

Node 3 - Client + Server:

go run ./cmd/node --id=node3 --port=50053 --peer=localhost:50051

