package gol

import (
	"flag"
	"log"
	"net/rpc"
	"strconv"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var server = flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	keyPresses <-chan rune
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	t := strconv.Itoa(p.ImageWidth)
	t = t + "x" + t
	c.ioCommand <- ioInput
	c.ioFilename <- t
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			world[j][i] = <-c.ioInput
		}
	}

	flag.Parse()
	client, err := rpc.Dial("tcp", *server)
	defer client.Close()

	updateRequest := stubs.StateRequest{
		World:       world,
		ImageHeight: p.ImageHeight,
		ImageWidth:  p.ImageWidth,
		Threads:     p.Threads,
		Turns:       p.Turns}

	updateResponse := new(stubs.StateResponse)

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				totalCellResponse := new(stubs.CellCountResponse)
				err = client.Call(stubs.CalculateTotalAliveCellsHandler, stubs.EmptyRequest{}, totalCellResponse)
				if err != nil {
					log.Fatal("call error : ", err)
				}
				c.events <- AliveCellsCount{totalCellResponse.TurnsComplete, totalCellResponse.TotalCells}
			}
		}
	}()
	client.Call(stubs.UpdateStateHandler, updateRequest, updateResponse)
	world = updateResponse.World

	cellCountRequest := stubs.AliveCellRequest{
		ImageHeight: p.ImageHeight,
		ImageWidth:  p.ImageWidth}

	cellCountResponse := new(stubs.AliveCellResponse)
	client.Call(stubs.GetAliveCellsHandler, cellCountRequest, cellCountResponse)

	var alive = cellCountResponse.Cells

	//fmt.Printf("Alive Cell Count: %d\n", len(alive))

	c.events <- FinalTurnComplete{p.Turns, alive}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
