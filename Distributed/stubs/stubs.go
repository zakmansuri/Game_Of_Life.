package stubs

import "uk.ac.bris.cs/gameoflife/util"

var UpdateStateHandler = "GOLOperations.UpdateState"
var GetAliveCellsHandler = "GOLOperations.GetAliveCells"
var CalculateTotalAliveCellsHandler = "GOLOperations.AliveCellCount"
var GetCurrentStateHandler = "GOLOperations.ReturnCurrentState"
var PausedGameHandler = "GOLOperations.PauseProcessing"
var KillServerHandler = "GOLOperations.KillServer"
var KillProcessesHandler = "GOLOperations.KillProcesses"

type StateResponse struct {
	World [][]byte
	Turns int
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
	Turns int
}

type EmptyRequest struct {
}

type EmptyResponse struct {
}

type CellCountResponse struct {
	TotalCells    int
	TurnsComplete int
}
