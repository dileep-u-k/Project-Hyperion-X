// internal/scheduler/scoring.go

package scheduler

import (
	"context"
	"sort"

	"github.com/dileep-u-k/hyperion-x-phase1/internal/metrics"
	corev1 "k8s.io/api/core/v1"
	klog "k8s.io/klog/v2"
)

type ScoringPolicy string

const (
	LeastLoaded ScoringPolicy = "leastLoaded" // prefer low CPU/mem usage
	BinPack     ScoringPolicy = "binPack"     // prefer tighter packing
)

type Scorer struct {
	Metrics *metrics.Client
	Policy  ScoringPolicy
}

type candidate struct {
	Node  corev1.Node
	Score float64
}

func (s *Scorer) ScoreNodes(ctx context.Context, nodes []corev1.Node, podsOnNode map[string]int) []candidate {
	cands := make([]candidate, 0, len(nodes))

	for _, n := range nodes {
		node := n // Create a local copy for the loop
		var ip string
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				ip = addr.Address
				break
			}
		}
		if ip == "" {
			klog.Warningf("Node %s has no InternalIP, skipping", node.Name)
			continue
		}

		// Fetch real-time node metrics from the agent
		m, err := s.Metrics.Get(ctx, ip)
		var score float64

		if err != nil {
			// **HANDLE METRIC FAILURES GRACEFULLY**
			// Instead of skipping, log a warning and assign a penalty score.
			// This makes the node least preferred but still available in an emergency.
			klog.Warningf("Failed to get metrics for node %s (%s): %v. Assigning penalty score.", node.Name, ip, err)
			score = -1000.0 // A large negative number to rank it last
		} else {
			// **CALCULATE SCORE FOR HEALTHY NODES**
			// Simple scoring policies
			switch s.Policy {
			case LeastLoaded:
				score = (100.0 - m.CPUUsagePct) + (100.0 - m.MemUsagePct)
			case BinPack:
				// Score is higher for nodes with higher utilization
				score = m.CPUUsagePct + m.MemUsagePct
			default:
				score = (100.0 - m.CPUUsagePct)
			}
		}

		// Light penalty to spread pods, less impactful than the main score
		score -= float64(podsOnNode[node.Name]) * 5.0

		cands = append(cands, candidate{Node: node, Score: score})
	}

	// Sort candidates from highest score to lowest
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].Score > cands[j].Score })
	return cands
}
