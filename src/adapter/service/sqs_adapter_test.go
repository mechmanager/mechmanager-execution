package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	service "mechmanager-execution/adapter/service"
	"mechmanager-execution/domain"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
)

// mockSQSAPI é um mock manual da interface SQSAPI
type mockSQSAPI struct {
	sendErr    error
	receiveErr error
	deleteErr  error
	messages   []sqstypes.Message
}

func (m *mockSQSAPI) SendMessage(_ context.Context, _ *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	return &sqs.SendMessageOutput{}, m.sendErr
}

func (m *mockSQSAPI) ReceiveMessage(_ context.Context, _ *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	if m.receiveErr != nil {
		return nil, m.receiveErr
	}
	return &sqs.ReceiveMessageOutput{Messages: m.messages}, nil
}

func (m *mockSQSAPI) DeleteMessage(_ context.Context, _ *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return &sqs.DeleteMessageOutput{}, m.deleteErr
}

func newExecution() *domain.Execution {
	now := time.Now()
	return &domain.Execution{
		ID:          uuid.New(),
		OrderID:     "order-sqs-001",
		Status:      domain.ExecutionStatusCompleted,
		RepairNotes: "correia trocada",
		CompletedAt: &now,
	}
}

// --- SendExecutionComplete ---

func TestSQSSender_SendExecutionComplete_Success(t *testing.T) {
	mock := &mockSQSAPI{}
	t.Setenv("SQS_EXECUTION_COMPLETE_URL", "http://sqs/execution-complete")
	sender := service.NewSQSSender(mock, nil)

	err := sender.SendExecutionComplete(newExecution())
	if err != nil {
		t.Fatalf("esperado sem erro, got: %v", err)
	}
}

func TestSQSSender_SendExecutionComplete_SQSError(t *testing.T) {
	mock := &mockSQSAPI{sendErr: errors.New("sqs unavailable")}
	t.Setenv("SQS_EXECUTION_COMPLETE_URL", "http://sqs/execution-complete")
	sender := service.NewSQSSender(mock, nil)

	err := sender.SendExecutionComplete(newExecution())
	if err == nil {
		t.Fatal("esperado erro quando SQS falha")
	}
}

// --- SendExecutionFailed ---

func TestSQSSender_SendExecutionFailed_Success(t *testing.T) {
	mock := &mockSQSAPI{}
	t.Setenv("SQS_EXECUTION_COMPLETE_URL", "http://sqs/execution-complete")
	sender := service.NewSQSSender(mock, nil)

	exec := newExecution()
	exec.Status = domain.ExecutionStatusFailed

	err := sender.SendExecutionFailed(exec, "peça indisponível")
	if err != nil {
		t.Fatalf("esperado sem erro, got: %v", err)
	}
}

func TestSQSSender_SendExecutionFailed_SQSError(t *testing.T) {
	mock := &mockSQSAPI{sendErr: errors.New("sqs error")}
	t.Setenv("SQS_EXECUTION_COMPLETE_URL", "http://sqs/execution-complete")
	sender := service.NewSQSSender(mock, nil)

	err := sender.SendExecutionFailed(newExecution(), "motivo")
	if err == nil {
		t.Fatal("esperado erro quando SQS falha no SendExecutionFailed")
	}
}

// --- Receive ---

func TestSQSSender_Receive_Success(t *testing.T) {
	body := "test-body"
	mock := &mockSQSAPI{
		messages: []sqstypes.Message{{Body: &body}},
	}
	sender := service.NewSQSSender(mock, nil)

	msgs, err := sender.Receive(context.Background(), "http://sqs/queue", 10, 20)
	if err != nil {
		t.Fatalf("esperado sem erro, got: %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("esperado 1 mensagem, got: %d", len(msgs))
	}
}

func TestSQSSender_Receive_Error(t *testing.T) {
	mock := &mockSQSAPI{receiveErr: errors.New("receive failed")}
	sender := service.NewSQSSender(mock, nil)

	msgs, err := sender.Receive(context.Background(), "http://sqs/queue", 10, 20)
	if err == nil {
		t.Fatal("esperado erro quando SQS falha no Receive")
	}
	if msgs != nil {
		t.Error("esperado nil para mensagens em caso de erro")
	}
}

// --- Delete ---

func TestSQSSender_Delete_Success(t *testing.T) {
	mock := &mockSQSAPI{}
	sender := service.NewSQSSender(mock, nil)

	handle := "receipt-handle-abc"
	err := sender.Delete(context.Background(), "http://sqs/queue", &handle)
	if err != nil {
		t.Fatalf("esperado sem erro, got: %v", err)
	}
}

func TestSQSSender_Delete_Error(t *testing.T) {
	mock := &mockSQSAPI{deleteErr: errors.New("delete failed")}
	sender := service.NewSQSSender(mock, nil)

	handle := "receipt-handle-abc"
	err := sender.Delete(context.Background(), "http://sqs/queue", &handle)
	if err == nil {
		t.Fatal("esperado erro quando SQS falha no Delete")
	}
}
