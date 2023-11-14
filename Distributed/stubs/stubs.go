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

type AliveCellRequest struct {
	ImageHeight int
	ImageWidth  int
}

type AliveCellResponse struct {
	Cells []util.Cell
}

type EmptyRequest struct {
}

type CellCountResponse struct {
	TotalCells    int
	TurnsComplete int
}
