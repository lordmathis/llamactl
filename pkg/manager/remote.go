package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const apiBasePath = "/api/v1/instances/"

// ErrInstanceNotFound is the sentinel returned when an instance cannot be
// located, either in the local registry or on the remote node. Use errors.Is
// to detect it.
var ErrInstanceNotFound = errors.New("instance not found")

// IsNotFoundError reports whether err (or anything in its chain) signals that
// the targeted instance does not exist. It catches both the local sentinel
// (ErrInstanceNotFound) and remote-side variants returned as a different body
// shape by older nodes.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrInstanceNotFound) {
		return true
	}
	var rnf *RemoteNotFoundError
	if errors.As(err, &rnf) {
		return true
	}
	// Fallback: some nodes return 400 with an "invalid_instance" body and a
	// message containing "not found". Treat those as not-found as well.
	msg := err.Error()
	return strings.Contains(msg, `"invalid_instance"`) &&
		strings.Contains(strings.ToLower(msg), "not found")
}

// RemoteNotFoundError is returned by remoteManager CRUD functions when the
// remote node reports the target instance does not exist. Callers can use
// errors.As to detect it; it also satisfies errors.Is(err, ErrInstanceNotFound)
// so IsNotFoundError works uniformly across local and remote failures.
type RemoteNotFoundError struct {
	Name string
	Err  error
}

func (e *RemoteNotFoundError) Error() string { return e.Err.Error() }
func (e *RemoteNotFoundError) Unwrap() error   { return e.Err }

func (e *RemoteNotFoundError) Is(target error) bool {
	return target == ErrInstanceNotFound
}

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
//
// If the response indicates the target instance does not exist (HTTP 404, or a
// 4xx whose JSON body has error == "invalid_instance"/"not_found"), the
// returned error is *RemoteNotFoundError so callers can distinguish "not there"
// from real failures and complete local cleanup, idempotent-retry, etc.
func parseRemoteResponse(resp *http.Response, result any) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return &RemoteNotFoundError{
			Err: fmt.Errorf("remote returned 404: %s", string(body)),
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if remoteBodyIsNotFound(body) {
			return &RemoteNotFoundError{
				Err: fmt.Errorf("remote returned %d with not-found body: %s", resp.StatusCode, string(body)),
			}
		}
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// remoteBodyIsNotFound returns true if the response body's JSON indicates the
// target object is missing. Older llamactl nodes and some existing handlers
// return 400 with {"error":"invalid_instance","details":"<name> not found"}
// instead of a real 404; we map those to the same not-found sentinel.
func remoteBodyIsNotFound(body []byte) bool {
	var payload struct {
		Error   string `json:"error"`
		Details string `json:"details"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	if payload.Error == "invalid_instance" {
		return true
	}
	if payload.Error == "not_found" || payload.Error == "notfound" {
		return true
	}
	if strings.Contains(strings.ToLower(payload.Details), "not found") {
		return true
	}
	return false
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
