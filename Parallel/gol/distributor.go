package gol

import (
	"fmt"
	"strconv"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

//  distributorChannels defines the channels used for communication with the SDL context.
//  It includes channels for handling events enabling efficient data flow
type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	keyPresses <-chan rune
}

//  OutPutWorld generates and sends an image of the current state of the world to the IO system.
func OutPutWorld(c distributorChannels, p Params, world [][]byte, turn int) {
	// Checks the IO system is idle by sending ioCheckIdle command and reciving idle status
	c.ioCommand <- ioCheckIdle
	Idle := <-c.ioIdle
	if Idle == true {
		// strconv to convert into string
		n := strconv.Itoa(turn)
		t := strconv.Itoa(p.ImageWidth)
		t = t + "x" + t
		c.ioCommand <- ioOutput
		// Send filename to IO system
		c.ioFilename <- t + "x" + n
		for i := 0; i < p.ImageHeight; i++ {
			for j := 0; j < p.ImageWidth; j++ {
				// Send the byte value of each cell to the IO system for image construction
				c.ioOutput <- world[j][i]
			}

		}
	}
}

// finds all alive cells and puts them in a slice
func findAliveCells(IMWD, IMHT int, world [][]byte) []util.Cell {

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

func updateNextState(p Params, world [][]byte, nextState [][]byte, bh int, h int, t int, c distributorChannels) [][]byte {

	for y := bh; y <= h; y++ {
		for x := 0; x < (p.ImageWidth); x++ {

			sum := (int(world[(y+p.ImageHeight-1)%p.ImageHeight][(x+p.ImageWidth-1)%p.ImageWidth]) +
				int(world[(y+p.ImageHeight-1)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth]) +
				int(world[(y+p.ImageHeight-1)%p.ImageHeight][(x+p.ImageWidth+1)%p.ImageWidth]) +
				int(world[(y+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth-1)%p.ImageWidth]) +
				int(world[(y+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth+1)%p.ImageWidth]) +
				int(world[(y+p.ImageHeight+1)%p.ImageHeight][(x+p.ImageWidth-1)%p.ImageWidth]) +
				int(world[(y+p.ImageHeight+1)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth]) +
				int(world[(y+p.ImageHeight+1)%p.ImageHeight][(x+p.ImageWidth+1)%p.ImageWidth])) / 255

			if world[y][x] == 0xFF {
				if sum < 2 {
					nextState[y][x] = 0x00
					c.events <- CellFlipped{t, util.Cell{x, y}}
				} else if sum == 2 || sum == 3 {
					nextState[y][x] = 0xFF
				} else {
					nextState[y][x] = 0x00
					c.events <- CellFlipped{t, util.Cell{x, y}}
				}
			} else {
				if sum == 3 {
					nextState[y][x] = 0xFF
					c.events <- CellFlipped{t, util.Cell{x, y}}
				} else {
					nextState[y][x] = 0x00
				}
			}
		}
	}

	workerSlice := nextState[bh : h+1]

	return workerSlice
}

// goes through 2D array to get number of alive cells
func getAliveCells(w [][]byte) int {
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

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// loads in the world into a 2D slice from a PGM image
	t := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageWidth)
	c.ioCommand <- ioInput
	c.ioFilename <- t

	world := make([][]uint8, p.ImageHeight)
	for i := 0; i < p.ImageHeight; i++ {
		world[i] = make([]uint8, p.ImageWidth)
	}

	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			world[j][i] = <-c.ioInput
			if world[j][i] == 0xFF {
				c.events <- CellFlipped{0, util.Cell{i, j}}
			}
		}
	}
	turn := 0
	quit := false

	ticker := time.NewTicker(2 * time.Second)
	done := make(chan bool, 1)

	heights := make([]int, p.Threads)
	// splits heights per thread fairly
	for i := 0; i < p.ImageHeight; i++ {
		heights[i%p.Threads]++
	}

	// Executes all turns of the Game of Life.
	for turn < p.Turns {

		bh := 0 // base height for the index of the world for worker
		h := -1 // end index of the world for worker
		var sliceChanW []chan [][]byte

		for i := 0; i < p.Threads; i++ {

			channelW := make(chan [][]byte)
			sliceChanW = append(sliceChanW, channelW)

			h += heights[i]

			go worker(p, bh, h, world, sliceChanW[i], turn+1, c)

			bh += heights[i]
		}

		NewState := make([][]byte, p.ImageHeight)
		for i := 0; i < p.ImageHeight; i++ {
			NewState[i] = make([]byte, p.ImageWidth)
		}

		rowIndex := 0
		// receives and assembles the resulting world
		for i := 0; i < p.Threads; i++ {
			workerSlice := <-sliceChanW[i]
			for _, row := range workerSlice {
				NewState[rowIndex] = row
				rowIndex++
			}
		}

		world = NewState
		c.events <- TurnComplete{turn + 1}

		// different conditions
		select {
		case <-ticker.C:
			c.events <- AliveCellsCount{turn + 1, getAliveCells(world)}
		case command := <-c.keyPresses:
			switch command {
			case 's':
				c.events <- StateChange{turn + 1, Executing}
				OutPutWorld(c, p, world, turn+1)
				//saves the game into file
			case 'q':
				c.events <- StateChange{turn + 1, Quitting}
				quit = true
				//quits the game and stops processing
			case 'p':
				c.events <- StateChange{turn + 1, Paused}
				OutPutWorld(c, p, world, turn+1)
				//pauses and outputs the game
				pause := false

				for {
					command := <-c.keyPresses
					switch command {
					case 'p':
						fmt.Println("continuing")
						c.events <- StateChange{turn + 1, Executing}
						c.events <- TurnComplete{turn + 1}
						pause = true
						break
					}
					if pause {
						break
					}
				}
			}
		default:
		}
		// for quiting the programme: q
		if quit {
			break
		}
		turn++
	}

	done <- false

	var alive []util.Cell
	alive = findAliveCells(p.ImageWidth, p.ImageHeight, world)

	// Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{turn, alive}

	// Makes sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	if <-c.ioIdle {
		n := strconv.Itoa(p.Turns)

		c.ioCommand <- ioOutput

		c.ioFilename <- t + "x" + n

		for i := 0; i < p.ImageHeight; i++ {
			for j := 0; j < p.ImageWidth; j++ {
				c.ioOutput <- world[j][i]
			}
		}
	}

	c.ioCommand <- ioCheckIdle
	if <-c.ioIdle {
		c.events <- StateChange{turn, Quitting}
	}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

