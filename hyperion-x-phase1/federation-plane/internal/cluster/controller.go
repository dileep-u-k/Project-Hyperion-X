package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var gvr = schema.GroupVersionResource{
	Group:    "hyperion.ai",
	Version:  "v1alpha1",
	Resource: "hyperionclusters",
}

type ClusterInfo struct {
	Status    string
	Clientset kubernetes.Interface
}

type Controller struct {
	// Client for the management cluster where this controller runs.
	MgmtKubeClient kubernetes.Interface
	MgmtDynClient  dynamic.Interface

	// Live map of connected remote clusters.
	mu                sync.RWMutex
	connectedClusters map[string]ClusterInfo
}

func NewController(mgmtKube kubernetes.Interface, mgmtDyn dynamic.Interface) *Controller {
	return &Controller{
		MgmtKubeClient:    mgmtKube,
		MgmtDynClient:     mgmtDyn,
		connectedClusters: make(map[string]ClusterInfo),
	}
}

func (c *Controller) Run(ctx context.Context) error {
	// NOTE: For Sprint 1 simplicity, we use a polling reconciler.
	// A production system would use informers and a workqueue for efficiency.
	reconcileTicker := time.NewTicker(10 * time.Second)
	defer reconcileTicker.Stop()

	// Ticker for the showcase status output.
	statusTicker := time.NewTicker(5 * time.Second)
	defer statusTicker.Stop()

	klog.Info("Cluster Controller starting...")
	c.reconcile(ctx) // Initial reconciliation

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-reconcileTicker.C:
			c.reconcile(ctx)
		case <-statusTicker.C:
			c.printStatus()
		}
	}
}

// reconcile attempts to connect to all registered HyperionCluster resources.
func (c *Controller) reconcile(ctx context.Context) {
	klog.Info("Reconciling all HyperionClusters...")
	list, err := c.MgmtDynClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		klog.Errorf("Failed to list HyperionClusters: %v", err)
		return
	}

	for _, u := range list.Items {
		var hc HyperionCluster
		b, _ := json.Marshal(u.Object)
		if err := json.Unmarshal(b, &hc); err != nil {
			klog.Errorf("Failed to unmarshal HyperionCluster %s: %v", u.GetName(), err)
			continue
		}

		go c.connectToCluster(ctx, hc)
	}
}

func (c *Controller) connectToCluster(ctx context.Context, hc HyperionCluster) {
	clusterName := hc.Name
	secretRef := hc.Spec.KubeconfigSecretRef

	// 1. Get the Secret containing the kubeconfig for the remote cluster.
	secret, err := c.MgmtKubeClient.CoreV1().Secrets(secretRef.Namespace).Get(ctx, secretRef.Name, metav1.GetOptions{})
	if err != nil {
		klog.Warningf("Failed to get secret %s/%s for cluster %s: %v", secretRef.Namespace, secretRef.Name, clusterName, err)
		c.updateStatus(clusterName, "Error: SecretNotFound", nil)
		return
	}

	kubeconfigBytes, ok := secret.Data["kubeconfig"]
	if !ok {
		klog.Warningf("Secret %s/%s for cluster %s is missing 'kubeconfig' data", secretRef.Namespace, secretRef.Name, clusterName)
		c.updateStatus(clusterName, "Error: InvalidSecret", nil)
		return
	}

	// 2. Build a client-go clientset from the kubeconfig bytes.
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		klog.Warningf("Failed to build REST config for cluster %s: %v", clusterName, err)
		c.updateStatus(clusterName, "Error: KubeconfigParse", nil)
		return
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		klog.Warningf("Failed to create clientset for cluster %s: %v", clusterName, err)
		c.updateStatus(clusterName, "Error: ClientCreationFailed", nil)
		return
	}

	// 3. Verify the connection by making a simple API call.
	_, err = clientset.Discovery().ServerVersion()
	if err != nil {
		klog.Warningf("Failed to get server version for cluster %s: %v", clusterName, err)
		c.updateStatus(clusterName, "Offline", nil)
		return
	}

	klog.Infof("Successfully connected to cluster: %s", clusterName)
	c.updateStatus(clusterName, "Online", clientset)
}

func (c *Controller) updateStatus(name, status string, clientset kubernetes.Interface) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connectedClusters[name] = ClusterInfo{
		Status:    status,
		Clientset: clientset,
	}
}

func (c *Controller) printStatus() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fmt.Println("\n--- CLUSTER CONNECTION STATUS ---")
	if len(c.connectedClusters) == 0 {
		fmt.Println("No clusters registered or connected yet.")
	}
	for name, info := range c.connectedClusters {
		fmt.Printf("- %s: %s\n", name, info.Status)
	}
	fmt.Println("-------------------------------")
}
