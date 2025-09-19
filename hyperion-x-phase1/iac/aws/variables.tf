variable "aws_region" {
  description = "The AWS region to deploy resources in."
  type        = string
  default     = "us-east-1"
}

variable "cluster_name_prefix" {
  description = "A prefix to use for naming the EKS clusters."
  type        = string
  default     = "hyperion-x"
}