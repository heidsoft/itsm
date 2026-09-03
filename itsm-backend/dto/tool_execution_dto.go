package dto

// ToolExecutionResponse distinguishes an accepted approval request from execution.
type ToolExecutionResponse struct {
	Status        string      `json:"status"`
	Summary       string      `json:"summary"`
	InvocationID  int         `json:"invocationId"`
	ApprovalState string      `json:"approvalState"`
	NextActions   []string    `json:"nextActions"`
	Artifacts     []string    `json:"artifacts"`
	Data          interface{} `json:"data"`
}
