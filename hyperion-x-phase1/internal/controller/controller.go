// internal/controller/controller.go

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// GVR for our AIJob CRD
var gvr = schema.GroupVersionResource{
	Group:    "hyperion.ai",
	Version:  "v1alpha1",
	Resource: "aijobs",
}

type Controller struct {
	Dyn  dynamic.Interface
	Kube kubernetes.Interface
}

// Very small "polling" controller for Phase 1 demo simplicity.
// In production you'd use informers, leader election, proper status updates, etc.
func (c *Controller) Run(ctx context.Context) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := c.reconcileAll(ctx); err != nil {
				klog.Errorf("Reconciliation loop failed: %v", err)
			}
		}
	}
}

func (c *Controller) reconcileAll(ctx context.Context) error {
	list, err := c.Dyn.Resource(gvr).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, u := range list.Items {
		// Use a copy to avoid issues with loop variable
		obj := u
		if err := c.reconcileOne(ctx, &obj); err != nil {
			klog.Errorf("Failed to reconcile %s/%s: %v", u.GetNamespace(), u.GetName(), err)
		}
	}
	return nil
}

func (c *Controller) reconcileOne(ctx context.Context, u *unstructured.Unstructured) error {
	// Convert unstructured → typed AIJob
	b, _ := json.Marshal(u.Object)
	var job AIJob
	if err := json.Unmarshal(b, &job); err != nil {
		return err
	}

	// 1. LIST existing pods for this AIJob
	sel := fmt.Sprintf("hyperion.ai/aijob=%s", job.Name)
	pl, err := c.Kube.CoreV1().Pods(job.Namespace).List(ctx, metav1.ListOptions{LabelSelector: sel})
	if err != nil {
		return err
	}

	// 2. COUNT pods by their phase to determine the current state
	var running, succeeded, failed int32
	for _, p := range pl.Items {
		switch p.Status.Phase {
		case corev1.PodRunning:
			running++
		case corev1.PodSucceeded:
			succeeded++
		case corev1.PodFailed:
			failed++
		}
	}

	// 3. DETERMINE the new status based on pod counts
	newStatus := AIJobStatus{
		RunningPods:   running,
		SucceededPods: succeeded,
	}

	// Logic to set the overall job phase
	if failed > 0 {
		newStatus.Phase = "Failed"
	} else if succeeded == job.Spec.Parallelism {
		newStatus.Phase = "Succeeded"
	} else if running > 0 {
		newStatus.Phase = "Running"
	} else {
		newStatus.Phase = "Pending"
	}

	// 4. UPDATE STATUS if it has changed. This is critical to avoid unnecessary API calls.
	if !reflect.DeepEqual(job.Status, newStatus) {
		klog.Infof("Updating status for AIJob %s/%s: Phase=%s, Running=%d, Succeeded=%d",
			job.Namespace, job.Name, newStatus.Phase, newStatus.RunningPods, newStatus.SucceededPods)

		// Get the latest version of the object to avoid conflicts
		latestU, err := c.Dyn.Resource(gvr).Namespace(job.Namespace).Get(ctx, job.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// Convert the new status to a map[string]interface{}
		statusMap, err := json.Marshal(newStatus)
		if err != nil {
			return err
		}
		var statusInterface map[string]interface{}
		if err := json.Unmarshal(statusMap, &statusInterface); err != nil {
			return err
		}

		// Set the status field on the unstructured object
		if err := unstructured.SetNestedField(latestU.Object, statusInterface, "status"); err != nil {
			return err
		}

		// Perform the dedicated /status subresource update
		_, err = c.Dyn.Resource(gvr).Namespace(job.Namespace).UpdateStatus(ctx, latestU, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update AIJob status: %w", err)
		}
	}

	// 5. CREATE missing replicas only if the job is not finished
	if newStatus.Phase == "Failed" || newStatus.Phase == "Succeeded" {
		return nil
	}

	have := int32(len(pl.Items))
	need := job.Spec.Parallelism
	if have < need {
		klog.Infof("AIJob %s/%s needs %d pods, has %d. Creating one...", job.Namespace, job.Name, need, have)
		p := BuildPod(&job, have)
		_, err := c.Kube.CoreV1().Pods(job.Namespace).Create(ctx, p, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}
