package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	portOut "mechmanager-execution/application/port/output"
	"mechmanager-execution/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type SQSAPI interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

var _ portOut.MessageManager = (*SQSSender)(nil)

type SQSSender struct {
	client           SQSAPI
	completeQueueURL string
}

func NewSQSSender(client SQSAPI) *SQSSender {
	return &SQSSender{
		client:           client,
		completeQueueURL: os.Getenv("SQS_EXECUTION_COMPLETE_URL"),
	}
}

type ExecutionEvent struct {
	OrderID     string     `json:"order_id"`
	ExecutionID string     `json:"execution_id"`
	Event       string     `json:"event"`
	RepairNotes string     `json:"repair_notes,omitempty"`
	Reason      string     `json:"reason,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

func (s *SQSSender) SendExecutionComplete(execution *domain.Execution) error {
	event := ExecutionEvent{
		OrderID:     execution.OrderID,
		ExecutionID: execution.ID.String(),
		Event:       "EXECUTION_COMPLETED",
		RepairNotes: execution.RepairNotes,
		CompletedAt: execution.CompletedAt,
	}
	return s.sendEvent(event)
}

func (s *SQSSender) SendExecutionFailed(execution *domain.Execution, reason string) error {
	event := ExecutionEvent{
		OrderID:     execution.OrderID,
		ExecutionID: execution.ID.String(),
		Event:       "EXECUTION_FAILED",
		Reason:      reason,
	}
	return s.sendEvent(event)
}

func (s *SQSSender) sendEvent(event ExecutionEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("erro ao serializar evento: %w", err)
	}
	_, err = s.client.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.completeQueueURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		return fmt.Errorf("erro ao enviar para SQS: %w", err)
	}
	log.Printf("[SQS] Evento %s enviado para fila execution-complete (order: %s)", event.Event, event.OrderID)
	return nil
}

func (s *SQSSender) Receive(ctx context.Context, queueURL string, maxMessages int32, waitTime int32) ([]types.Message, error) {
	result, err := s.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: maxMessages,
		WaitTimeSeconds:     waitTime,
	})
	if err != nil {
		return nil, err
	}
	return result.Messages, nil
}

func (s *SQSSender) Delete(ctx context.Context, queueURL string, receiptHandle *string) error {
	_, err := s.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: receiptHandle,
	})
	return err
}
