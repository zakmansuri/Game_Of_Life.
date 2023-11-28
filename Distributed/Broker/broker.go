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

var worker1 = flag.String("worker1", "127.0.0.1:8050", "ip:port to connect to first worker")
var worker2 = flag.String("worker2", "127.0.0.1:8051", "ip:port to connect to second worker")
var worker3 = flag.String("worker3", "127.0.0.1:8052", "ip:port to connect to third worker")
var worker4 = flag.String("worker4", "127.0.0.1:8053", "ip:port to connect to fourth worker")

var Kchan = make(chan bool)

type GOLOperations struct {
	Mu      sync.Mutex
	World   [][]byte
	Turns   int
	Quit    bool
	Paused  bool
	Workers []*rpc.Client
	// Fault tolerance
	Continue bool
}

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

func calculateSlice(IMWD, start, end int, world [][]byte) [][]byte {
	var newSlice [][]byte
	dy := end - start
	newSlice = append(newSlice, []byte{})
	if start == 0 {
		for x := 0; x < IMWD; x++ {
			newSlice[0] = append(newSlice[0], world[len(world)-1][x])
		}
	} else {
		for x := 0; x < IMWD; x++ {
			newSlice[0] = append(newSlice[0], world[start-1][x])
		}
	}
	for i := 0; i < dy; i++ {
		newSlice = append(newSlice, []byte{})
		for x := 0; x < IMWD; x++ {
			newSlice[i+1] = append(newSlice[i+1], world[start+i][x])
		}
	}
	newSlice = append(newSlice, []byte{})
	if end == len(world) {
		for x := 0; x < IMWD; x++ {
			newSlice[dy+1] = append(newSlice[dy+1], world[0][x])
		}
	} else {
		for x := 0; x < IMWD; x++ {
			newSlice[dy+1] = append(newSlice[dy+1], world[end][x])
		}
	}
	return newSlice
}

func worker(req stubs.WorkerRequest, res *stubs.WorkerResponse, client *rpc.Client, channel chan [][]byte) {
	client.Call(stubs.CalculateNextStateHandler, req, res)
	channel <- res.Slice
}

func execute(world [][]byte, IMHT, IMWD int, workers []*rpc.Client) [][]byte {
	dy := IMHT / len(workers)
	var responseChannels = make([]chan [][]byte, len(workers))
	var newWorld [][]byte
	for i := 0; i < len(workers); i++ {
		request := stubs.WorkerRequest{
			Slice: calculateSlice(IMWD, i*dy, (i+1)*dy, world),
			Start: i * dy,
			End:   (i + 1) * dy,
		}
		response := new(stubs.WorkerResponse)
		responseChannel := make(chan [][]byte)
		responseChannels[i] = responseChannel
		go worker(request, response, workers[i], responseChannel)
	}
	for i := 0; i < len(workers); i++ {
		slice := <-responseChannels[i]
		newWorld = append(newWorld, slice...)
	}
	return newWorld
}

func (g *GOLOperations) KillServer(req stubs.KillRequest, res *stubs.KillResponse) (err error) {
	Kchan <- true
	return
}

// QuitBroker Fault tolerance
func (g *GOLOperations) QuitBroker(req stubs.KillRequest, res *stubs.KillResponse) (err error) {
	g.Mu.Lock()
	defer g.Mu.Unlock()
	g.Continue = true
	return
}

func (g *GOLOperations) PressedKey(req stubs.KeyRequest, res *stubs.KeyResponse) (err error) {
	g.Mu.Lock()
	defer g.Mu.Unlock()
	res.Turns = g.Turns
	res.World = g.World
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
	return
}

func (g *GOLOperations) UpdateState(req stubs.StateRequest, res *stubs.StateResponse) (err error) {
	// Fault tolerance
	g.Mu.Lock()
	if !g.Continue {
		g.World = req.World
		g.Turns = 0
	}
	g.Quit = false
	g.Paused = false
	g.Mu.Unlock()
	for !g.Quit && g.Turns < req.Turns {
		g.Mu.Lock()
		if !g.Paused {
			g.World = execute(g.World, len(g.World), len(g.World[0]), g.Workers)
			g.Turns++
		}
		g.Mu.Unlock()
	}
	if !g.Quit {
		g.Continue = false
	}
	res.World = g.World
	res.Turns = g.Turns
	return
}

func (g *GOLOperations) CalculateTotalCells(req stubs.TotalCellRequest, res *stubs.TotalCellResponse) (err error) {
	g.Mu.Lock()
	defer g.Mu.Unlock()
	res.AliveCells = totalAliveCells(g.World)
	res.Turns = g.Turns
	return
}

func main() {
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
	workers := []*rpc.Client{w1, w2, w3, w4}
	listener, _ := net.Listen("tcp", ":"+*brokerAddr)
	// Fault tolerance
	rpc.Register(&GOLOperations{Workers: workers, Continue: false})
	defer listener.Close()
	go rpc.Accept(listener)
	<-Kchan
	for i := range workers {
		err := workers[i].Call(stubs.KillServerHandler, stubs.KillRequest{}, new(stubs.KillResponse))
		if err != nil {
			log.Fatal("Worker Kill Request Call Error:", err)
		}
	}
	os.Exit(0)
}
