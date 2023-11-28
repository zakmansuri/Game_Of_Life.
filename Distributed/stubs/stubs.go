package stubs

var UpdateStateHandler = "GOLOperations.UpdateState"
var CalculateNextStateHandler = "GOLOperations.CalculateNextState"
var CalcualteTotalAliveCells = "GOLOperations.CalculateTotalCells"
var KeyPresshandler = "GOLOperations.PressedKey"
var KillServerHandler = "GOLOperations.KillServer"

// QuitHandler Fault tolerance
var QuitHandler = "GOLOperations.QuitBroker"

type StateRequest struct {
	World [][]byte
	Turns int
	IMHT  int
	IMWD  int
	Cont  bool
}

type StateResponse struct {
	World [][]byte
	Turns int
}

type WorkerRequest struct {
	Slice [][]byte
	Start int
	End   int
}

type WorkerResponse struct {
	Slice [][]byte
}

type TotalCellRequest struct {
}

type TotalCellResponse struct {
	AliveCells int
	Turns      int
}

type KeyRequest struct {
	Key rune
}

type KeyResponse struct {
	World [][]byte
	Turns int
}

type KillRequest struct {
}

type KillResponse struct {
}

type QuitRequest struct {
	Cont bool
}

type QuitResponse struct {
}
