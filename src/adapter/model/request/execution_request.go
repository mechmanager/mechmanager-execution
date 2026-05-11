package request

type UpdateExecutionStatusInput struct {
	Status          string `json:"status" binding:"required"`
	MechanicID      string `json:"mechanic_id,omitempty"`
	DiagnosticNotes string `json:"diagnostic_notes,omitempty"`
	RepairNotes     string `json:"repair_notes,omitempty"`
}

type CompleteExecutionInput struct {
	RepairNotes string `json:"repair_notes" binding:"required"`
}

type FailExecutionInput struct {
	Reason string `json:"reason" binding:"required"`
}
