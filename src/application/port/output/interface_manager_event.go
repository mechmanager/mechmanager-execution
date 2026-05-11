package output

import (
	"context"
	"mechmanager-execution/domain"

	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type MessageManager interface {
	SendExecutionComplete(execution *domain.Execution) error
	SendExecutionFailed(execution *domain.Execution, reason string) error
	Receive(ctx context.Context, queueURL string, maxMessages int32, waitTime int32) ([]types.Message, error)
	Delete(ctx context.Context, queueURL string, receiptHandle *string) error
}
