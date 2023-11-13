package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

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

type GOLOperations struct{}

func (u *GOLOperations) UpdateState(req stubs.StateRequest, res *stubs.StateResponse) (err error) {
	turn := 0
	for turn < req.Turns {
		req.World = calculateNextState(req.World, req.ImageHeight, req.ImageWidth)
		turn++
	}
	res.World = req.World
	return
}

func (u *GOLOperations) GetAliveCells(req stubs.StateRequest, res *stubs.CellCountResponse) (err error) {
	res.Cells = calculateAliveCells(req.World, req.ImageHeight, req.ImageWidth)
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
