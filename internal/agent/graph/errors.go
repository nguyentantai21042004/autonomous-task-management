package graph

import "errors"

var (
	// ErrEmptyResponse: LLM tra ve response khong co Parts
	ErrEmptyResponse = errors.New("empty LLM response")

	// ErrNoPendingTool: NodeExecuteTool duoc goi nhung khong co PendingTool
	ErrNoPendingTool = errors.New("no pending tool in state")

	// ErrMaxSteps: engine vuot qua gioi han MaxGraphSteps
	ErrMaxSteps = errors.New("exceeded max graph steps")
)
