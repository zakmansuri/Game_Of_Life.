package gol

import (
	"fmt"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	// TODO: Create a 2D slice to store the world.
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%d%s%d", p.ImageHeight, "x", p.ImageWidth)
	IMHT := p.ImageHeight
	IMWD := p.ImageWidth
	world := make([][]byte, IMHT)
	for y := range world {
		world[y] = make([]byte, IMWD)
		for x := range world[y] {
			world[y][x] = <-c.ioInput
		}
	}

	turn := 0
	// TODO: Execute all turns of the Game of Life.
	for turn <= p.Turns {
		world = calculateNextState(p, world)
		turn++
	}
	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{turn, calculateAliveCells(p, world)}
	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func calculateNextState(p Params, world [][]byte) [][]byte {
	IMHT := p.ImageHeight
	IMWD := p.ImageWidth

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
			if world[y][x] == 255 {
				if sum < 2 {
					newWorld[y][x] = 0
				} else if sum == 2 || sum == 3 {
					newWorld[y][x] = 255
				} else {
					newWorld[y][x] = 0
				}
			} else {
				if sum == 3 {
					newWorld[y][x] = 255
				} else {
					newWorld[y][x] = 0
				}
			}
		}
	}
	return newWorld
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	aliveCells := []util.Cell{}
	for y := range world {
		for x := range world[y] {
			if world[y][x] == 255 {
				aliveCells = append(aliveCells, util.Cell{x, y})
			}
		}
	}
	return aliveCells
}
