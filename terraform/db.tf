# VPC existente
data "aws_vpc" "execution_vpc" {
  id = "vpc-0ec0bef1e5e730f53"
}

# Subnets existentes (exemplo: duas em AZs diferentes)
data "aws_subnet" "db_a" {
  id = "subnet-071b331ef4bb16031" # us-east-1d
}

data "aws_subnet" "db_b" {
  id = "subnet-0e6d0f3ea13ba4754" # us-east-1b
}

# Security Group do banco — tráfego aberto
resource "aws_security_group" "execution_db_sg" {
  name        = "execution-db-sg"
  description = "Allow Postgres access"
  vpc_id      = data.aws_vpc.execution_vpc.id

  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # aberto
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "execution-db-sg"
  }
}

# DB Subnet Group usando subnets existentes
resource "aws_db_subnet_group" "execution_db_subnet_group" {
  name       = "execution-db-subnet-group"
  subnet_ids = [data.aws_subnet.db_a.id, data.aws_subnet.db_b.id]

  tags = {
    Name = "execution-db-subnet-group"
  }
}

# Parameter Group
resource "aws_db_parameter_group" "execution_postgres_params" {
  name        = "execution-postgres-params"
  family      = "postgres18" # mantém como você definiu
  description = "Parameter group para mechmanager-execution"

  parameter {
    name  = "rds.force_ssl"
    value = "0"
  }
}

# RDS Instance
resource "aws_db_instance" "execution_postgres" {
  identifier           = var.db_identifier
  allocated_storage    = 20
  engine               = "postgres"
  instance_class       = "db.t3.micro"
  db_name              = var.db_name
  username             = var.db_username
  password             = var.db_password
  skip_final_snapshot  = true
  publicly_accessible  = true

  vpc_security_group_ids = [aws_security_group.execution_db_sg.id]
  db_subnet_group_name   = aws_db_subnet_group.execution_db_subnet_group.name
  parameter_group_name   = aws_db_parameter_group.execution_postgres_params.name

  tags = {
    Name    = "execution-db"
    Service = "mechmanager-execution"
    Env     = var.environment
  }
}
