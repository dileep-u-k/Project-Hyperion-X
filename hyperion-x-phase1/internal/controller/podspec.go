package controller

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func BuildPod(job *AIJob, idx int32) *corev1.Pod {
	name := job.Name
	if job.Spec.Parallelism > 1 {
		name = fmt.Sprintf("%s-%d", job.Name, idx)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: job.Namespace,
			Labels: map[string]string{
				"app":                   "hyperion-aijob",
				"hyperion.ai/scheduler": "hyperion-scheduler",
				"hyperion.ai/aijob":     job.Name,
			},
			Annotations: job.Spec.Annotations,
		},
		Spec: corev1.PodSpec{
			SchedulerName: "hyperion-scheduler",
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{{
				Name:    "worker",
				Image:   job.Spec.Image,
				Command: job.Spec.Command,
				Resources: corev1.ResourceRequirements{
					Requests: toResourceList(job),
					Limits:   toResourceList(job),
				},
			}},
		},
	}
	return pod
}

func toResourceList(job *AIJob) corev1.ResourceList {
	rl := corev1.ResourceList{}
	if job.Spec.Resources.CPU != "" {
		rl[corev1.ResourceCPU] = resource.MustParse(job.Spec.Resources.CPU)
	}
	if job.Spec.Resources.Memory != "" {
		rl[corev1.ResourceMemory] = resource.MustParse(job.Spec.Resources.Memory)
	}
	if job.Spec.Resources.NvidiaGPU != nil && *job.Spec.Resources.NvidiaGPU > 0 {
		rl["nvidia.com/gpu"] = resource.MustParse(fmt.Sprintf("%d", *job.Spec.Resources.NvidiaGPU))
	}
	return rl
}
