package gol

func worker(p Params, bh int, h int, world [][]byte, out chan<- [][]byte, t int, c distributorChannels) {
	temp := make([][]byte, p.ImageWidth)
	for i := 0; i < p.ImageWidth; i++ {
		temp[i] = make([]byte, p.ImageWidth)
	}
	newSlice := updateNextState(p, world, temp, bh, h, t, c)

	out <- newSlice
}
