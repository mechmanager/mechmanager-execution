package persistence

import (
	"mechmanager-execution/domain"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ExecutionModel struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrderID         string         `gorm:"not null;uniqueIndex"`
	Status          string         `gorm:"not null;default:'QUEUED'"`
	MechanicID      string         `gorm:""`
	DiagnosticNotes string         `gorm:"type:text"`
	RepairNotes     string         `gorm:"type:text"`
	StartedAt       time.Time      `gorm:"not null"`
	UpdatedAt       time.Time      `gorm:"not null"`
	CompletedAt     *time.Time     `gorm:""`
	CreatedAt       time.Time      `gorm:"not null"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (ExecutionModel) TableName() string {
	return "executions"
}

func (m *ExecutionModel) ToDomain() *domain.Execution {
	return &domain.Execution{
		ID:              m.ID,
		OrderID:         m.OrderID,
		Status:          domain.ExecutionStatus(m.Status),
		MechanicID:      m.MechanicID,
		DiagnosticNotes: m.DiagnosticNotes,
		RepairNotes:     m.RepairNotes,
		StartedAt:       m.StartedAt,
		UpdatedAt:       m.UpdatedAt,
		CompletedAt:     m.CompletedAt,
		CreatedAt:       m.CreatedAt,
	}
}

func FromDomain(e *domain.Execution) *ExecutionModel {
	return &ExecutionModel{
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
