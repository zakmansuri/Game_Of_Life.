package stubs

//handlers for each GOL operation on broker and worker
var UpdateStateHandler = "GOLOperations.UpdateState"
var CalculateNextStateHandler = "GOLOperations.CalculateNextState"
var CalculateTotalAliveCells = "GOLOperations.CalculateTotalCells"
var KeyPresshandler = "GOLOperations.PressedKey"
var KillServerHandler = "GOLOperations.KillServer"

//set of structs for each handler with required variables inside
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
