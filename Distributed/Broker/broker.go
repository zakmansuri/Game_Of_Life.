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

func worker(req stubs.WorkerRequest, res *stubs.WorkerResponse, client *rpc.Client, channel chan [][]byte) {
	client.Call(stubs.CalculateNextStateHandler, req, res)
	channel <- res.Slice
}

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

func execute(world [][]byte, IMHT, IMWD int, workers []*rpc.Client) [][]byte {
	dy := calculateHeights(IMHT, len(workers))
	var responseChannels = make([]chan [][]byte, len(workers))
	var newWorld [][]byte
	for i := 0; i < len(workers); i++ {
		//Slice := calculateSlice2(IMWD, i*dy[i], (i+1)*dy[i], world)
		EXPSlice := calculateSlice(IMWD, IMHT, i*dy[i], (i+1)*dy[i], world)
		request := stubs.WorkerRequest{
			Slice: EXPSlice,
			Start: i * dy[i],
			End:   (i + 1) * dy[i],
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
	g.Mu.Lock()
	g.World = req.World
	g.Turns = 0
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
	rpc.Register(&GOLOperations{Workers: workers})
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
