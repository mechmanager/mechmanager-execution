# --- SQS ---

output "execution_queue_url" {
  description = "URL da fila SQS de entrada (OS Service → Execution Service)"
  value       = aws_sqs_queue.execution_queue.url
}

output "execution_queue_arn" {
  value = aws_sqs_queue.execution_queue.arn
}

output "execution_complete_url" {
  description = "URL da fila SQS de saída (Execution Service → OS Service)"
  value       = aws_sqs_queue.execution_complete.url
}

output "execution_complete_arn" {
  value = aws_sqs_queue.execution_complete.arn
}

output "execution_queue_dlq_url" {
  description = "URL da DLQ da fila de entrada"
  value       = aws_sqs_queue.execution_queue_dlq.url
}

output "execution_complete_dlq_url" {
  description = "URL da DLQ da fila de saída"
  value       = aws_sqs_queue.execution_complete_dlq.url
}

# --- DynamoDB ---

output "dynamo_execution_table_name" {
  value = aws_dynamodb_table.execution_queue.name
}

output "dynamo_execution_table_arn" {
  value = aws_dynamodb_table.execution_queue.arn
}

# --- RDS ---

output "db_endpoint" {
  description = "Endpoint do RDS PostgreSQL"
  value       = aws_db_instance.execution_postgres.endpoint
}

output "db_name" {
  value = aws_db_instance.execution_postgres.db_name
}
