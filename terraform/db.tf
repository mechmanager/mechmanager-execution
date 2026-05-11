# -----------------------------------------------------------
# PostgreSQL (RDS) — banco relacional (SQL)
# Armazena o histórico completo de execuções com todas as
# transições de status, notas de diagnóstico e reparo
# -----------------------------------------------------------

# VPC
resource "aws_vpc" "execution_vpc" {
  cidr_block           = "10.2.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name    = "execution-vpc"
    Service = "mechmanager-execution"
  }
}

# Subnets
resource "aws_subnet" "execution_public_a" {
  vpc_id                  = aws_vpc.execution_vpc.id
  cidr_block              = "10.2.1.0/24"
  availability_zone       = "us-east-1a"
  map_public_ip_on_launch = true

  tags = {
    Name = "execution-public-subnet-a"
  }
}

resource "aws_subnet" "execution_public_b" {
  vpc_id                  = aws_vpc.execution_vpc.id
  cidr_block              = "10.2.2.0/24"
  availability_zone       = "us-east-1b"
  map_public_ip_on_launch = true

  tags = {
    Name = "execution-public-subnet-b"
  }
}

# Internet Gateway
resource "aws_internet_gateway" "execution_gw" {
  vpc_id = aws_vpc.execution_vpc.id

  tags = {
    Name = "execution-gw"
  }
}

# Route Table
resource "aws_route_table" "execution_public_rt" {
  vpc_id = aws_vpc.execution_vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.execution_gw.id
  }

  tags = {
    Name = "execution-public-rt"
  }
}

resource "aws_route_table_association" "execution_a" {
  subnet_id      = aws_subnet.execution_public_a.id
  route_table_id = aws_route_table.execution_public_rt.id
}

resource "aws_route_table_association" "execution_b" {
  subnet_id      = aws_subnet.execution_public_b.id
  route_table_id = aws_route_table.execution_public_rt.id
}

# Security Group
resource "aws_security_group" "execution_db_sg" {
  name        = "execution-db-sg"
  description = "Allow Postgres access"
  vpc_id      = aws_vpc.execution_vpc.id

  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
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

# DB Subnet Group
resource "aws_db_subnet_group" "execution_db_subnet_group" {
  name       = "execution-db-subnet-group"
  subnet_ids = [aws_subnet.execution_public_a.id, aws_subnet.execution_public_b.id]

  tags = {
    Name = "execution-db-subnet-group"
  }
}

# Parameter Group
resource "aws_db_parameter_group" "execution_postgres_params" {
  name        = "execution-postgres-params"
  family      = "postgres18"
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
