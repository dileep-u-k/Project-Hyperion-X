package cluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HyperionClusterSpec defines the desired state of a federated cluster.
type HyperionClusterSpec struct {
	Provider            string `json:"provider"`
	Region              string `json:"region"`
	APIEndpoint         string `json:"apiEndpoint"`
	KubeconfigSecretRef struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"kubeconfigSecretRef"`
}

// HyperionClusterStatus defines the observed state of a federated cluster.
type HyperionClusterStatus struct {
	Phase             string      `json:"phase"`
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime"`
}

// HyperionCluster is the Schema for the hyperionclusters API.
type HyperionCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HyperionClusterSpec   `json:"spec"`
	Status HyperionClusterStatus `json:"status,omitempty"`
}
