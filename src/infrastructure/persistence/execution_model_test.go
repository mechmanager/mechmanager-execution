package persistence_test

import (
	"testing"
	"time"

	"mechmanager-execution/domain"
	"mechmanager-execution/infrastructure/persistence"

	"github.com/google/uuid"
)

func TestTableName(t *testing.T) {
	m := persistence.ExecutionModel{}
	if m.TableName() != "executions" {
		t.Errorf("TableName incorreto: got %s", m.TableName())
	}
}

func TestToDomain(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	m := &persistence.ExecutionModel{
		ID:              id,
		OrderID:         "order-001",
		Status:          "IN_DIAGNOSIS",
		MechanicID:      "mec-001",
		DiagnosticNotes: "barulho",
		RepairNotes:     "correia",
		StartedAt:       now,
		UpdatedAt:       now,
		CompletedAt:     nil,
		CreatedAt:       now,
	}

	e := m.ToDomain()

	if e.ID != id {
		t.Errorf("ID incorreto")
	}
	if e.OrderID != "order-001" {
		t.Errorf("OrderID incorreto")
	}
	if e.Status != domain.ExecutionStatusInDiagnosis {
		t.Errorf("Status incorreto: got %s", e.Status)
	}
	if e.MechanicID != "mec-001" {
		t.Errorf("MechanicID incorreto")
	}
	if e.DiagnosticNotes != "barulho" {
		t.Errorf("DiagnosticNotes incorreto")
	}
	if e.CompletedAt != nil {
		t.Error("CompletedAt deveria ser nil")
	}
}

func TestToDomain_WithCompletedAt(t *testing.T) {
	now := time.Now()
	m := &persistence.ExecutionModel{
		ID:          uuid.New(),
		OrderID:     "order-002",
		Status:      "COMPLETED",
		CompletedAt: &now,
	}

	e := m.ToDomain()

	if e.CompletedAt == nil {
		t.Error("CompletedAt não deve ser nil")
	}
	if e.Status != domain.ExecutionStatusCompleted {
		t.Errorf("Status incorreto: got %s", e.Status)
	}
}

func TestFromDomain(t *testing.T) {
	now := time.Now()
	id := uuid.New()

	e := &domain.Execution{
		ID:              id,
		OrderID:         "order-003",
		Status:          domain.ExecutionStatusInRepair,
		MechanicID:      "mec-002",
		DiagnosticNotes: "diagnóstico",
		RepairNotes:     "reparo",
		StartedAt:       now,
		UpdatedAt:       now,
		CompletedAt:     nil,
		CreatedAt:       now,
	}

	m := persistence.FromDomain(e)

	if m.ID != id {
		t.Errorf("ID incorreto")
	}
	if m.OrderID != "order-003" {
		t.Errorf("OrderID incorreto")
	}
	if m.Status != "IN_REPAIR" {
		t.Errorf("Status incorreto: got %s", m.Status)
	}
	if m.MechanicID != "mec-002" {
		t.Errorf("MechanicID incorreto")
	}
	if m.CompletedAt != nil {
		t.Error("CompletedAt deveria ser nil")
	}
}

func TestFromDomain_WithCompletedAt(t *testing.T) {
	now := time.Now()
	e := &domain.Execution{
		ID:          uuid.New(),
		OrderID:     "order-004",
		Status:      domain.ExecutionStatusCompleted,
		CompletedAt: &now,
	}

	m := persistence.FromDomain(e)

	if m.CompletedAt == nil {
		t.Error("CompletedAt não deve ser nil")
	}
}
