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
	"net/url"
	"sync"
	"time"
)

const apiBasePath = "/api/v1/instances/"

// remoteManager handles HTTP operations for remote instances.
type remoteManager struct {
	mu             sync.RWMutex
	client         *http.Client
	nodeMap        map[string]*config.NodeConfig // node name -> node config
	instanceToNode map[string]*config.NodeConfig // instance name -> node config
}

// newRemoteManager creates a new remote manager.
func newRemoteManager(nodes map[string]config.NodeConfig, timeout time.Duration) *remoteManager {
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
func (rm *remoteManager) getNodeForInstance(instanceName string) (*config.NodeConfig, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	node, exists := rm.instanceToNode[instanceName]
	return node, exists
}

// SetInstanceNode maps an instance to a specific node.
// Returns an error if the node doesn't exist.
func (rm *remoteManager) setInstanceNode(instanceName, nodeName string) error {
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
func (rm *remoteManager) removeInstance(instanceName string) {
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

// createInstance creates a new instance on a remote node.
func (rm *remoteManager) createInstance(ctx context.Context, node *config.NodeConfig, name string, opts *instance.Options) (*instance.Instance, error) {
	escapedName := url.PathEscape(name)

	path := fmt.Sprintf("%s%s/", apiBasePath, escapedName)

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

// getInstance retrieves an instance by name from a remote node.
func (rm *remoteManager) getInstance(ctx context.Context, node *config.NodeConfig, name string) (*instance.Instance, error) {

	escapedName := url.PathEscape(name)

	path := fmt.Sprintf("%s%s/", apiBasePath, escapedName)
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

// updateInstance updates an existing instance on a remote node.
func (rm *remoteManager) updateInstance(ctx context.Context, node *config.NodeConfig, name string, opts *instance.Options) (*instance.Instance, error) {

	escapedName := url.PathEscape(name)

	path := fmt.Sprintf("%s%s/", apiBasePath, escapedName)

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

// deleteInstance deletes an instance from a remote node.
func (rm *remoteManager) deleteInstance(ctx context.Context, node *config.NodeConfig, name string) error {

	escapedName := url.PathEscape(name)

	path := fmt.Sprintf("%s%s/", apiBasePath, escapedName)
	resp, err := rm.makeRemoteRequest(ctx, node, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return parseRemoteResponse(resp, nil)
}

// startInstance starts an instance on a remote node.
func (rm *remoteManager) startInstance(ctx context.Context, node *config.NodeConfig, name string) (*instance.Instance, error) {

	escapedName := url.PathEscape(name)

	path := fmt.Sprintf("%s%s/start", apiBasePath, escapedName)
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

// stopInstance stops an instance on a remote node.
func (rm *remoteManager) stopInstance(ctx context.Context, node *config.NodeConfig, name string) (*instance.Instance, error) {

	escapedName := url.PathEscape(name)

	path := fmt.Sprintf("%s%s/stop", apiBasePath, escapedName)
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

// restartInstance restarts an instance on a remote node.
func (rm *remoteManager) restartInstance(ctx context.Context, node *config.NodeConfig, name string) (*instance.Instance, error) {
	escapedName := url.PathEscape(name)

	path := fmt.Sprintf("%s%s/restart", apiBasePath, escapedName)
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

// getInstanceLogs retrieves logs for an instance from a remote node.
func (rm *remoteManager) getInstanceLogs(ctx context.Context, node *config.NodeConfig, name string, numLines int) (string, error) {

	escapedName := url.PathEscape(name)

	path := fmt.Sprintf("%s%s/logs?lines=%d", apiBasePath, escapedName, numLines)
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
