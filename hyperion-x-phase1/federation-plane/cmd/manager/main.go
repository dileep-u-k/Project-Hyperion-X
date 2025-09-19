package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"

	"github.com/dileep-u-k/hyperion-x-phase1/federation-plane/internal/cluster"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()
	defer klog.Flush()

	// This controller runs on a "management" cluster (e.g., your local minikube).
	// It uses the local kubeconfig to connect to the management cluster's API.
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error creating Kubernetes client: %v", err)
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error creating dynamic client: %v", err)
	}

	ctl := cluster.NewController(kubeClient, dynClient)
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := ctl.Run(ctx); err != nil {
		klog.Fatalf("Controller failed: %v", err)
	}
}
