package gol

import (
	"flag"
	"fmt"
	"net/rpc"
	"strconv"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	keyPresses <-chan rune
}

func findAliveCells(world [][]byte, IMWD, IMHT int) []util.Cell {

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

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	t := strconv.Itoa(p.ImageWidth)
	t = t + "x" + t
	c.ioCommand <- ioInput
	c.ioFilename <- t
	world := make([][]uint8, p.ImageHeight)
	for i := range world {
		world[i] = make([]uint8, p.ImageWidth)
	}
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			world[j][i] = <-c.ioInput
		}
	}

	server := flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")
	flag.Parse()
	fmt.Println("Server: ", *server)
	client, _ := rpc.Dial("tcp", *server)
	defer client.Close()
	updateRequest := stubs.StateRequest{World: world,
		ImageHeight: p.ImageHeight,
		ImageWidth:  p.ImageWidth,
		Turns:       p.Turns,
		Threads:     p.Threads}

	updateResponse := new(stubs.StateResponse)
	client.Call(stubs.UpdateStateHandler, updateRequest, updateResponse)
	world = updateResponse.World
	//cellCountRequest := stubs.StateRequest{
	//	World:       world,
	//	ImageHeight: p.ImageHeight,
	//	ImageWidth:  p.ImageWidth,
	//	Turns:       p.Turns,
	//	Threads:     p.Threads}
	//cellCountResponse := new(stubs.CellCountResponse)
	//client.Call(stubs.GetAliveCellsHandler, cellCountRequest, cellCountResponse)
	var alive []util.Cell
	alive = findAliveCells(world, p.ImageWidth, p.ImageHeight)

	c.events <- FinalTurnComplete{p.Turns, alive}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
