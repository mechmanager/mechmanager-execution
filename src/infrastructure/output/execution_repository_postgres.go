package output

import (
	"mechmanager-execution/application/port/output"
	"mechmanager-execution/domain"
	"mechmanager-execution/infrastructure/persistence"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var _ output.ExecutionRepositoryInterface = (*ExecutionPostgresRepository)(nil)

type ExecutionPostgresRepository struct {
	db *gorm.DB
}

func NewExecutionPostgresRepository(db *gorm.DB) *ExecutionPostgresRepository {
	return &ExecutionPostgresRepository{db: db}
}

func (r *ExecutionPostgresRepository) Save(execution *domain.Execution) (*domain.Execution, error) {
	model := persistence.FromDomain(execution)
	if err := r.db.Create(model).Error; err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *ExecutionPostgresRepository) FindByID(id uuid.UUID) (*domain.Execution, error) {
	var model persistence.ExecutionModel
	if err := r.db.Where("id = ?", id).First(&model).Error; err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *ExecutionPostgresRepository) FindByOrderID(orderID string) (*domain.Execution, error) {
	var model persistence.ExecutionModel
	if err := r.db.Where("order_id = ?", orderID).First(&model).Error; err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *ExecutionPostgresRepository) FindAll() ([]*domain.Execution, error) {
	var models []persistence.ExecutionModel
	if err := r.db.Find(&models).Error; err != nil {
		return nil, err
	}
	result := make([]*domain.Execution, len(models))
	for i, m := range models {
		copy := m
		result[i] = copy.ToDomain()
	}
	return result, nil
}

func (r *ExecutionPostgresRepository) Update(execution *domain.Execution) (*domain.Execution, error) {
	model := persistence.FromDomain(execution)
	if err := r.db.Save(model).Error; err != nil {
		return nil, err
	}
	return model.ToDomain(), nil
}
