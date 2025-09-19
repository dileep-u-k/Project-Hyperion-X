package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/dileep-u-k/hyperion-x-phase1/internal/controller"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func restConfig() (*rest.Config, error) {
	if kube := os.Getenv("KUBECONFIG"); kube != "" {
		return clientcmd.BuildConfigFromFlags("", kube)
	}
	return rest.InClusterConfig()
}

func main() {
	// Initialize klog. It's essential for all Kubernetes components.
	klog.InitFlags(nil)
	flag.Parse()
	defer klog.Flush()

	klog.Info("Starting Hyperion-X controller...")

	cfg, err := restConfig()
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %v", err)
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error creating dynamic client: %v", err)
	}
	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error creating kubernetes client: %v", err)
	}

	ctl := &controller.Controller{Dyn: dyn, Kube: cli}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := ctl.Run(ctx); err != nil {
		klog.Fatalf("Controller failed to run: %v", err)
	}
	klog.Info("Controller shut down gracefully.")
}
