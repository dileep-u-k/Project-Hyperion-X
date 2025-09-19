# Using the official Terraform AWS EKS module to simplify cluster creation.
module "eks_cluster_a" {
  source  = "terraform-aws-modules/eks/aws"
  version = "20.8.4"

  cluster_name    = "${var.cluster_name_prefix}-a"
  cluster_version = "1.29"

  vpc_id     = module.vpc.vpc_id
  subnet_ids = [module.vpc.private_subnets[0]] # Deploy in the first AZ

  eks_managed_node_groups = {
    one = {
      min_size     = 1
      max_size     = 2
      desired_size = 1

      instance_types = ["t3.medium"]
    }
  }

  tags = {
    "Project" = "Hyperion-X"
  }
}

module "eks_cluster_b" {
  source  = "terraform-aws-modules/eks/aws"
  version = "20.8.4"

  cluster_name    = "${var.cluster_name_prefix}-b"
  cluster_version = "1.29"

  vpc_id     = module.vpc.vpc_id
  subnet_ids = [module.vpc.private_subnets[1]] # Deploy in the second AZ

  eks_managed_node_groups = {
    one = {
      min_size     = 1
      max_size     = 2
      desired_size = 1

      instance_types = ["t3.medium"]
    }
  }

  tags = {
    "Project" = "Hyperion-X"
  }
}