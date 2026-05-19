variable "cluster_name" {
  type    = string
  default = "mechmanager-execution"
}

variable "region" {
  type    = string
  default = "us-east-1"
}

variable "node_min_size" {
  type    = number
  default = 1
}

variable "node_max_size" {
  type    = number
  default = 2
}

variable "node_instance_type" {
  type    = string
  default = "t3.medium"
}