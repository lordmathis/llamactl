package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"net/http"
	"sync"
	"time"
)

// remoteManager handles HTTP operations for remote instances.
type remoteManager struct {
	mu             sync.RWMutex
	client         *http.Client
	nodeMap        map[string]*config.NodeConfig // node name -> node config
	instanceToNode map[string]*config.NodeConfig // instance name -> node config
}

// NewRemoteManager creates a new remote manager.
func NewRemoteManager(nodes map[string]config.NodeConfig, timeout time.Duration) *remoteManager {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// Build node config map
	nodeMap := make(map[string]*config.NodeConfig)
	for name := range nodes {
		nodeCopy := nodes[name]
		nodeMap[name] = &nodeCopy
	}

	return &remoteManager{
		client: &http.Client{
			Timeout: timeout,
		},
		nodeMap:        nodeMap,
		instanceToNode: make(map[string]*config.NodeConfig),
	}
}

// GetNodeForInstance returns the node configuration for a given instance.
// Returns nil if the instance is not mapped to any node.
func (rm *remoteManager) GetNodeForInstance(instanceName string) (*config.NodeConfig, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	node, exists := rm.instanceToNode[instanceName]
	return node, exists
}

// SetInstanceNode maps an instance to a specific node.
// Returns an error if the node doesn't exist.
func (rm *remoteManager) SetInstanceNode(instanceName, nodeName string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	node, exists := rm.nodeMap[nodeName]
	if !exists {
		return fmt.Errorf("node %s not found", nodeName)
	}

	rm.instanceToNode[instanceName] = node
	return nil
}

// RemoveInstance removes the instance-to-node mapping.
func (rm *remoteManager) RemoveInstance(instanceName string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.instanceToNode, instanceName)
}

// --- HTTP request helpers ---

// makeRemoteRequest creates and executes an HTTP request to a remote node with context support.
func (rm *remoteManager) makeRemoteRequest(ctx context.Context, nodeConfig *config.NodeConfig, method, path string, body any) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := fmt.Sprintf("%s%s", nodeConfig.Address, path)
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if nodeConfig.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", nodeConfig.APIKey))
	}

	resp, err := rm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// parseRemoteResponse parses an HTTP response and unmarshals the result.
func parseRemoteResponse(resp *http.Response, result any) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// --- Remote CRUD operations ---

// ListInstances lists all instances on a remote node.
func (rm *remoteManager) ListInstances(ctx context.Context, node *config.NodeConfig) ([]*instance.Instance, error) {
	resp, err := rm.makeRemoteRequest(ctx, node, "GET", "/api/v1/instances/", nil)
	if err != nil {
		return nil, err
	}

	var instances []*instance.Instance
	if err := parseRemoteResponse(resp, &instances); err != nil {
		return nil, err
	}

	return instances, nil
}

// CreateInstance creates a new instance on a remote node.
func (rm *remoteManager) CreateInstance(ctx context.Context, node *config.NodeConfig, name string, opts *instance.Options) (*instance.Instance, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/", name)

	resp, err := rm.makeRemoteRequest(ctx, node, "POST", path, opts)
	if err != nil {
		return nil, err
	}

	var inst instance.Instance
	if err := parseRemoteResponse(resp, &inst); err != nil {
		return nil, err
	}

	return &inst, nil
}

// GetInstance retrieves an instance by name from a remote node.
func (rm *remoteManager) GetInstance(ctx context.Context, node *config.NodeConfig, name string) (*instance.Instance, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/", name)
	resp, err := rm.makeRemoteRequest(ctx, node, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var inst instance.Instance
	if err := parseRemoteResponse(resp, &inst); err != nil {
		return nil, err
	}

	return &inst, nil
}

// UpdateInstance updates an existing instance on a remote node.
func (rm *remoteManager) UpdateInstance(ctx context.Context, node *config.NodeConfig, name string, opts *instance.Options) (*instance.Instance, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/", name)

	resp, err := rm.makeRemoteRequest(ctx, node, "PUT", path, opts)
	if err != nil {
		return nil, err
	}

	var inst instance.Instance
	if err := parseRemoteResponse(resp, &inst); err != nil {
		return nil, err
	}

	return &inst, nil
}

// DeleteInstance deletes an instance from a remote node.
func (rm *remoteManager) DeleteInstance(ctx context.Context, node *config.NodeConfig, name string) error {
	path := fmt.Sprintf("/api/v1/instances/%s/", name)
	resp, err := rm.makeRemoteRequest(ctx, node, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return parseRemoteResponse(resp, nil)
}

// StartInstance starts an instance on a remote node.
func (rm *remoteManager) StartInstance(ctx context.Context, node *config.NodeConfig, name string) (*instance.Instance, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/start", name)
	resp, err := rm.makeRemoteRequest(ctx, node, "POST", path, nil)
	if err != nil {
		return nil, err
	}

	var inst instance.Instance
	if err := parseRemoteResponse(resp, &inst); err != nil {
		return nil, err
	}

	return &inst, nil
}

// StopInstance stops an instance on a remote node.
func (rm *remoteManager) StopInstance(ctx context.Context, node *config.NodeConfig, name string) (*instance.Instance, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/stop", name)
	resp, err := rm.makeRemoteRequest(ctx, node, "POST", path, nil)
	if err != nil {
		return nil, err
	}

	var inst instance.Instance
	if err := parseRemoteResponse(resp, &inst); err != nil {
		return nil, err
	}

	return &inst, nil
}

// RestartInstance restarts an instance on a remote node.
func (rm *remoteManager) RestartInstance(ctx context.Context, node *config.NodeConfig, name string) (*instance.Instance, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/restart", name)
	resp, err := rm.makeRemoteRequest(ctx, node, "POST", path, nil)
	if err != nil {
		return nil, err
	}

	var inst instance.Instance
	if err := parseRemoteResponse(resp, &inst); err != nil {
		return nil, err
	}

	return &inst, nil
}

// GetInstanceLogs retrieves logs for an instance from a remote node.
func (rm *remoteManager) GetInstanceLogs(ctx context.Context, node *config.NodeConfig, name string, numLines int) (string, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/logs?lines=%d", name, numLines)
	resp, err := rm.makeRemoteRequest(ctx, node, "GET", path, nil)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Logs endpoint returns plain text (Content-Type: text/plain)
	return string(body), nil
}
