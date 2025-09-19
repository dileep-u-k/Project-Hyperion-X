// internal/scheduler/scheduler.go

package scheduler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	klog "k8s.io/klog/v2"
)

// Scheduler watches Pending pods for us and binds them to chosen nodes.
type Scheduler struct {
	Kube   kubernetes.Interface
	Lst    *Listers
	Scorer *Scorer
	Name   string
}

func New(k kubernetes.Interface, name string, scorer *Scorer) *Scheduler {
	return &Scheduler{Kube: k, Lst: &Listers{Client: k}, Scorer: scorer, Name: name}
}

func (s *Scheduler) Run(ctx context.Context) error {
	w, err := s.Lst.WatchPendingPods(ctx, s.Name)
	if err != nil {
		return err
	}
	defer w.Stop()
	klog.Info("hyperion-scheduler running…")
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-w.ResultChan():
			pod, ok := ev.Object.(*corev1.Pod)
			if !ok || pod == nil {
				continue
			}

			// This is the idiomatic, correct check. Only handle pods that
			// explicitly request this scheduler by name.
			if pod.Spec.SchedulerName != s.Name {
				continue
			}

			// Ignore pods that are already assigned to a node
			if pod.Spec.NodeName != "" {
				continue
			}

			if err := s.scheduleOne(ctx, pod); err != nil {
				klog.Errorf("schedule failed for %s/%s: %v", pod.Namespace, pod.Name, err)
			}
		}
	}
}

func (s *Scheduler) scheduleOne(ctx context.Context, pod *corev1.Pod) error {
	nodes, err := s.Lst.ListSchedulableNodes(ctx)
	if err != nil {
		return err
	}
	if len(nodes) == 0 {
		return fmt.Errorf("no nodes available")
	}

	// === PREDICATE STAGE ===
	// Filter out nodes that cannot satisfy the pod's resource requests.
	filteredNodes := s.filterNodes(pod, nodes)
	if len(filteredNodes) == 0 {
		return fmt.Errorf("no nodes found that can satisfy pod resource requests")
	}

	// Precompute simple packing count (how many pods already on each node)
	podsOnNode := map[string]int{}
	pl, _ := s.Kube.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	for _, p := range pl.Items {
		if p.Spec.NodeName != "" {
			podsOnNode[p.Spec.NodeName]++
		}
	}

	// === PRIORITY STAGE ===
	// Score the viable nodes.
	ranked := s.Scorer.ScoreNodes(ctx, filteredNodes, podsOnNode)
	if len(ranked) == 0 {
		return fmt.Errorf("no viable nodes after scoring (metrics might be missing from all)")
	}

	target := ranked[0].Node
	klog.Infof("binding %s/%s -> node %s (score=%.2f)", pod.Namespace, pod.Name, target.Name, ranked[0].Score)

	// Use the Binding subresource (RBAC: create on pods/binding)
	b := &corev1.Binding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			UID:       pod.UID,
		},
		Target: corev1.ObjectReference{Kind: "Node", Name: target.Name},
	}
	return s.Kube.CoreV1().Pods(pod.Namespace).Bind(ctx, b, metav1.CreateOptions{})
}

// filterNodes checks which nodes can satisfy the resource requests of a pod.
func (s *Scheduler) filterNodes(pod *corev1.Pod, nodes []corev1.Node) []corev1.Node {
	var viableNodes []corev1.Node
	podRequests := getPodResourceRequests(pod)

	for _, node := range nodes {
		if nodeFits(podRequests, &node) {
			viableNodes = append(viableNodes, node)
		}
	}
	return viableNodes
}

// getPodResourceRequests calculates the total resource requests for all containers in a pod.
func getPodResourceRequests(pod *corev1.Pod) corev1.ResourceList {
	requests := corev1.ResourceList{}
	for _, container := range pod.Spec.Containers {
		for name, quantity := range container.Resources.Requests {
			if existing, ok := requests[name]; ok {
				existing.Add(quantity)
				requests[name] = existing
			} else {
				requests[name] = quantity
			}
		}
	}
	return requests
}

// nodeFits checks if a node's allocatable resources are sufficient for the pod's requests.
func nodeFits(requests corev1.ResourceList, node *corev1.Node) bool {
	allocatable := node.Status.Allocatable
	for name, quantity := range requests {
		if nodeQuantity, ok := allocatable[name]; !ok || quantity.Cmp(nodeQuantity) > 0 {
			klog.V(4).Infof("Node %s is not viable: missing resource %s or insufficient quantity", node.Name, name)
			return false
		}
	}
	return true
}
