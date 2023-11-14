package stubs

import "uk.ac.bris.cs/gameoflife/util"

var UpdateStateHandler = "GOLOperations.UpdateState"
var GetAliveCellsHandler = "GOLOperations.GetAliveCells"
var CalculateTotalAliveCellsHandler = "GOLOperations.AliveCellCount"

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

type AliveCellResponse struct {
	Cells []util.Cell
}

type CellCountRequest struct {
	World [][]byte
}

type CellCountResponse struct {
	TotalCells int
}
