package gol

// worker function is designed to update a slice of the Game of Life world
func worker(p Params, bh int, h int, world [][]byte, out chan<- [][]byte, t int, c distributorChannels) {
	// Initialize a temporary slice to store the updated state of the worlds slice.
	temp := make([][]byte, p.ImageWidth)
	for i := range temp {
		temp[i] = make([]byte, p.ImageWidth)
	}

	out <- updateNextState(p, world, temp, bh, h, t, c)

}

