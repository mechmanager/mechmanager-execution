package input

import (
	"mechmanager-execution/domain"

	"github.com/google/uuid"
)

type ExecutionUseCaseInterface interface {
	CreateFromOrder(orderID, mechanicID string) (*domain.Execution, error)
	FindByID(id uuid.UUID) (*domain.Execution, error)
	FindByOrderID(orderID string) (*domain.Execution, error)
	ListAll() ([]*domain.Execution, error)
	UpdateStatus(id uuid.UUID, status domain.ExecutionStatus, notes string) (*domain.Execution, error)
	Complete(id uuid.UUID, repairNotes string) (*domain.Execution, error)
	Fail(id uuid.UUID, reason string) (*domain.Execution, error)
}
