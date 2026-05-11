package response

import (
	"mechmanager-execution/domain"
	"time"

	"github.com/google/uuid"
)

type ExecutionResponse struct {
	ID              uuid.UUID  `json:"id"`
	OrderID         string     `json:"order_id"`
	Status          string     `json:"status"`
	MechanicID      string     `json:"mechanic_id,omitempty"`
	DiagnosticNotes string     `json:"diagnostic_notes,omitempty"`
	RepairNotes     string     `json:"repair_notes,omitempty"`
	StartedAt       time.Time  `json:"started_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

func FromDomain(e *domain.Execution) *ExecutionResponse {
	return &ExecutionResponse{
		ID:              e.ID,
		OrderID:         e.OrderID,
		Status:          string(e.Status),
		MechanicID:      e.MechanicID,
		DiagnosticNotes: e.DiagnosticNotes,
		RepairNotes:     e.RepairNotes,
		StartedAt:       e.StartedAt,
		UpdatedAt:       e.UpdatedAt,
		CompletedAt:     e.CompletedAt,
		CreatedAt:       e.CreatedAt,
	}
}
