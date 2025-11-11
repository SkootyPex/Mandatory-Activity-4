# Mandatory-Activity-4
Implementation of Ricart-Argawala

Missing: 
- Mutual Exclusion
- Request Queue

How to run:

In 3 seperate terminals:

Node 1 - Server:

```
go run ./server/node --id=node1 --port=50051
```

Node 2 - Client + Server:

```
go run ./server/node --id=node2 --port=50052 --peers=localhost:50051
```

Node 3 - Client + Server:

```
go run ./server/node --id=node3 --port=50053 --peers=localhost:50051
```
