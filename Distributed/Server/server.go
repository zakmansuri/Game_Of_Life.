package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var Kchan = make(chan bool)

type GOLOperations struct {
}

func (g *GOLOperations) KillServer(req stubs.KillRequest, res *stubs.KillResponse) (err error) {
	Kchan <- true
	return
}

func (g *GOLOperations) CalculateNextState(req stubs.WorkerRequest, res *stubs.WorkerResponse) (err error) {
	IMHT := len(req.Slice)
	IMWD := len(req.Slice[0])
	newSlice := make([][]byte, len(req.Slice)-2)
	for i := range newSlice {
		newSlice[i] = make([]byte, len(req.Slice[i]))
	}
	for y := 1; y < IMHT-1; y++ {
		for x := 0; x < IMWD; x++ {
			sum := (int(req.Slice[(y+IMHT-1)%IMHT][(x+IMWD-1)%IMWD]) +
				int(req.Slice[(y+IMHT-1)%IMHT][(x+IMWD)%IMWD]) +
				int(req.Slice[(y+IMHT-1)%IMHT][(x+IMWD+1)%IMWD]) +
				int(req.Slice[(y+IMHT)%IMHT][(x+IMWD-1)%IMWD]) +
				int(req.Slice[(y+IMHT)%IMHT][(x+IMWD+1)%IMWD]) +
				int(req.Slice[(y+IMHT+1)%IMHT][(x+IMWD-1)%IMWD]) +
				int(req.Slice[(y+IMHT+1)%IMHT][(x+IMWD)%IMWD]) +
				int(req.Slice[(y+IMHT+1)%IMHT][(x+IMWD+1)%IMWD])) / 255

			if req.Slice[y][x] == 0xFF {
				if sum < 2 {
					newSlice[y-1][x] = 0x00
				} else if sum == 2 || sum == 3 {
					newSlice[y-1][x] = 0xFF
				} else {
					newSlice[y-1][x] = 0x00
				}
			} else {
				if sum == 3 {
					newSlice[y-1][x] = 0xFF
				} else {
					newSlice[y-1][x] = 0x00
				}
			}
		}
	}
	res.Slice = newSlice
	return
}

func main() {
	pAddr := flag.String("port", "8050", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&GOLOperations{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	go rpc.Accept(listener)
	<-Kchan
	os.Exit(0)
}
