package output

import (
	"mechmanager-execution/domain"

	"github.com/google/uuid"
)

type ExecutionRepositoryInterface interface {
	Save(execution *domain.Execution) (*domain.Execution, error)
	FindByID(id uuid.UUID) (*domain.Execution, error)
	FindByOrderID(orderID string) (*domain.Execution, error)
	FindAll() ([]*domain.Execution, error)
	Update(execution *domain.Execution) (*domain.Execution, error)
}
