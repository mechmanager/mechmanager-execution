package output

import "mechmanager-execution/domain"

type ExecutionQueueRepositoryInterface interface {
	Enqueue(execution *domain.Execution) error
	UpdateQueueStatus(orderID string, status domain.ExecutionStatus) error
	GetByStatus(status domain.ExecutionStatus) ([]*domain.Execution, error)
	Remove(orderID string) error
}
