package stubs

import "uk.ac.bris.cs/gameoflife/util"

var UpdateStateHandler = "GOLOperations.UpdateState"
var TurnHandler = "WorkerOperations.CalculateNextState"
var GetAliveCellsHandler = "GOLOperations.GetAliveCells"
var CalculateTotalAliveCellsHandler = "GOLOperations.AliveCellCount"
var GetCurrentStateHandler = "GOLOperations.ReturnCurrentState"
var PausedGameHandler = "GOLOperations.PauseProcessing"
var KillServerHandler = "GOLOperations.KillServer"
var KillWorkerHandler = "WorkerOperations.KillServer"
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

type TurnRequest struct {
	Slice [][]byte
	World [][]byte
	Start int
	End   int
}

type TurnResponse struct {
	Slice [][]byte
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

type QuitResponse struct {
	MSG string
}

type EmptyResponse struct {
}

type CellCountResponse struct {
	TotalCells    int
	TurnsComplete int
}
