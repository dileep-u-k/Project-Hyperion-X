package controller

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type AIJobSpec struct {
	Image       string            `json:"image"`
	Command     []string          `json:"command"`
	Resources   ResourcesSpec     `json:"resources"`
	Parallelism int32             `json:"parallelism"`
	Priority    string            `json:"priority"`
	Annotations map[string]string `json:"annotations"`
}

// Add this struct
type AIJobStatus struct {
	Phase         string `json:"phase"` // e.g., Pending, Running, Succeeded, Failed
	RunningPods   int32  `json:"runningPods"`
	SucceededPods int32  `json:"succeededPods"`
}

type ResourcesSpec struct {
	CPU       string `json:"cpu"`
	Memory    string `json:"memory"`
	NvidiaGPU *int32 `json:"nvidiaGpu"`
}

type AIJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AIJobSpec   `json:"spec"`
	Status            AIJobStatus `json:"status,omitempty"` // <-- ADD THIS
}
