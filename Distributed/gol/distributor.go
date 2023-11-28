package gol

import (
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"strconv"
	"time"
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

var brokerAddr = flag.String("broker", "127.0.0.1:8030", "IP:port string to connect to as broker")

func calculateAliveCells(world [][]byte, IMHT, IMWD int) []util.Cell {
	var slice []util.Cell
	for y := 0; y < IMHT; y++ {
		for x := 0; x < IMWD; x++ {
			if world[y][x] == 0xFF {
				slice = append(slice, util.Cell{x, y})
			}
		}
	}
	return slice
}

func savePGMImage(c distributorChannels, w [][]byte, f string, IMHT, IMWD int) {
	c.ioCommand <- ioOutput
	c.ioFilename <- f
	for y := 0; y < IMHT; y++ {
		for x := 0; x < IMWD; x++ {
			c.ioOutput <- w[y][x]
		}
	}
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
			world[i][j] = <-c.ioInput
		}
	}
	stateRequest := stubs.StateRequest{
		World: world,
		Turns: p.Turns,
		IMHT:  p.ImageHeight,
		IMWD:  p.ImageWidth,
	}
	stateResponse := new(stubs.StateResponse)

	flag.Parse()
	client, _ := rpc.Dial("tcp", *brokerAddr)
	defer client.Close()

	ticker := time.NewTicker(2 * time.Second)
	finished := make(chan bool)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cellRequest := stubs.TotalCellRequest{}
				cellResponse := new(stubs.TotalCellResponse)
				_ = client.Call(stubs.CalcualteTotalAliveCells, cellRequest, cellResponse)
				c.events <- AliveCellsCount{cellResponse.Turns, cellResponse.AliveCells}

			case <-finished:
				return
			case command := <-c.keyPresses:
				keyRequest := stubs.KeyRequest{command}
				keyResponse := new(stubs.KeyResponse)
				err := client.Call(stubs.KeyPresshandler, keyRequest, keyResponse)
				if err != nil {
					log.Fatal("Key Press Call Error:", err)
				}
				outFileName := t + "x" + strconv.Itoa(keyResponse.Turns)
				switch command {
				case 's':
					c.events <- StateChange{keyResponse.Turns, Executing}
					savePGMImage(c, keyResponse.World, outFileName, p.ImageHeight, p.ImageWidth)
				case 'k':
					err := client.Call(stubs.KillServerHandler, stubs.KillRequest{}, new(stubs.KillResponse))
					savePGMImage(c, keyResponse.World, outFileName, p.ImageHeight, p.ImageWidth)
					c.events <- StateChange{keyResponse.Turns, Quitting}
					if err != nil {
						log.Fatal("Kill Request Call Error:", err)
					}
					finished <- true
				case 'q':
					c.events <- StateChange{keyResponse.Turns, Quitting}
					finished <- true
				case 'p':
					paused := true
					fmt.Println(keyResponse.Turns)
					c.events <- StateChange{keyResponse.Turns, Paused}
					for paused == true {
						command := <-c.keyPresses
						switch command {
						case 'p':
							keyRequest := stubs.KeyRequest{command}
							keyResponse := new(stubs.KeyResponse)
							client.Call(stubs.KeyPresshandler, keyRequest, keyResponse)
							c.events <- StateChange{keyResponse.Turns, Executing}
							fmt.Println("Continuing")
							paused = false
						}
					}
				}

			}
		}
	}()

	err := client.Call(stubs.UpdateStateHandler, stateRequest, stateResponse)
	if err != nil {
		log.Fatal("Update State Call Error:", err)
	}
	if stateResponse.Turns == p.Turns {
		finished <- true
	}
	world = stateResponse.World
	turns := stateResponse.Turns
	// Make sure that the Io has finished any output before exiting.
	c.events <- FinalTurnComplete{turns, calculateAliveCells(world, p.ImageHeight, p.ImageWidth)}

	c.ioCommand <- ioCheckIdle
	Idle := <-c.ioIdle
	if Idle == true {
		outFileName := t + "x" + strconv.Itoa(p.Turns)
		savePGMImage(c, world, outFileName, p.ImageHeight, p.ImageWidth)
	}

	c.ioCommand <- ioCheckIdle
	Idle = <-c.ioIdle

	c.events <- StateChange{turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
