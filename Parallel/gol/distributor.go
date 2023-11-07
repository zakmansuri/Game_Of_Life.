package gol

import (
	"fmt"
	"strconv"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	// Event is what is used to communicate with SDL
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	keyPresses <-chan rune
}

// OutPutWorldImage : used to send the world to convert it to a PGM image via ioOutput
func OutPutWorldImage(c distributorChannels, p Params, world [][]byte, turn int) {
	c.ioCommand <- ioCheckIdle
	Idle := <-c.ioIdle
	if Idle == true {
		n := strconv.Itoa(turn)
		t := strconv.Itoa(p.ImageWidth)
		t = t + "x" + t
		c.ioCommand <- ioOutput
		c.ioFilename <- t + "x" + n
		for i := 0; i < p.ImageHeight; i++ {
			for j := 0; j < p.ImageWidth; j++ {
				c.ioOutput <- world[j][i]
			}

		}
	}
}

// finds all alive cells and puts them in a slice
func calculateAliveCells(w, h int, world [][]byte) []util.Cell {

	var slice []util.Cell
	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			if world[i][j] == 0xFF {
				slice = append(slice, util.Cell{i, j})
			}
		}
	}

	return slice
}

// sums up the number of alive cells around a specific cell
func countAliveCellsAroundCell(p Params, world [][]byte, x int, y int) byte {
	sum := world[(y+p.ImageHeight-1)%p.ImageHeight][(x+p.ImageWidth-1)%p.ImageWidth] + world[(y+p.ImageHeight-1)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth] +
		world[(y+p.ImageHeight-1)%p.ImageHeight][(x+p.ImageWidth+1)%p.ImageWidth] + world[(y+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth-1)%p.ImageWidth] + world[(y+p.ImageHeight)%p.ImageHeight][(x+p.ImageWidth+1)%p.ImageWidth] +
		world[(y+p.ImageHeight+1)%p.ImageHeight][(x+p.ImageWidth-1)%p.ImageWidth] + world[(y+p.ImageHeight+1)%p.ImageHeight][(x+p.ImageWidth)%p.ImageWidth] +
		world[(y+p.ImageHeight+1)%p.ImageHeight][(x+p.ImageWidth+1)%p.ImageWidth]

	return sum
}

func updateW(p Params, world [][]byte, world2 [][]byte, bh int, h int, t int, c distributorChannels) [][]byte {

	for i := bh; i <= h; i++ {
		for j := 0; j < (p.ImageWidth); j++ {

			count := countAliveCellsAroundCell(p, world, j, i)
			count = 255 - count + 1

			if world[i][j] == 0xFF {
				if count < 2 {
					world2[i][j] = 0x00
					c.events <- CellFlipped{t, util.Cell{j, i}}
				} else if count == 2 || count == 3 {
					world2[i][j] = 0xFF
				} else {
					world2[i][j] = 0x00
					c.events <- CellFlipped{t, util.Cell{j, i}}
				}
			} else {
				if count == 3 {
					world2[i][j] = 0xFF
					c.events <- CellFlipped{t, util.Cell{j, i}}
				} else {
					world2[i][j] = 0x00
				}
			}
		}
	}

	temp := world2[bh : h+1]

	return temp
}

// goes through 2D array to get number of alive cells
func getNumberAliveCells(w [][]byte) int {
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
			if world[j][i] == 0xFF {
				c.events <- CellFlipped{0, util.Cell{i, j}}
			}
		}
	}
	turn := 0
	qStatus := false

	heights := make([]int, p.Threads)
	// splits heights per thread fairly
	for i := 0; i < p.ImageHeight; i++ {
		heights[i%p.Threads]++
	}

	ticker := time.NewTicker(2 * time.Second)
	done := make(chan bool, 1)

	// Executes all turns of the Game of Life.
	for turn = 0; turn < p.Turns; turn++ {

		bh := 0
		h := -1
		var sChanW []chan [][]byte

		for i := 0; i < p.Threads; i++ {

			chanW := make(chan [][]byte)
			sChanW = append(sChanW, chanW)

			h += heights[i]

			go worker(p, bh, h, world, sChanW[i], turn+1, c)

			bh += heights[i]
		}

		NewWorld := make([][]byte, p.ImageHeight)
		for i := range NewWorld {
			NewWorld[i] = make([]uint8, p.ImageWidth)
		}

		index := 0
		// receives and assembles the resulting world
		for i := 0; i < p.Threads; i++ {
			v := <-sChanW[i]
			for _, row := range v {
				NewWorld[index] = row
				index++
			}
		}

		world = NewWorld
		c.events <- TurnComplete{turn + 1}

		// different conditions
		select {
		case <-ticker.C:
			c.events <- AliveCellsCount{turn + 1, getNumberAliveCells(world)}
		case command := <-c.keyPresses:
			switch command {
			case 's':
				c.events <- StateChange{turn + 1, Executing}
				OutPutWorldImage(c, p, world, turn+1)
			case 'q':
				c.events <- StateChange{turn + 1, Quitting}
				qStatus = true
			case 'p':
				c.events <- StateChange{turn + 1, Paused}
				OutPutWorldImage(c, p, world, turn+1)
				pStatus := 0

				for {
					command := <-c.keyPresses
					switch command {
					case 'p':
						fmt.Println("continuing")
						c.events <- StateChange{turn + 1, Executing}
						c.events <- TurnComplete{turn + 1}
						pStatus = 1
					}
					if pStatus == 1 {
						break
					}
				}
			}
		default:
		}
		// for quiting the programme: q
		if qStatus == true {
			break
		}

	}

	done <- false

	var alive []util.Cell
	alive = calculateAliveCells(p.ImageWidth, p.ImageHeight, world)

	// Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{turn, alive}

	// Makes sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	Idle := <-c.ioIdle
	if Idle == true {
		n := strconv.Itoa(p.Turns)
		//fmt.Println("yes")

		c.ioCommand <- ioOutput

		c.ioFilename <- t + "x" + n

		for i := 0; i < p.ImageHeight; i++ {
			for j := 0; j < p.ImageWidth; j++ {
				c.ioOutput <- world[j][i]
			}
		}
	}

	c.ioCommand <- ioCheckIdle
	Idle = <-c.ioIdle

	if Idle == true {
		c.events <- StateChange{turn, Quitting}
	}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
