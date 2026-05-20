package service

import (
	"errors"
	"log"
	"time"

	portIn "mechmanager-execution/application/port/input"
	portOut "mechmanager-execution/application/port/output"
	"mechmanager-execution/domain"

	"github.com/google/uuid"
)

var _ portIn.ExecutionUseCaseInterface = (*ExecutionAdapter)(nil)

type ExecutionAdapter struct {
	repo      portOut.ExecutionRepositoryInterface
	dynamo    portOut.ExecutionQueueRepositoryInterface
	messenger portOut.MessageManager
}

func NewExecutionAdapter(
	repo portOut.ExecutionRepositoryInterface,
	dynamo portOut.ExecutionQueueRepositoryInterface,
	messenger portOut.MessageManager,
) *ExecutionAdapter {
	return &ExecutionAdapter{repo: repo, dynamo: dynamo, messenger: messenger}
}

func (a *ExecutionAdapter) CreateFromOrder(orderID, mechanicID string) (*domain.Execution, error) {
	if orderID == "" {
		return nil, errors.New("order_id é obrigatório")
	}
	if existing, err := a.repo.FindByOrderID(orderID); err == nil && existing != nil {
		log.Printf("[EXECUTION][WARN] OS %s já possui execução (status: %s) — mensagem duplicada descartada", orderID, existing.Status)
		return existing, nil
	}
	now := time.Now()
	execution := &domain.Execution{
		ID:         uuid.New(),
		OrderID:    orderID,
		Status:     domain.ExecutionStatusQueued,
		MechanicID: mechanicID,
		StartedAt:  now,
		UpdatedAt:  now,
		CreatedAt:  now,
	}
	saved, err := a.repo.Save(execution)
	if err != nil {
		return nil, err
	}
	if err := a.dynamo.Enqueue(saved); err != nil {
		log.Printf("[WARN] falha ao enfileirar no DynamoDB: %v", err)
	}
	log.Printf("[EXECUTION] OS %s enfileirada com status QUEUED", orderID)
	return saved, nil
}

func (a *ExecutionAdapter) FindByID(id uuid.UUID) (*domain.Execution, error) {
	return a.repo.FindByID(id)
}

func (a *ExecutionAdapter) FindByOrderID(orderID string) (*domain.Execution, error) {
	return a.repo.FindByOrderID(orderID)
}

func (a *ExecutionAdapter) ListAll() ([]*domain.Execution, error) {
	return a.repo.FindAll()
}

func (a *ExecutionAdapter) UpdateStatus(id uuid.UUID, status domain.ExecutionStatus, notes string) (*domain.Execution, error) {
	execution, err := a.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if !IsValidTransition(execution.Status, status) {
		return nil, errors.New("transição de status inválida: " + string(execution.Status) + " → " + string(status))
	}
	execution.Status = status
	execution.UpdatedAt = time.Now()
	switch status {
	case domain.ExecutionStatusInDiagnosis:
		execution.DiagnosticNotes = notes
	case domain.ExecutionStatusInRepair:
		execution.RepairNotes = notes
	}
	updated, err := a.repo.Update(execution)
	if err != nil {
		return nil, err
	}
	if err := a.dynamo.UpdateQueueStatus(execution.OrderID, status); err != nil {
		log.Printf("[WARN] falha ao atualizar DynamoDB: %v", err)
	}
	log.Printf("[EXECUTION] OS %s atualizada para %s", execution.OrderID, status)
	return updated, nil
}

func (a *ExecutionAdapter) Complete(id uuid.UUID, repairNotes string) (*domain.Execution, error) {
	execution, err := a.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if execution.Status != domain.ExecutionStatusInRepair {
		return nil, errors.New("execução deve estar em IN_REPAIR para ser concluída")
	}
	now := time.Now()
	execution.Status = domain.ExecutionStatusCompleted
	execution.RepairNotes = repairNotes
	execution.CompletedAt = &now
	execution.UpdatedAt = now

	updated, err := a.repo.Update(execution)
	if err != nil {
		return nil, err
	}
	if err := a.dynamo.Remove(execution.OrderID); err != nil {
		log.Printf("[WARN] falha ao remover do DynamoDB: %v", err)
	}
	if err := a.messenger.SendExecutionComplete(updated); err != nil {
		log.Printf("[WARN] falha ao enviar evento de conclusão para OS Service: %v", err)
	}
	log.Printf("[EXECUTION] OS %s CONCLUÍDA — evento enviado ao OS Service", execution.OrderID)
	return updated, nil
}

func (a *ExecutionAdapter) Fail(id uuid.UUID, reason string) (*domain.Execution, error) {
	execution, err := a.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	execution.Status = domain.ExecutionStatusFailed
	execution.DiagnosticNotes = reason
	execution.UpdatedAt = now

	updated, err := a.repo.Update(execution)
	if err != nil {
		return nil, err
	}
	if err := a.dynamo.UpdateQueueStatus(execution.OrderID, domain.ExecutionStatusFailed); err != nil {
		log.Printf("[WARN] falha ao atualizar DynamoDB: %v", err)
	}
	if err := a.messenger.SendExecutionFailed(updated, reason); err != nil {
		log.Printf("[WARN] falha ao enviar evento de falha (Saga rollback): %v", err)
	}
	log.Printf("[EXECUTION] OS %s FALHOU — evento de compensação Saga enviado", execution.OrderID)
	return updated, nil
}

// IsValidTransition valida as transições de status do Saga
func IsValidTransition(current, next domain.ExecutionStatus) bool {
	transitions := map[domain.ExecutionStatus][]domain.ExecutionStatus{
		domain.ExecutionStatusQueued:      {domain.ExecutionStatusInDiagnosis, domain.ExecutionStatusFailed},
		domain.ExecutionStatusInDiagnosis: {domain.ExecutionStatusInRepair, domain.ExecutionStatusFailed},
		domain.ExecutionStatusInRepair:    {domain.ExecutionStatusCompleted, domain.ExecutionStatusFailed},
	}
	allowed, ok := transitions[current]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == next {
			return true
		}
	}
	return false
}
