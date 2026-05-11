package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

type ExecutionStatus string

const (
	ExecutionStatusQueued      ExecutionStatus = "QUEUED"
	ExecutionStatusInDiagnosis ExecutionStatus = "IN_DIAGNOSIS"
	ExecutionStatusInRepair    ExecutionStatus = "IN_REPAIR"
	ExecutionStatusCompleted   ExecutionStatus = "COMPLETED"
	ExecutionStatusFailed      ExecutionStatus = "FAILED"
)

type Execution struct {
	ID              uuid.UUID       `json:"id"`
	OrderID         string          `json:"order_id"`
	Status          ExecutionStatus `json:"status"`
	MechanicID      string          `json:"mechanic_id,omitempty"`
	DiagnosticNotes string          `json:"diagnostic_notes,omitempty"`
	RepairNotes     string          `json:"repair_notes,omitempty"`
	StartedAt       time.Time       `json:"started_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}
