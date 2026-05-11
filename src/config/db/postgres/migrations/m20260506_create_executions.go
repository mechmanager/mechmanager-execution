package migrations

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func M20260506CreateExecutions() *gormigrate.Migration {
	type Execution struct {
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

	return &gormigrate.Migration{
		ID: "20260506_create_executions",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&Execution{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable("executions")
		},
	}
}
