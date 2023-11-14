package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type GOLOperations struct {
	Mu    sync.Mutex
	World [][]byte
	Turns int
}

func calculateNextState(world [][]byte, IMHT, IMWD int) [][]byte {
	newWorld := make([][]byte, IMHT)
	for i := range newWorld {
		newWorld[i] = make([]byte, IMWD)
	}

	for y := 0; y < IMHT; y++ {
		for x := 0; x < IMWD; x++ {
			sum := (int(world[(y+IMHT-1)%IMHT][(x+IMWD-1)%IMWD]) +
				int(world[(y+IMHT-1)%IMHT][(x+IMWD)%IMWD]) +
				int(world[(y+IMHT-1)%IMHT][(x+IMWD+1)%IMWD]) +
				int(world[(y+IMHT)%IMHT][(x+IMWD-1)%IMWD]) +
				int(world[(y+IMHT)%IMHT][(x+IMWD+1)%IMWD]) +
				int(world[(y+IMHT+1)%IMHT][(x+IMWD-1)%IMWD]) +
				int(world[(y+IMHT+1)%IMHT][(x+IMWD)%IMWD]) +
				int(world[(y+IMHT+1)%IMHT][(x+IMWD+1)%IMWD])) / 255

			if world[y][x] == 0xFF {
				if sum < 2 {
					newWorld[y][x] = 0x00
				} else if sum == 2 || sum == 3 {
					newWorld[y][x] = 0xFF
				} else {
					newWorld[y][x] = 0x00
				}
			} else {
				if sum == 3 {
					newWorld[y][x] = 0xFF
				} else {
					newWorld[y][x] = 0x00
				}
			}
		}
	}

	return newWorld
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

func (u *GOLOperations) UpdateState(req stubs.StateRequest, res *stubs.StateResponse) (err error) {
	u.Turns = 0
	u.World = req.World
	for u.Turns < req.Turns {
		u.Mu.Lock()
		//fmt.Print("State Locked\n")
		u.World = calculateNextState(u.World, req.ImageHeight, req.ImageWidth)
		u.Turns++
		u.Mu.Unlock()
	}
	res.World = u.World
	return
}

func (u *GOLOperations) GetAliveCells(req stubs.AliveCellRequest, res *stubs.AliveCellResponse) (err error) {
	u.Mu.Lock()
	defer u.Mu.Unlock()
	res.Cells = calculateAliveCells(u.World, req.ImageHeight, req.ImageWidth)
	return
}

func (u *GOLOperations) AliveCellCount(req stubs.EmptyRequest, res *stubs.CellCountResponse) (err error) {
	u.Mu.Lock()
	defer u.Mu.Unlock()
	res.TotalCells = totalAliveCells(u.World)
	res.TurnsComplete = u.Turns
	return
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&GOLOperations{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
