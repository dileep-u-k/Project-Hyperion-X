package scheduler

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// Listers wraps simple list/watch helpers used by the scheduler.
type Listers struct {
	Client kubernetes.Interface
}

func (l *Listers) ListSchedulableNodes(ctx context.Context) ([]corev1.Node, error) {
	nl, err := l.Client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]corev1.Node, 0, len(nl.Items))
	for _, n := range nl.Items {
		if n.Spec.Unschedulable {
			continue
		}
		out = append(out, n)
	}
	return out, nil
}

func (l *Listers) WatchPendingPods(ctx context.Context, schedulerName string) (watch.Interface, error) {
	fieldSelector := fields.OneTermEqualSelector("status.phase", string(corev1.PodPending))
	// We match on a label our controller sets and/or SchedulerName in the Pod spec.
	return l.Client.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
		LabelSelector: labels.Set{"hyperion.ai/scheduler": schedulerName}.String(),
	})
}
