package output

import (
	"context"
	"fmt"
	"mechmanager-execution/application/port/output"
	"mechmanager-execution/domain"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var _ output.ExecutionQueueRepositoryInterface = (*ExecutionDynamoRepository)(nil)

type ExecutionDynamoRepository struct {
	client    *dynamodb.Client
	tableName string
}

func NewExecutionDynamoRepository(client *dynamodb.Client) *ExecutionDynamoRepository {
	return &ExecutionDynamoRepository{
		client:    client,
		tableName: os.Getenv("DYNAMO_EXECUTION_TABLE"),
	}
}

func (r *ExecutionDynamoRepository) Enqueue(execution *domain.Execution) error {
	_, err := r.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item: map[string]ddbtypes.AttributeValue{
			"order_id":     &ddbtypes.AttributeValueMemberS{Value: execution.OrderID},
			"execution_id": &ddbtypes.AttributeValueMemberS{Value: execution.ID.String()},
			"status":       &ddbtypes.AttributeValueMemberS{Value: string(execution.Status)},
			"mechanic_id":  &ddbtypes.AttributeValueMemberS{Value: execution.MechanicID},
			"enqueued_at":  &ddbtypes.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
			"updated_at":   &ddbtypes.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
		},
	})
	if err != nil {
		return fmt.Errorf("dynamo enqueue: %w", err)
	}
	return nil
}

func (r *ExecutionDynamoRepository) UpdateQueueStatus(orderID string, status domain.ExecutionStatus) error {
	_, err := r.client.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]ddbtypes.AttributeValue{
			"order_id": &ddbtypes.AttributeValueMemberS{Value: orderID},
		},
		UpdateExpression: aws.String("SET #s = :s, updated_at = :u"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":s": &ddbtypes.AttributeValueMemberS{Value: string(status)},
			":u": &ddbtypes.AttributeValueMemberN{Value: strconv.FormatInt(time.Now().Unix(), 10)},
		},
	})
	if err != nil {
		return fmt.Errorf("dynamo update status: %w", err)
	}
	return nil
}

func (r *ExecutionDynamoRepository) GetByStatus(status domain.ExecutionStatus) ([]*domain.Execution, error) {
	out, err := r.client.Scan(context.TODO(), &dynamodb.ScanInput{
		TableName:        aws.String(r.tableName),
		FilterExpression: aws.String("#s = :s"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":s": &ddbtypes.AttributeValueMemberS{Value: string(status)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("dynamo scan: %w", err)
	}

	var executions []*domain.Execution
	for _, item := range out.Items {
		e := &domain.Execution{}
		if v, ok := item["order_id"].(*ddbtypes.AttributeValueMemberS); ok {
			e.OrderID = v.Value
		}
		if v, ok := item["status"].(*ddbtypes.AttributeValueMemberS); ok {
			e.Status = domain.ExecutionStatus(v.Value)
		}
		if v, ok := item["mechanic_id"].(*ddbtypes.AttributeValueMemberS); ok {
			e.MechanicID = v.Value
		}
		executions = append(executions, e)
	}
	return executions, nil
}

func (r *ExecutionDynamoRepository) Remove(orderID string) error {
	_, err := r.client.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]ddbtypes.AttributeValue{
			"order_id": &ddbtypes.AttributeValueMemberS{Value: orderID},
		},
	})
	if err != nil {
		return fmt.Errorf("dynamo remove: %w", err)
	}
	return nil
}
