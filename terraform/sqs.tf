terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  required_version = ">= 1.2.0"
}

provider "aws" {
  region = var.aws_region
}

# -----------------------------------------------------------
# Dead Letter Queues (DLQ) — armazena mensagens que falharam
# -----------------------------------------------------------

resource "aws_sqs_queue" "execution_queue_dlq" {
  name                      = "${var.execution_queue_name}-dlq"
  message_retention_seconds = 1209600 # 14 dias

  tags = {
    Service = "mechmanager-execution"
    Env     = var.environment
    Role    = "dlq"
  }
}

resource "aws_sqs_queue" "execution_complete_dlq" {
  name                      = "${var.execution_complete_queue_name}-dlq"
  message_retention_seconds = 1209600 # 14 dias

  tags = {
    Service = "mechmanager-execution"
    Env     = var.environment
    Role    = "dlq"
  }
}

# -----------------------------------------------------------
# Fila principal: OS Service → Execution Service
# Publicada pelo OS Service quando uma OS é aceita (aceite)
# Consumida pelo Execution Service para criar a execução
# -----------------------------------------------------------

resource "aws_sqs_queue" "execution_queue" {
  name                       = var.execution_queue_name
  visibility_timeout_seconds = 60
  message_retention_seconds  = 86400 # 1 dia
  receive_wait_time_seconds  = 20    # long polling

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.execution_queue_dlq.arn
    maxReceiveCount     = 3
  })

  tags = {
    Service = "mechmanager-execution"
    Env     = var.environment
    Role    = "input"
  }
}

# -----------------------------------------------------------
# Fila de conclusão: Execution Service → OS Service
# Publicada pelo Execution Service ao concluir ou falhar
# Consumida pelo OS Service para atualizar o status da OS
# Saga rollback: evento EXECUTION_FAILED reverte a OS
# -----------------------------------------------------------

resource "aws_sqs_queue" "execution_complete" {
  name                       = var.execution_complete_queue_name
  visibility_timeout_seconds = 60
  message_retention_seconds  = 86400 # 1 dia
  receive_wait_time_seconds  = 20    # long polling

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.execution_complete_dlq.arn
    maxReceiveCount     = 3
  })

  tags = {
    Service = "mechmanager-execution"
    Env     = var.environment
    Role    = "output"
  }
}
