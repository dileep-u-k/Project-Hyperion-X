package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// cacheEntry holds the metrics and the time they were fetched.
type cacheEntry struct {
	metrics   *NodeMetrics
	timestamp time.Time
}

// Client fetches metrics from a node agent, with a simple in-memory cache.
type Client struct {
	httpClient *http.Client
	cache      map[string]cacheEntry
	mu         sync.RWMutex
	ttl        time.Duration
}

// New creates a new metrics client with a cache.
func New() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 600 * time.Millisecond},
		cache:      make(map[string]cacheEntry),
		ttl:        5 * time.Second, // Cache metrics for 5 seconds
	}
}

// Get retrieves metrics for a given node, checking the cache first.
func (c *Client) Get(ctx context.Context, nodeIP string) (*NodeMetrics, error) {
	// Check cache first (with a read lock)
	c.mu.RLock()
	entry, found := c.cache[nodeIP]
	c.mu.RUnlock()

	if found && time.Since(entry.timestamp) < c.ttl {
		klog.V(5).Infof("Metrics cache HIT for node %s", nodeIP)
		return entry.metrics, nil
	}
	klog.V(4).Infof("Metrics cache MISS for node %s. Fetching from agent.", nodeIP)

	// If not in cache or expired, fetch from the agent
	metrics, err := c.fetchFromAgent(ctx, nodeIP)
	if err != nil {
		return nil, err
	}

	// Store the new metrics in the cache (with a write lock)
	c.mu.Lock()
	c.cache[nodeIP] = cacheEntry{
		metrics:   metrics,
		timestamp: time.Now(),
	}
	c.mu.Unlock()

	return metrics, nil
}

// fetchFromAgent performs the actual HTTP request to the agent.
func (c *Client) fetchFromAgent(ctx context.Context, nodeIP string) (*NodeMetrics, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s:9090/metrics", nodeIP), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent returned non-200 status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var nm NodeMetrics
	if err := json.Unmarshal(body, &nm); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics JSON: %w", err)
	}

	return &nm, nil
}
