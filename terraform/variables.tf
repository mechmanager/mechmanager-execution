variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "environment" {
  type    = string
  default = "prod"
}

# --- SQS ---

variable "execution_queue_name" {
  type        = string
  default     = "execution-queue"
  description = "Fila consumida pelo Execution Service ao receber OS aceitas do OS Service"
}

variable "execution_complete_queue_name" {
  type        = string
  default     = "execution-complete"
  description = "Fila publicada pelo Execution Service ao concluir ou falhar uma execução"
}

# --- DynamoDB ---

variable "dynamo_execution_table" {
  type        = string
  default     = "execution-queue"
  description = "Tabela DynamoDB para estado em tempo real da fila de execução"
}

# --- RDS ---

variable "db_identifier" {
  type    = string
  default = "execution-db-v1"
}

variable "db_name" {
  type    = string
  default = "mechmanager_execution"
}

variable "db_username" {
  type    = string
  default = "meu_usuario"
}

variable "db_password" {
  type      = string
  sensitive = true
  default   = "minha_senha"
}
