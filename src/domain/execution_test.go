package domain_test

import (
	"testing"
	"time"

	"mechmanager-execution/domain"

	"github.com/google/uuid"
)

func TestExecutionStatusConstants(t *testing.T) {
	if domain.ExecutionStatusQueued != "QUEUED" {
		t.Errorf("esperado QUEUED, got %s", domain.ExecutionStatusQueued)
	}
	if domain.ExecutionStatusInDiagnosis != "IN_DIAGNOSIS" {
		t.Errorf("esperado IN_DIAGNOSIS, got %s", domain.ExecutionStatusInDiagnosis)
	}
	if domain.ExecutionStatusInRepair != "IN_REPAIR" {
		t.Errorf("esperado IN_REPAIR, got %s", domain.ExecutionStatusInRepair)
	}
	if domain.ExecutionStatusCompleted != "COMPLETED" {
		t.Errorf("esperado COMPLETED, got %s", domain.ExecutionStatusCompleted)
	}
	if domain.ExecutionStatusFailed != "FAILED" {
		t.Errorf("esperado FAILED, got %s", domain.ExecutionStatusFailed)
	}
}

func TestErrNotFound(t *testing.T) {
	if domain.ErrNotFound == nil {
		t.Fatal("ErrNotFound não deve ser nil")
	}
	if domain.ErrNotFound.Error() != "not found" {
		t.Errorf("mensagem incorreta: %s", domain.ErrNotFound.Error())
	}
}

func TestExecutionStruct(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	e := domain.Execution{
		ID:              id,
		OrderID:         "order-123",
		Status:          domain.ExecutionStatusQueued,
		MechanicID:      "mec-001",
		DiagnosticNotes: "barulho no motor",
		RepairNotes:     "correia trocada",
		StartedAt:       now,
		UpdatedAt:       now,
		CompletedAt:     nil,
		CreatedAt:       now,
	}

	if e.ID != id {
		t.Error("ID incorreto")
	}
	if e.OrderID != "order-123" {
		t.Error("OrderID incorreto")
	}
	if e.Status != domain.ExecutionStatusQueued {
		t.Error("Status incorreto")
	}
	if e.CompletedAt != nil {
		t.Error("CompletedAt deveria ser nil")
	}
}

func TestExecutionWithCompletedAt(t *testing.T) {
	now := time.Now()
	e := domain.Execution{
		ID:          uuid.New(),
		OrderID:     "order-456",
		Status:      domain.ExecutionStatusCompleted,
		CompletedAt: &now,
	}
	if e.CompletedAt == nil {
		t.Error("CompletedAt não deve ser nil quando preenchido")
	}
}
