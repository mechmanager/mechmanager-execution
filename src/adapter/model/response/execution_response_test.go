package response_test

import (
	"testing"
	"time"

	"mechmanager-execution/adapter/model/response"
	"mechmanager-execution/domain"

	"github.com/google/uuid"
)

func TestFromDomain_AllFields(t *testing.T) {
	now := time.Now()
	id := uuid.New()
	e := &domain.Execution{
		ID:              id,
		OrderID:         "order-001",
		Status:          domain.ExecutionStatusInRepair,
		MechanicID:      "mec-001",
		DiagnosticNotes: "barulho no motor",
		RepairNotes:     "correia trocada",
		StartedAt:       now,
		UpdatedAt:       now,
		CompletedAt:     nil,
		CreatedAt:       now,
	}

	r := response.FromDomain(e)

	if r.ID != id {
		t.Errorf("ID incorreto: got %v", r.ID)
	}
	if r.OrderID != "order-001" {
		t.Errorf("OrderID incorreto: got %s", r.OrderID)
	}
	if r.Status != "IN_REPAIR" {
		t.Errorf("Status incorreto: got %s", r.Status)
	}
	if r.MechanicID != "mec-001" {
		t.Errorf("MechanicID incorreto")
	}
	if r.DiagnosticNotes != "barulho no motor" {
		t.Errorf("DiagnosticNotes incorreto")
	}
	if r.RepairNotes != "correia trocada" {
		t.Errorf("RepairNotes incorreto")
	}
	if r.CompletedAt != nil {
		t.Error("CompletedAt deveria ser nil")
	}
}

func TestFromDomain_WithCompletedAt(t *testing.T) {
	now := time.Now()
	e := &domain.Execution{
		ID:          uuid.New(),
		OrderID:     "order-002",
		Status:      domain.ExecutionStatusCompleted,
		CompletedAt: &now,
	}

	r := response.FromDomain(e)

	if r.CompletedAt == nil {
		t.Error("CompletedAt não deve ser nil quando preenchido")
	}
	if r.Status != "COMPLETED" {
		t.Errorf("Status incorreto: got %s", r.Status)
	}
}
