package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var kChan = make(chan int)

type WorkerOperations struct {
	Quit bool
}

func (w *WorkerOperations) CalculateNextState(req stubs.TurnRequest, res *stubs.TurnResponse) (err error) {
	IMHT := len(req.World)
	IMWD := len(req.World[0])
	newSlice := make([][]byte, len(req.Slice))
	for i := range newSlice {
		newSlice[i] = make([]byte, len(req.Slice[i]))
	}
	for y := req.Start; y < req.End; y++ {
		for x := 0; x < IMWD; x++ {
			sum := (int(req.World[(y+IMHT-1)%IMHT][(x+IMWD-1)%IMWD]) +
				int(req.World[(y+IMHT-1)%IMHT][(x+IMWD)%IMWD]) +
				int(req.World[(y+IMHT-1)%IMHT][(x+IMWD+1)%IMWD]) +
				int(req.World[(y+IMHT)%IMHT][(x+IMWD-1)%IMWD]) +
				int(req.World[(y+IMHT)%IMHT][(x+IMWD+1)%IMWD]) +
				int(req.World[(y+IMHT+1)%IMHT][(x+IMWD-1)%IMWD]) +
				int(req.World[(y+IMHT+1)%IMHT][(x+IMWD)%IMWD]) +
				int(req.World[(y+IMHT+1)%IMHT][(x+IMWD+1)%IMWD])) / 255

			if req.World[y][x] == 0xFF {
				if sum < 2 {
					newSlice[y-req.Start][x] = 0x00
				} else if sum == 2 || sum == 3 {
					newSlice[y-req.Start][x] = 0xFF
				} else {
					newSlice[y-req.Start][x] = 0x00
				}
			} else {
				if sum == 3 {
					newSlice[y-req.Start][x] = 0xFF
				} else {
					newSlice[y-req.Start][x] = 0x00
				}
			}
		}
	}
	res.Slice = newSlice
	return
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

//func (u *GOLOperations) UpdateState(req stubs.StateRequest, res *stubs.StateResponse) (err error) {
//	u.Mu.Lock()
//	u.Turns = 0
//	u.Quit = false
//	u.World = make([][]byte, req.ImageHeight)
//	for y := 0; y < req.ImageHeight; y++ {
//		u.World[y] = make([]byte, req.ImageWidth)
//	}
//	for y := 0; y < req.ImageHeight; y++ {
//		for x := 0; x < req.ImageWidth; x++ {
//			u.World[y][x] = req.World[y][x]
//		}
//	}
//	u.Mu.Unlock()
//	for u.Turns < req.Turns && u.Quit != true {
//		u.Mu.Lock()
//		fmt.Println(u.Turns, req.ImageHeight)
//		if u.Paused == false {
//			u.World = calculateNextState(u.World, req.ImageHeight, req.ImageWidth)
//			u.Turns++
//		}
//		u.Mu.Unlock()
//	}
//	res.World = u.World
//	res.Turns = u.Turns
//	return
//}

//func (u *GOLOperations) GetAliveCells(req stubs.AliveCellRequest, res *stubs.AliveCellResponse) (err error) {
//	u.Mu.Lock()
//	defer u.Mu.Unlock()
//	res.Cells = calculateAliveCells(u.World, req.ImageHeight, req.ImageWidth)
//	res.Turns = u.Turns
//	return
//}

//func (u *GOLOperations) AliveCellCount(req stubs.EmptyRequest, res *stubs.CellCountResponse) (err error) {
//	u.Mu.Lock()
//	defer u.Mu.Unlock()
//	res.TotalCells = totalAliveCells(u.World)
//	res.TurnsComplete = u.Turns
//	return
//}

//func (u *GOLOperations) ReturnCurrentState(req stubs.EmptyRequest, res *stubs.StateResponse) (err error) {
//	u.Mu.Lock()
//	defer u.Mu.Unlock()
//	res.World = u.World
//	res.Turns = u.Turns
//	return
//}

//func (u *GOLOperations) PauseProcessing(req stubs.EmptyRequest, res *stubs.StateResponse) (err error) {
//	u.Mu.Lock()
//	defer u.Mu.Unlock()
//	if u.Paused == true {
//		u.Paused = false
//	} else {
//		u.Paused = true
//	}
//	res.World = u.World
//	res.Turns = u.Turns
//	return
//}

func (w *WorkerOperations) KillServer(req stubs.EmptyRequest, res *stubs.EmptyResponse) (err error) {
	kChan <- 1
	return
}

//func (w *WorkerOperations) KillProcesses(req stubs.EmptyRequest, res *stubs.EmptyResponse) (err error) {
//	w.Quit = true
//	return
//}

func main() {
	pAddr := flag.String("port", "8040", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	go func() {
		<-kChan
		os.Exit(0)
	}()
	rpc.Register(&WorkerOperations{false})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
