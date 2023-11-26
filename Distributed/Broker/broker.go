package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var kChan = make(chan int)

var worker1 = flag.String("worker1", "127.0.0.1:8040", "ip:port to connect to first worker")
var worker2 = flag.String("worker2", "127.0.0.1:8041", "ip:port to connect to second worker")
var worker3 = flag.String("worker3", "127.0.0.1:8042", "ip:port to connect to third worker")
var worker4 = flag.String("worker4", "127.0.0.1:8043", "ip:port to connect to fourth worker")

type GOLOperations struct {
	Mu      sync.Mutex
	World   [][]byte
	Turns   int
	Paused  bool
	Quit    bool
	Workers []*rpc.Client
}

func calculateAliveCells(world [][]byte, IMHT, IMWD int) []util.Cell {
	var slice []util.Cell
	for y := 0; y < IMHT; y++ {
		for x := 0; x < IMWD; x++ {
			if world[y][x] == 0xFF {
				slice = append(slice, util.Cell{y, x})
			}
		}
	}
	return slice
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

func worker(req stubs.TurnRequest, res *stubs.TurnResponse, client *rpc.Client, responseChannel chan [][]byte) {
	client.Call(stubs.TurnHandler, req, res)
	newSlice := res.Slice
	responseChannel <- newSlice
}

func calculateSlice(IMWD, start, end int, world [][]byte) [][]byte {
	var newSlice [][]byte
	dy := end - start
	for i := 0; i < dy; i++ {
		newSlice = append(newSlice, []byte{})
		for j := 0; j < IMWD; j++ {
			newSlice[i] = append(newSlice[i], world[start+i][j])
		}
	}
	return newSlice
}

func executeTurn(world [][]byte, workers []*rpc.Client, IMWD, IMHT int) [][]byte {
	dy := IMHT / len(workers)
	responseChannels := make([]chan [][]byte, len(workers))
	var newWorld [][]byte
	for i := 0; i < len(workers); i++ {
		request := stubs.TurnRequest{World: world, Slice: calculateSlice(IMWD, i*dy, (i+1)*dy, world), Start: i * dy, End: (i + 1) * dy}
		response := new(stubs.TurnResponse)
		responseChannel := make(chan [][]byte)
		responseChannels[i] = responseChannel
		go worker(request, response, workers[i], responseChannels[i])
	}
	for i := 0; i < len(responseChannels); i++ {
		newSlice := <-responseChannels[i]
		newWorld = append(newWorld, newSlice...)
	}
	return newWorld
}

func (u *GOLOperations) GetAliveCells(req stubs.AliveCellRequest, res *stubs.AliveCellResponse) (err error) {
	u.Mu.Lock()
	defer u.Mu.Unlock()
	res.Cells = calculateAliveCells(u.World, req.ImageHeight, req.ImageWidth)
	res.Turns = u.Turns
	return
}

func (u *GOLOperations) AliveCellCount(req stubs.EmptyRequest, res *stubs.CellCountResponse) (err error) {
	u.Mu.Lock()
	defer u.Mu.Unlock()
	res.TotalCells = totalAliveCells(u.World)
	res.TurnsComplete = u.Turns
	return
}

func (u *GOLOperations) ReturnCurrentState(req stubs.EmptyRequest, res *stubs.StateResponse) (err error) {
	u.Mu.Lock()
	defer u.Mu.Unlock()
	res.World = u.World
	res.Turns = u.Turns
	return
}

func (u *GOLOperations) PauseProcessing(req stubs.EmptyRequest, res *stubs.StateResponse) (err error) {
	u.Mu.Lock()
	defer u.Mu.Unlock()
	if u.Paused == true {
		u.Paused = false
	} else {
		u.Paused = true
	}
	res.World = u.World
	res.Turns = u.Turns
	return
}

func (u *GOLOperations) KillProcesses(req stubs.EmptyRequest, res *stubs.QuitResponse) (err error) {
	u.Mu.Lock()
	defer u.Mu.Lock()
	u.Quit = true
	res.MSG = "HELLO"
	return
}

func (u *GOLOperations) KillServer(req stubs.EmptyRequest, res *stubs.EmptyResponse) (err error) {
	u.Mu.Lock()
	defer u.Mu.Unlock()
	u.Quit = true
	kChan <- 1
	return
}

func (u *GOLOperations) UpdateState(req stubs.StateRequest, res *stubs.StateResponse) (err error) {
	u.Mu.Lock()
	u.Turns = 0
	u.Quit = false
	u.Paused = false
	u.World = req.World
	u.Mu.Unlock()
	for u.Turns < req.Turns && u.Quit != true {
		u.Mu.Lock()
		if u.Paused == false {
			u.World = executeTurn(u.World, u.Workers, req.ImageWidth, req.ImageHeight)
			u.Turns++
		}
		u.Mu.Unlock()
	}
	res.World = u.World
	res.Turns = u.Turns
	return
}

func main() {
	pAddr := flag.String("port", "8038", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	client1, _ := rpc.Dial("tcp", *worker1)
	defer client1.Close()
	client2, _ := rpc.Dial("tcp", *worker2)
	defer client2.Close()
	client3, _ := rpc.Dial("tcp", *worker3)
	defer client3.Close()
	client4, _ := rpc.Dial("tcp", *worker4)
	defer client4.Close()
	clients := []*rpc.Client{client1, client2, client3, client4}

	var world [][]byte
	turns := 0
	paused := false
	quit := false

	listener, _ := net.Listen("tcp", ":"+*pAddr)
	rpc.Register(&GOLOperations{World: world, Turns: turns, Paused: paused, Quit: quit, Workers: clients})
	defer listener.Close()

	go func() {
		<-kChan
		for i := range clients {
			clients[i].Call(stubs.KillWorkerHandler, stubs.EmptyRequest{}, new(stubs.EmptyResponse))
		}
		os.Exit(0)
	}()

	rpc.Accept(listener)
}
