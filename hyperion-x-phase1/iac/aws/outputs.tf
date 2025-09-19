output "cluster_a_name" {
  description = "The name of the first EKS cluster."
  value       = module.eks_cluster_a.cluster_name
}

output "cluster_a_kubeconfig" {
  description = "The kubeconfig for the first EKS cluster."
  value       = module.eks_cluster_a.kubeconfig
  sensitive   = true
}

output "cluster_b_name" {
  description = "The name of the second EKS cluster."
  value       = module.eks_cluster_b.cluster_name
}

output "cluster_b_kubeconfig" {
  description = "The kubeconfig for the second EKS cluster."
  value       = module.eks_cluster_b.kubeconfig
  sensitive   = true
}