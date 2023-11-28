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

//added flag to indicate if the state should continue from the previous call
var cont = flag.Bool("continue", false, "Bool to continue call")

func calculateAliveCells(world [][]byte, IMHT, IMWD int) []util.Cell {
	//function to return slice of alive cells as util.Cell struct
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
	//function to save image to output
	c.ioCommand <- ioOutput
	c.ioFilename <- f
	for y := 0; y < IMHT; y++ {
		for x := 0; x < IMWD; x++ {
			//send data byte by byte
			c.ioOutput <- w[y][x]
		}
	}
}

//main distributor function to connect with broker
func distributor(p Params, c distributorChannels) {
	//gain input of file from IO
	t := strconv.Itoa(p.ImageWidth)
	t = t + "x" + t
	c.ioCommand <- ioInput
	c.ioFilename <- t

	//put input world into new 2D array
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			world[i][j] = <-c.ioInput
		}
	}

	//new request to broker containing world, turns, image height, image width and Continue flag value
	stateRequest := stubs.StateRequest{
		World: world,
		Turns: p.Turns,
		IMHT:  p.ImageHeight,
		IMWD:  p.ImageWidth,
		Cont:  *cont,
	}

	//pointer to response variable
	stateResponse := new(stubs.StateResponse)

	//parse all flags and connect to broker on default "127.0.0.1:8030"
	flag.Parse()
	client, _ := rpc.Dial("tcp", *brokerAddr)
	defer client.Close()

	//new ticker to tick every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	//channel to indicate if program has finished processing and returned a result
	finished := make(chan bool)

	//anonymous go function to deal with ticker and keypresses
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				//new request and response pointer for call to retrieve the count of alive cells
				cellRequest := stubs.TotalCellRequest{}
				cellResponse := new(stubs.TotalCellResponse)
				//call to broker to retrieve count of alive cells
				_ = client.Call(stubs.CalcualteTotalAliveCells, cellRequest, cellResponse)
				//report alive cell count to events
				c.events <- AliveCellsCount{cellResponse.Turns, cellResponse.AliveCells}
			case <-finished:
				//ends the go routine when the state changes have finished
				return
			case command := <-c.keyPresses:
				//creates new request and response pointer for key presses to broker
				keyRequest := stubs.KeyRequest{command}
				keyResponse := new(stubs.KeyResponse)
				//call broker method for keypresses
				err := client.Call(stubs.KeyPresshandler, keyRequest, keyResponse)
				if err != nil {
					log.Fatal("Key Press Call Error:", err)
				}

				//creates appropriate file name for output
				outFileName := t + "x" + strconv.Itoa(keyResponse.Turns)
				switch command {
				case 's':
					//report a state change to executing with the turns and save PGM image
					c.events <- StateChange{keyResponse.Turns, Executing}
					savePGMImage(c, keyResponse.World, outFileName, p.ImageHeight, p.ImageWidth)
				case 'k':
					//make a call to the broker to kill the broker and save the image
					err := client.Call(stubs.KillServerHandler, stubs.KillRequest{}, new(stubs.KillResponse))
					savePGMImage(c, keyResponse.World, outFileName, p.ImageHeight, p.ImageWidth)
					//report state change to quitting as broker and workers are killed
					c.events <- StateChange{keyResponse.Turns, Quitting}
					if err != nil {
						log.Fatal("Kill Request Call Error:", err)
					}
					//send true to the finished channel to end the goroutine
					finished <- true
				case 'q':
					//change the state to quitting and send true to finished channel to end goroutine
					c.events <- StateChange{keyResponse.Turns, Quitting}
					finished <- true
				case 'p':
					//set paused to true and print the turns
					paused := true
					fmt.Println(keyResponse.Turns)

					//set state to paused as an event
					c.events <- StateChange{keyResponse.Turns, Paused}
					for paused == true {
						//wait until another 'P' is sent through as a keypress
						command := <-c.keyPresses
						switch command {
						case 'p':
							//send another key press call to the broker to ensure broker continues with processing
							keyRequest := stubs.KeyRequest{command}
							keyResponse := new(stubs.KeyResponse)
							client.Call(stubs.KeyPresshandler, keyRequest, keyResponse)
							c.events <- StateChange{keyResponse.Turns, Executing}
							fmt.Println("Continuing")
							//set paused to false to exit the for loop
							paused = false
						}
					}
				}

			}
		}
	}()

	//call the broker to pass the request and wait until a state is returned in the response
	err := client.Call(stubs.UpdateStateHandler, stateRequest, stateResponse)
	if err != nil {
		log.Fatal("Update State Call Error:", err)
	}

	//if the program completed then we can stop the goroutine, otherwise we know the program was quit
	if stateResponse.Turns == p.Turns {
		finished <- true
	}

	//retrieve the values from response variable
	world = stateResponse.World
	turns := stateResponse.Turns

	//calculate the slice of cells which are alive in the program and send it as a FinalTurnComplete event
	c.events <- FinalTurnComplete{turns, calculateAliveCells(world, p.ImageHeight, p.ImageWidth)}

	c.ioCommand <- ioCheckIdle
	Idle := <-c.ioIdle
	if Idle == true {
		//once Idle, output the file as a PGM
		outFileName := t + "x" + strconv.Itoa(p.Turns)
		savePGMImage(c, world, outFileName, p.ImageHeight, p.ImageWidth)
	}

	c.ioCommand <- ioCheckIdle
	Idle = <-c.ioIdle

	c.events <- StateChange{turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
