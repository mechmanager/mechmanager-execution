package service_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"

	service "mechmanager-execution/adapter/service"
	"mechmanager-execution/domain"
)

func newDisabledNRApp(t *testing.T) *newrelic.Application {
	t.Helper()
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("test-execution"),
		newrelic.ConfigLicense("0123456789012345678901234567890123456789"),
		newrelic.ConfigEnabled(false),
	)
	if err != nil {
		t.Fatalf("failed to create NR app: %v", err)
	}
	return app
}

// mockSQSClient implements sqsClient interface
type mockSQSClient struct {
	sendErr    error
	receiveOut *sqs.ReceiveMessageOutput
	receiveErr error
	deleteErr  error
}

func (m *mockSQSClient) SendMessage(_ context.Context, _ *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	if m.sendErr != nil {
		return nil, m.sendErr
	}
	return &sqs.SendMessageOutput{}, nil
}

func (m *mockSQSClient) ReceiveMessage(_ context.Context, _ *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	if m.receiveErr != nil {
		return nil, m.receiveErr
	}
	if m.receiveOut != nil {
		return m.receiveOut, nil
	}
	return &sqs.ReceiveMessageOutput{}, nil
}

func (m *mockSQSClient) DeleteMessage(_ context.Context, _ *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	if m.deleteErr != nil {
		return nil, m.deleteErr
	}
	return &sqs.DeleteMessageOutput{}, nil
}

func awsSQSString(s string) *string { return &s }

func newExecution(orderID string) *domain.Execution {
	now := time.Now()
	return &domain.Execution{
		ID:        uuid.New(),
		OrderID:   orderID,
		Status:    domain.ExecutionStatusInRepair,
		StartedAt: now,
		UpdatedAt: now,
		CreatedAt: now,
	}
}

// --- NewSQSSender ---

func TestNewSQSSender_withEnvVar(t *testing.T) {
	os.Setenv("SQS_EXECUTION_COMPLETE_URL", "https://sqs.us-east-1.amazonaws.com/123/exec-complete")
	defer os.Unsetenv("SQS_EXECUTION_COMPLETE_URL")

	sender := service.NewSQSSender(nil, nil)
	if sender == nil {
		t.Fatal("expected non-nil sender")
	}
}

func TestNewSQSSender_emptyEnvVar(t *testing.T) {
	os.Unsetenv("SQS_EXECUTION_COMPLETE_URL")
	sender := service.NewSQSSender(nil, nil)
	if sender == nil {
		t.Fatal("expected non-nil sender")
	}
}

// --- SendExecutionComplete with New Relic ---

func TestSendExecutionComplete_WithNewRelic(t *testing.T) {
	client := &mockSQSClient{}
	nrApp := newDisabledNRApp(t)
	sender := service.NewSQSSender(client, nrApp)

	now := time.Now()
	exec := newExecution("order-nr-1")
	exec.RepairNotes = "ok"
	exec.CompletedAt = &now

	if err := sender.SendExecutionComplete(exec); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestSendExecutionFailed_WithNewRelic(t *testing.T) {
	client := &mockSQSClient{}
	nrApp := newDisabledNRApp(t)
	sender := service.NewSQSSender(client, nrApp)

	if err := sender.SendExecutionFailed(newExecution("order-nr-2"), "falha nr"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- SendExecutionComplete ---

func TestSendExecutionComplete_Success(t *testing.T) {
	client := &mockSQSClient{}
	sender := service.NewSQSSender(client, nil)

	now := time.Now()
	exec := newExecution("order-complete-1")
	exec.RepairNotes = "tudo ok"
	exec.CompletedAt = &now

	if err := sender.SendExecutionComplete(exec); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestSendExecutionComplete_SQSError(t *testing.T) {
	client := &mockSQSClient{sendErr: errors.New("sqs down")}
	sender := service.NewSQSSender(client, nil)

	err := sender.SendExecutionComplete(newExecution("order-err"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- SendExecutionFailed ---

func TestSendExecutionFailed_Success(t *testing.T) {
	client := &mockSQSClient{}
	sender := service.NewSQSSender(client, nil)

	if err := sender.SendExecutionFailed(newExecution("order-fail-1"), "peça indisponível"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestSendExecutionFailed_SQSError(t *testing.T) {
	client := &mockSQSClient{sendErr: errors.New("network error")}
	sender := service.NewSQSSender(client, nil)

	err := sender.SendExecutionFailed(newExecution("order-fail-2"), "motivo")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- Receive ---

func TestReceive_ReturnsMessages(t *testing.T) {
	body := `{"order_id":"o1","event":"EXECUTION_COMPLETED"}`
	client := &mockSQSClient{
		receiveOut: &sqs.ReceiveMessageOutput{
			Messages: []types.Message{
				{MessageId: awsSQSString("m1"), Body: awsSQSString(body)},
			},
		},
	}
	sender := service.NewSQSSender(client, nil)

	msgs, err := sender.Receive(context.TODO(), "https://sqs.us-east-1.amazonaws.com/123/q", 10, 20)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if *msgs[0].MessageId != "m1" {
		t.Errorf("expected message id m1, got %s", *msgs[0].MessageId)
	}
}

func TestReceive_Error(t *testing.T) {
	client := &mockSQSClient{receiveErr: errors.New("timeout")}
	sender := service.NewSQSSender(client, nil)

	_, err := sender.Receive(context.TODO(), "https://sqs.us-east-1.amazonaws.com/123/q", 10, 20)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReceive_EmptyQueue(t *testing.T) {
	client := &mockSQSClient{receiveOut: &sqs.ReceiveMessageOutput{Messages: []types.Message{}}}
	sender := service.NewSQSSender(client, nil)

	msgs, err := sender.Receive(context.TODO(), "https://sqs.us-east-1.amazonaws.com/123/q", 10, 20)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

// --- Delete ---

func TestDelete_Success(t *testing.T) {
	client := &mockSQSClient{}
	sender := service.NewSQSSender(client, nil)

	receipt := "rh-abc"
	if err := sender.Delete(context.TODO(), "https://sqs.us-east-1.amazonaws.com/123/q", &receipt); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDelete_Error(t *testing.T) {
	client := &mockSQSClient{deleteErr: errors.New("delete failed")}
	sender := service.NewSQSSender(client, nil)

	receipt := "rh-xyz"
	err := sender.Delete(context.TODO(), "https://sqs.us-east-1.amazonaws.com/123/q", &receipt)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
