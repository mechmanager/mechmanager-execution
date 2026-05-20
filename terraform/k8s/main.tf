# VPC existente
data "aws_vpc" "this" {
  id = "vpc-0ec0bef1e5e730f53"
}

# Subnets já criadas (exemplo: duas para EKS)
data "aws_subnet" "eks_a" {
  id = "subnet-071b331ef4bb16031" # us-east-1d
}

data "aws_subnet" "eks_b" {
  id = "subnet-0e6d0f3ea13ba4754" # us-east-1b
}

# Security group para o cluster EKS
resource "aws_security_group" "eks_cluster_sg" {
  name        = "${var.cluster_name}-cluster-sg"
  vpc_id      = data.aws_vpc.this.id
  description = "EKS cluster security group"

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# Usar role existente LabRole
data "aws_iam_role" "lab" {
  name = "LabRole"
}

# EKS Cluster usando subnets existentes
resource "aws_eks_cluster" "this" {
  name     = var.cluster_name
  role_arn = data.aws_iam_role.lab.arn
  version  = "1.30"

  vpc_config {
    subnet_ids              = [data.aws_subnet.eks_a.id, data.aws_subnet.eks_b.id]
    endpoint_private_access = false
    endpoint_public_access  = true
    public_access_cidrs     = ["0.0.0.0/0"]
  }
}

# Managed node group
resource "aws_eks_node_group" "default" {
  cluster_name    = aws_eks_cluster.this.name
  node_group_name = "${var.cluster_name}-ng"
  node_role_arn   = data.aws_iam_role.lab.arn
  subnet_ids      = [data.aws_subnet.eks_a.id, data.aws_subnet.eks_b.id]

  scaling_config {
    desired_size = var.node_min_size
    max_size     = var.node_max_size
    min_size     = var.node_min_size
  }

  instance_types = [var.node_instance_type]
  capacity_type  = "ON_DEMAND"

  depends_on = [aws_eks_cluster.this]
}
