# -----------------------------------------------------------
# DynamoDB — fila de execução em tempo real (NoSQL)
# Armazena o estado atual de cada OS na fila de execução
# Permite consultas rápidas por status sem ir ao PostgreSQL
# -----------------------------------------------------------

resource "aws_dynamodb_table" "execution_queue" {
  name         = var.dynamo_execution_table
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "order_id"

  attribute {
    name = "order_id"
    type = "S"
  }

  attribute {
    name = "status"
    type = "S"
  }

  attribute {
    name = "enqueued_at"
    type = "N"
  }

  # GSI para buscar itens por status (ex: todos QUEUED, todos IN_REPAIR)
  global_secondary_index {
    name            = "GSI_StatusEnqueuedAt"
    hash_key        = "status"
    range_key       = "enqueued_at"
    projection_type = "ALL"
  }

  server_side_encryption {
    enabled = true
  }

  point_in_time_recovery {
    enabled = true
  }

  tags = {
    Service = "mechmanager-execution"
    Env     = var.environment
  }
}
