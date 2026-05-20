package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	controllers "mechmanager-execution/adapter/controllers"
	usecase "mechmanager-execution/adapter/service"
	security "mechmanager-execution/config"
	"mechmanager-execution/config/db/postgres"
	"mechmanager-execution/config/db/postgres/migrations"
	infraOutput "mechmanager-execution/infrastructure/output"
)

// OrderMessage representa a mensagem recebida do OS Service via SQS
type OrderMessage struct {
	OrderID    string `json:"order_id"`
	MechanicID string `json:"mechanic_id,omitempty"`
	Status     string `json:"status"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: .env não encontrado, usando variáveis de ambiente")
	}

	ctx := context.Background()

	// AWS Config
	cfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		log.Fatalf("aws config: %v", err)
	}
	var sqsClient *sqs.Client
	var ddbClient *dynamodb.Client
	endpoint := os.Getenv("AWS_ENDPOINT_URL")
	if endpoint != "" {
		sqsClient = sqs.NewFromConfig(cfg, func(o *sqs.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
		ddbClient = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	} else {
		sqsClient = sqs.NewFromConfig(cfg)
		ddbClient = dynamodb.NewFromConfig(cfg)
	}

	// PostgreSQL
	db := postgres.StartPostgresDBConnection()
	migrations.RunMigrations(db)

	// Gin
	r := security.NewGinEngine(nil)

	// Repositories
	pgRepo := infraOutput.NewExecutionPostgresRepository(db)
	dynamoRepo := infraOutput.NewExecutionDynamoRepository(ddbClient)

	// SQS Sender
	sender := usecase.NewSQSSender(sqsClient)

	// Use Case
	executionUseCase := usecase.NewExecutionAdapter(pgRepo, dynamoRepo, sender)

	// Controller
	executionController := controllers.NewExecutionController(executionUseCase)

	// Routes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "mechmanager-execution"})
	})

	r.GET("/executions", executionController.ListAll)
	r.GET("/executions/order/:id", executionController.FindByOrderID)
	r.GET("/executions/:id", executionController.FindByID)
	r.PATCH("/executions/:id/status", executionController.UpdateStatus)
	r.PATCH("/executions/:id/complete", executionController.Complete)
	r.PATCH("/executions/:id/fail", executionController.Fail)

	// SQS Consumer — consome mensagens do OS Service quando uma OS é aceita
	executionQueueURL := os.Getenv("SQS_EXECUTION_QUEUE_URL")
	go func() {
		log.Printf("[SQS] Iniciando consumer na fila: %s", executionQueueURL)
		for {
			messages, err := sender.Receive(ctx, executionQueueURL, 10, 20)
			if err != nil {
				log.Printf("[SQS] Erro ao receber mensagens: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			for _, msg := range messages {
				var orderMsg OrderMessage
				if err := json.Unmarshal([]byte(*msg.Body), &orderMsg); err != nil {
					log.Printf("[SQS] Erro ao desserializar mensagem: %v", err)
					continue
				}
				log.Printf("[SQS] OS recebida para execução: %s", orderMsg.OrderID)
				if _, err := executionUseCase.CreateFromOrder(orderMsg.OrderID, orderMsg.MechanicID); err != nil {
					log.Printf("[SQS] Erro ao criar execução para OS %s: %v", orderMsg.OrderID, err)
					continue
				}
				if err := sender.Delete(ctx, executionQueueURL, msg.ReceiptHandle); err != nil {
					log.Printf("[SQS] Erro ao deletar mensagem da fila: %v", err)
				}
			}
		}
	}()

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("mechmanager-execution iniciado na porta :%s", port)
	r.Run(":" + port)
}
