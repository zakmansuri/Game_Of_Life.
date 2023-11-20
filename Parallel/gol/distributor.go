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

// findAliveCells iterates through the game world and identifies all the alive cells.
// It returns a slice of util.Cell, each representing the coordinates of an alive cell.
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

// updateNextState updates the state of a portion of the game world for the next turn based on 
// the Game of Life rules and sends events when cells change state.
func updateNextState(p Params, world [][]byte, nextState [][]byte, bh int, h int, t int, c distributorChannels) [][]byte {

	for y := bh; y <= h; y++ {
		for x := 0; x < (p.ImageWidth); x++ {
        		 // Calculate the sum of the states of the 8 neighboring cells, normalized to 0 or 1.
			sum := (int(world[(y+p.ImageHeight-1)%p.ImageHeight][(x+p.ImageWidth-1)%p.ImageWidth]) +
				int(world[(y+p.ImageHeight-1)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth]) +
				int(world[(y+p.ImageHeight-1)%p.ImageHeight][(x+p.ImageWidth+1)%p.ImageWidth]) +
				int(world[(y+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth-1)%p.ImageWidth]) +
				int(world[(y+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth+1)%p.ImageWidth]) +
				int(world[(y+p.ImageHeight+1)%p.ImageHeight][(x+p.ImageWidth-1)%p.ImageWidth]) +
				int(world[(y+p.ImageHeight+1)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth]) +
				int(world[(y+p.ImageHeight+1)%p.ImageHeight][(x+p.ImageWidth+1)%p.ImageWidth])) / 255

			 // Apply Game of Life rules based on the sum calculated.
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

	// Extract and return the slice of the updated state for this worker's portion.
	workerSlice := nextState[bh : h+1]
	return workerSlice
}

// getAliveCells counts the number of alive cells in a 2D array representing the Game of Life world.
func getAliveCells(w [][]byte) int {
	count := 0

	// Iterate over each row of the world.
	for i := 0; i < len(w); i++ {
		// Iterate over each column in the current row.
		for j := 0; j < len(w[0]); j++ {
			if w[i][j] == 255 {
				 // Increment the count for each alive cell found.
				count++
			}
		}
	}
	 // Return the total count of alive cells.
	return count
}

// distributor is responsible for dividing the work among worker goroutines and handling interactions
// with other parts of the system in a Game of Life simulation.
func distributor(p Params, c distributorChannels) {

  	// Prepare the filename for loading the initial world state from a PGM image.
   	// The filename is constructed based on the image width and height.
	t := strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.ImageWidth)
	// Send commands to the IO system to load the initial state.
	c.ioCommand <- ioInput
	c.ioFilename <- t

	// Initialize the world as a 2D slice of uint8 values, representing the game grid.
	world := make([][]uint8, p.ImageHeight)
	for i := 0; i < p.ImageHeight; i++ {
		world[i] = make([]uint8, p.ImageWidth)
	}

	// Fill the world slice with data received from the IO system.
    	// This loop reads the state of each cell in the PGM image.
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			world[j][i] = <-c.ioInput
			// If a cell is alive (denoted by 0xFF), send a CellFlipped event.
			if world[j][i] == 0xFF {
				c.events <- CellFlipped{0, util.Cell{i, j}}
			}
		}
	}

	// Initialize the turn counter and a flag to indicate when to quit the simulation.
	turn := 0
	quit := false

	// Create a ticker that will send a signal every 2 seconds.
	ticker := time.NewTicker(2 * time.Second)
	// Create a channel to signal when the simulation is done.
	done := make(chan bool, 1)

	// Initialize a slice to store the number of rows each worker goroutine will process.
	heights := make([]int, p.Threads)
	// Distribute the rows of the game world among the worker goroutines evenly.
	for i := 0; i < p.ImageHeight; i++ {
		heights[i%p.Threads]++
	}

	// Main loop for executing all turns of the Game of Life simulation.
	for turn < p.Turns {

		// Initialize variables for dividing the work among worker goroutines.
		bh := 0 // base height for the index of the world for worker
		h := -1 // end index of the world for worker
		var sliceChanW []chan [][]byte

		// Divide work among worker goroutines and start them.
		for i := 0; i < p.Threads; i++ {
			
     			// Create a channel for communication with each worker.
			channelW := make(chan [][]byte)
			sliceChanW = append(sliceChanW, channelW)

			h += heights[i]

			go worker(p, bh, h, world, sliceChanW[i], turn+1, c)

			// Update the base height for the next worker's segment.
			bh += heights[i]
		}

		// Initialize a 2D slice to store the new state of the world.
		NewState := make([][]byte, p.ImageHeight)
		for i := 0; i < p.ImageHeight; i++ {
			NewState[i] = make([]byte, p.ImageWidth)
		}

		// Assemble the new state of the world from slices returned by the workers.
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
		// Send an event indicating the completion of the current turn.
		c.events <- TurnComplete{turn + 1}

    		// Handle different conditions based on input or time triggers.
		select {
		case <-ticker.C:
			// Send an event with the count of alive cells.
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
		default: // No input received.
		}
   		// Check if the quit flag is set and break the loop if so.
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

