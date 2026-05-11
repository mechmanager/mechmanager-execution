# mechmanager-execution

Microserviço responsável pelo gerenciamento da fila de execução de Ordens de Serviço (OS) da oficina mecânica. Faz parte do ecossistema **MechManager** — FIAP Tech Challenge.

## Responsabilidades

- Consumir OS aceitas publicadas pelo **OS Service** via SQS
- Gerenciar a fila de execução com estados progressivos (Saga Pattern)
- Persistir o histórico completo no PostgreSQL
- Manter o estado em tempo real no DynamoDB
- Notificar o OS Service ao concluir ou falhar uma execução via SQS

## Arquitetura

```
OS Service ──SQS──► [execution-queue] ──► mechmanager-execution
                                                  │
                                    ┌─────────────┼─────────────┐
                                    ▼             ▼             ▼
                               PostgreSQL     DynamoDB     [execution-complete] ──SQS──► OS Service
                            (histórico)   (fila em tempo real)
```

### Padrões utilizados

- **Arquitetura Hexagonal** (Ports & Adapters)
- **Saga Pattern** — máquina de estados com evento de compensação em caso de falha
- **DDD** — domínio isolado sem dependência de frameworks

### Máquina de estados (Saga)

```
QUEUED ──► IN_DIAGNOSIS ──► IN_REPAIR ──► COMPLETED
   └──────────────┴──────────────┴──────► FAILED (rollback)
```

## Stack

| Camada | Tecnologia |
|---|---|
| Linguagem | Go 1.25 |
| HTTP | Gin |
| ORM | GORM + gormigrate |
| SQL | PostgreSQL 15 |
| NoSQL | AWS DynamoDB |
| Mensageria | AWS SQS |
| Infra | Terraform |
| Container | Docker / docker-compose |
| Orquestração | Kubernetes (EKS) |
| CI/CD | GitHub Actions + GHCR |

## Como rodar localmente

### Pré-requisitos

- Go 1.22+
- Docker Desktop
- Credenciais AWS válidas (AWS Academy)

### 1. Configure o `.env`

```bash
cp src/.env.example src/.env
# Edite com suas credenciais AWS e dados do banco
```

Variáveis necessárias:

```env
DATABASE_DSN=host=localhost port=5433 user=postgres password=postgres dbname=mechmanager_execution sslmode=disable
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=mechmanager_execution

AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
AWS_SESSION_TOKEN=...

DYNAMO_EXECUTION_TABLE=execution-queue
SQS_EXECUTION_QUEUE_URL=https://sqs.us-east-1.amazonaws.com/<account-id>/execution-queue
SQS_EXECUTION_COMPLETE_URL=https://sqs.us-east-1.amazonaws.com/<account-id>/execution-complete

GIN_MODE=debug
CORS_ALLOW_ORIGINS=http://localhost,http://localhost:3000
```

### 2. Suba o banco e a aplicação

```bash
cd src

# Inicia o PostgreSQL
docker compose up db -d

# Inicia o serviço (migrations rodam automaticamente)
go run ./cmd/api
```

## Endpoints

### Health check

```
GET /health
```

### Execuções

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/executions` | Lista todas as execuções |
| `GET` | `/executions/:id` | Busca execução por ID |
| `GET` | `/executions/order/:id` | Busca execução por Order ID |
| `PATCH` | `/executions/:id/status` | Avança o status (QUEUED→IN_DIAGNOSIS→IN_REPAIR) |
| `PATCH` | `/executions/:id/complete` | Conclui a execução (IN_REPAIR→COMPLETED) |
| `PATCH` | `/executions/:id/fail` | Registra falha e aciona compensação Saga |

### Exemplos

**Avançar status:**
```bash
curl -X PATCH http://localhost:8080/executions/{id}/status \
  -H "Content-Type: application/json" \
  -d '{"status":"IN_DIAGNOSIS","diagnostic_notes":"Motor com barulho anormal"}'
```

**Concluir execução:**
```bash
curl -X PATCH http://localhost:8080/executions/{id}/complete \
  -H "Content-Type: application/json" \
  -d '{"repair_notes":"Correia dentada substituída com sucesso"}'
```

**Registrar falha (Saga rollback):**
```bash
curl -X PATCH http://localhost:8080/executions/{id}/fail \
  -H "Content-Type: application/json" \
  -d '{"reason":"Peça de reposição indisponível"}'
```

## Infraestrutura (Terraform)

```bash
cd terraform

# Inicializar e provisionar
terraform init
terraform apply

# Outputs: SQS URLs, DynamoDB table name, RDS endpoint
terraform output
```

Recursos provisionados:
- SQS `execution-queue` + DLQ
- SQS `execution-complete` + DLQ
- DynamoDB `execution-queue` (com GSI por status)
- RDS PostgreSQL `mechmanager_execution`

## Testes

```bash
cd src
go test ./...
```

## CI/CD

O pipeline GitHub Actions executa em todo push:

1. `gofmt` — formatação
2. `go vet` — análise estática
3. `go test ./...` — testes unitários
4. Build e push da imagem Docker para GHCR
5. Deploy no EKS com rollback automático em caso de falha

## Comunicação entre serviços

| Direção | Fila | Evento |
|---|---|---|
| OS Service → Execution | `execution-queue` | OS aceita |
| Execution → OS Service | `execution-complete` | `EXECUTION_COMPLETED` ou `EXECUTION_FAILED` |
