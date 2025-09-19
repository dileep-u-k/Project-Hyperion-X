package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/dileep-u-k/hyperion-x-phase1/internal/metrics"
	"github.com/dileep-u-k/hyperion-x-phase1/internal/scheduler"
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
	klog.InitFlags(nil)

	var policy string
	var name string
	flag.StringVar(&policy, "policy", "leastLoaded", "scoring policy: leastLoaded|binPack")
	flag.StringVar(&name, "name", "hyperion-scheduler", "scheduler name")
	flag.Parse()
	defer klog.Flush()

	klog.Infof("Starting Hyperion-X scheduler with policy '%s'...", policy)

	cfg, err := restConfig()
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error creating kubernetes client: %v", err)
	}

	mc := metrics.New()
	scorer := &scheduler.Scorer{Metrics: mc, Policy: scheduler.ScoringPolicy(policy)}
	sch := scheduler.New(clientset, name, scorer)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Readiness endpoint for K8s
	go func() {
		http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
		if err := http.ListenAndServe(":8080", nil); err != nil {
			klog.Errorf("Healthz endpoint failed: %v", err)
		}
	}()

	if err := sch.Run(ctx); err != nil {
		klog.Fatalf("Scheduler failed to run: %v", err)
	}
	klog.Info("Scheduler shut down gracefully.")
}
