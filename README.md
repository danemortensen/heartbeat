# heartbeat
Heartbeat protocol for membership status
Written in GO.

## AUTHOR: Dane Mortensen, Kartik Mendiratta, Gilbert Han

# Specification
Each node will send its heartbeat every 2 seconds, sends the table each second and fails every 8 seconds.

We assigned each node's left and right neighbors to send its heartbeat, and the simulation won't start until it has at least 3 nodes in total.

# Simulate Gossip
To build the go file, run `./make.sh`
   - This builds `heartbeater.go` and `demo/heartbeat.go` files

To start the master node, run `./master.sh`
   - It will start a server at `localhost:3000`

To start the worker, run `./worker.sh <PORT NUMBER>`
   - This will start a worker server at the specified port number and connect to master at `port 3000`

## TO CREATE MULTIPLE WORKERS, YOU MUST HAVE MULTIPLE COMMAND LINE OPEN
