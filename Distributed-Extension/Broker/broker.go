package main

import (
	"flag"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

//4 workers
var worker1 = flag.String("worker1", "127.0.0.1:8040", "ip:port to connect to first worker")
var worker2 = flag.String("worker2", "127.0.0.1:8041", "ip:port to connect to second worker")
var worker3 = flag.String("worker3", "127.0.0.1:8042", "ip:port to connect to third worker")
var worker4 = flag.String("worker4", "127.0.0.1:8043", "ip:port to connect to fourth worker")

//channel used to kill broker
var Kchan = make(chan bool)

//struct to hold exported type variables
type GOLOperations struct {
	Mu      sync.Mutex
	World   [][]byte
	Turns   int
	Quit    bool
	Paused  bool
	Workers []*rpc.Client
}

//helper function to calculate total number of alive cells
func totalAliveCells(w [][]byte) int {
	count := 0
	for i := 0; i < len(w); i++ {
		for j := 0; j < len(w[0]); j++ {
			if w[i][j] == 255 {
				count++
			}
		}
	}
	return count
}

//calculates the slice required by the worker
func calculateSlice(IMWD, IMHT, start, end int, world [][]byte) [][]byte {
	var newSlice [][]byte
	dy := end - start
	for y := 0; y < dy+2; y++ {
		newSlice = append(newSlice, []byte{})
		r := start + y - 1
		if r < 0 {
			r += IMHT
		} else if r >= IMHT {
			r -= IMHT
		}
		for x := 0; x < IMWD; x++ {
			newSlice[y] = append(newSlice[y], world[r][x])
		}
	}
	return newSlice
}

//function to call specified worker and retrieve updated slice
func worker(req stubs.WorkerRequest, res *stubs.WorkerResponse, client *rpc.Client, channel chan [][]byte) {
	client.Call(stubs.CalculateNextStateHandler, req, res)
	channel <- res.Slice
}

//function to calculate height of each slice
func calculateHeights(IMHT, N int) []int {
	dy := make([]int, N)
	r := IMHT % N
	x := IMHT / N
	for i := 0; i < N; i++ {
		dy[i] = x
	}
	dy[0] = dy[0] + r
	return dy
}

//executes each worker using goroutine and retrieves slices to reconstruct next state
func step(world [][]byte, IMHT, IMWD int, workers []*rpc.Client) [][]byte {
	dy := calculateHeights(IMHT, len(workers))
	responseChannels := make([]chan [][]byte, len(workers))
	var newWorld [][]byte
	//send request to each worker as go routine
	for i := 0; i < len(workers); i++ {
		request := stubs.WorkerRequest{
			Slice: calculateSlice(IMWD, IMHT, i*dy[i], (i+1)*dy[i], world),
			Start: i * dy[i],
			End:   (i + 1) * dy[i],
		}
		response := new(stubs.WorkerResponse)
		responseChannel := make(chan [][]byte)
		responseChannels[i] = responseChannel
		go worker(request, response, workers[i], responseChannel)
	}
	//receive all responses and put world back together
	for i := 0; i < len(workers); i++ {
		slice := <-responseChannels[i]
		newWorld = append(newWorld, slice...)
	}
	return newWorld
}

//exported function to kill the server by passing value down Kchan channel
func (g *GOLOperations) KillServer(req stubs.KillRequest, res *stubs.KillResponse) (err error) {
	Kchan <- true
	return
}

//takes key press and deals with appropriate functions and returns world and turns
func (g *GOLOperations) PressedKey(req stubs.KeyRequest, res *stubs.KeyResponse) (err error) {
	g.Mu.Lock()
	defer g.Mu.Unlock()
	switch req.Key {
	case 'p':
		if g.Paused == false {
			g.Paused = true
		} else {
			g.Paused = false
		}
	case 'q':
		g.Quit = true
	case 'k':
		g.Quit = true
	}
	res.Turns = g.Turns
	res.World = g.World
	return
}

//Exported function to deal with initial call to broker
func (g *GOLOperations) UpdateState(req stubs.StateRequest, res *stubs.StateResponse) (err error) {
	// Fault tolerance
	g.Mu.Lock()
	//only continues if continue flag is true, and if turns match
	if !req.Cont || g.Turns == req.Turns || g.Turns == 0 || req.IMHT != len(g.World) {
		g.World = req.World
		g.Turns = 0
	}
	g.Quit = false
	g.Paused = false
	g.Mu.Unlock()
	for !g.Quit && g.Turns < req.Turns {
		g.Mu.Lock()
		if !g.Paused {
			//calls execute for every turn to calculate new turn
			g.World = step(g.World, len(g.World), len(g.World[0]), g.Workers)
			g.Turns++
		}
		g.Mu.Unlock()
		//unlocks Mutex after every turn to allow other processes to access variables
	}
	res.World = g.World
	res.Turns = g.Turns
	return
}

//exported functions to calculate the number of cells triggered by the ticker
func (g *GOLOperations) CalculateTotalCells(req stubs.TotalCellRequest, res *stubs.TotalCellResponse) (err error) {
	g.Mu.Lock()
	defer g.Mu.Unlock()
	res.AliveCells = totalAliveCells(g.World)
	res.Turns = g.Turns
	return
}

func main() {
	//listens on port 8030 for distributor
	brokerAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	w1, _ := rpc.Dial("tcp", *worker1)
	w2, _ := rpc.Dial("tcp", *worker2)
	w3, _ := rpc.Dial("tcp", *worker3)
	w4, _ := rpc.Dial("tcp", *worker4)
	defer w1.Close()
	defer w2.Close()
	defer w3.Close()
	defer w4.Close()
	//dials into every worker and adds them to the exported type list
	workers := []*rpc.Client{w1, w2, w3, w4}
	listener, _ := net.Listen("tcp", ":"+*brokerAddr)
	// Fault tolerance
	rpc.Register(&GOLOperations{Workers: workers})
	defer listener.Close()
	go rpc.Accept(listener)
	<-Kchan
	//waits for kill channel to be activated at which it kills off every client and then exits with code 0
	for i := range workers {
		err := workers[i].Call(stubs.KillServerHandler, stubs.KillRequest{}, new(stubs.KillResponse))
		if err != nil {
			log.Fatal("Worker Kill Request Call Error:", err)
		}
	}
	os.Exit(0)
}
