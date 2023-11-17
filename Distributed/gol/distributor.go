package gol

import (
	"flag"
	"fmt"
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

func savePGMImage(c distributorChannels, w [][]byte, f string) {
	c.ioCommand <- ioOutput
	c.ioFilename <- f
	for y := 0; y < len(w); y++ {
		for x := 0; x < len(w[0]); x++ {
			c.ioOutput <- w[x][y]
		}
	}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	t := strconv.Itoa(p.ImageWidth)
	fileName := t + "x" + t
	c.ioCommand <- ioInput
	c.ioFilename <- fileName
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
	ticker := time.NewTicker(2 * time.Second)
	quitStatus := false

	go func() {
		for {
			select {
			case <-ticker.C:
				totalCellRequest := stubs.EmptyRequest{}
				totalCellResponse := new(stubs.CellCountResponse)
				err = client.Call(stubs.CalculateTotalAliveCellsHandler, totalCellRequest, totalCellResponse)
				if err != nil {
					log.Fatal("call error : ", err)
					return
				}
				c.events <- AliveCellsCount{totalCellResponse.TurnsComplete, totalCellResponse.TotalCells}
			case command := <-c.keyPresses:
				stateRequest := stubs.EmptyRequest{}
				stateResponse := new(stubs.StateResponse)
				client.Call(stubs.GetCurrentStateHandler, stateRequest, stateResponse)
				switch command {
				case 's':
					outFileName := fileName + "x" + strconv.Itoa(stateResponse.Turns)
					c.events <- StateChange{stateResponse.Turns, Executing}
					savePGMImage(c, stateResponse.World, outFileName)
					c.events <- ImageOutputComplete{stateResponse.Turns, outFileName}
				case 'q':
					client.Call(stubs.KillProcessesHandler, stubs.EmptyRequest{}, &stubs.EmptyResponse{})
					c.events <- StateChange{stateResponse.Turns, Quitting}
					quitStatus = true
				case 'k':
					outFileName := fileName + "x" + strconv.Itoa(stateResponse.Turns)
					c.events <- StateChange{stateResponse.Turns, Quitting}
					savePGMImage(c, stateResponse.World, outFileName)
					client.Call(stubs.KillServerHandler, stubs.EmptyRequest{}, &stubs.StateResponse{})
					quitStatus = true
				case 'p':
					pauseRequest := stubs.EmptyRequest{}
					pauseResponse := new(stubs.StateResponse)
					client.Call(stubs.PausedGameHandler, pauseRequest, pauseResponse)
					c.events <- StateChange{pauseResponse.Turns, Paused}
					pStatus := 0
					for {
						command := <-c.keyPresses
						switch command {
						case 'p':
							pauseRequest := stubs.EmptyRequest{}
							pauseResponse := new(stubs.StateResponse)
							client.Call(stubs.PausedGameHandler, pauseRequest, pauseResponse)
							c.events <- StateChange{pauseResponse.Turns, Executing}
							fmt.Println("Continuing")
							pStatus = 1
							break
						}
						if pStatus == 1 {
							break
						}
					}
				}
			}
			if quitStatus {
				break
			}
		}
		client.Close()
		ticker.Stop()
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

	c.events <- FinalTurnComplete{cellCountResponse.Turns, alive}

	c.ioCommand <- ioCheckIdle
	isIdle := <-c.ioIdle
	if isIdle {
		outFileName := fileName + "x" + strconv.Itoa(p.Turns)
		savePGMImage(c, world, outFileName)
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
