package stubs

import "uk.ac.bris.cs/gameoflife/util"

var UpdateStateHandler = "GOLOperations.UpdateState"
var GetAliveCellsHandler = "GOLOperations.GetAliveCellCount"

type StateResponse struct {
	World [][]byte
}

type StateRequest struct {
	World       [][]byte
	ImageHeight int
	ImageWidth  int
	Threads     int
	Turns       int
}

type CellCountResponse struct {
	Cells []util.Cell
}
