package service_test

import (
	"context"
	"mechmanager-execution/domain"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
)

// mockRepo simula o repositório PostgreSQL
type mockRepo struct {
	executions map[uuid.UUID]*domain.Execution
	saveErr    error
	findErr    error
	updateErr  error
}

func newMockRepo() *mockRepo {
	return &mockRepo{executions: make(map[uuid.UUID]*domain.Execution)}
}

func (m *mockRepo) Save(e *domain.Execution) (*domain.Execution, error) {
	if m.saveErr != nil {
		return nil, m.saveErr
	}
	m.executions[e.ID] = e
	return e, nil
}

func (m *mockRepo) FindByID(id uuid.UUID) (*domain.Execution, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	e, ok := m.executions[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return e, nil
}

func (m *mockRepo) FindByOrderID(orderID string) (*domain.Execution, error) {
	for _, e := range m.executions {
		if e.OrderID == orderID {
			return e, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockRepo) FindAll() ([]*domain.Execution, error) {
	result := make([]*domain.Execution, 0, len(m.executions))
	for _, e := range m.executions {
		result = append(result, e)
	}
	return result, nil
}

func (m *mockRepo) Update(e *domain.Execution) (*domain.Execution, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	m.executions[e.ID] = e
	return e, nil
}

// mockDynamo simula o repositório DynamoDB
type mockDynamo struct {
	enqueueErr error
	updateErr  error
	removeErr  error
	enqueued   []string
	removed    []string
}

func (m *mockDynamo) Enqueue(e *domain.Execution) error {
	if m.enqueueErr != nil {
		return m.enqueueErr
	}
	m.enqueued = append(m.enqueued, e.OrderID)
	return nil
}

func (m *mockDynamo) UpdateQueueStatus(orderID string, status domain.ExecutionStatus) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return nil
}

func (m *mockDynamo) GetByStatus(status domain.ExecutionStatus) ([]*domain.Execution, error) {
	return nil, nil
}

func (m *mockDynamo) Remove(orderID string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	m.removed = append(m.removed, orderID)
	return nil
}

// mockMessenger simula o SQS sender
type mockMessenger struct {
	completeEvents []string
	failedEvents   []string
	completeErr    error
	failedErr      error
}

func (m *mockMessenger) SendExecutionComplete(e *domain.Execution) error {
	if m.completeErr != nil {
		return m.completeErr
	}
	m.completeEvents = append(m.completeEvents, e.OrderID)
	return nil
}

func (m *mockMessenger) SendExecutionFailed(e *domain.Execution, reason string) error {
	if m.failedErr != nil {
		return m.failedErr
	}
	m.failedEvents = append(m.failedEvents, e.OrderID)
	return nil
}

func (m *mockMessenger) Receive(_ context.Context, _ string, _ int32, _ int32) ([]types.Message, error) {
	return nil, nil
}

func (m *mockMessenger) Delete(_ context.Context, _ string, _ *string) error {
	return nil
}
