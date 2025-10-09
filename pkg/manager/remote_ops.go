package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"llamactl/pkg/config"
	"llamactl/pkg/instance"
	"net/http"
)

// stripNodesFromOptions creates a copy of the instance options without the Nodes field
// to prevent routing loops when sending requests to remote nodes
func (im *instanceManager) stripNodesFromOptions(options *instance.CreateInstanceOptions) *instance.CreateInstanceOptions {
	if options == nil {
		return nil
	}

	// Create a copy of the options struct
	optionsCopy := *options

	// Clear the Nodes field to prevent the remote node from trying to route further
	optionsCopy.Nodes = nil

	return &optionsCopy
}

// makeRemoteRequest is a helper function to make HTTP requests to a remote node
func (im *instanceManager) makeRemoteRequest(nodeConfig *config.NodeConfig, method, path string, body any) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		// Strip nodes from CreateInstanceOptions to prevent routing loops
		if options, ok := body.(*instance.CreateInstanceOptions); ok {
			body = im.stripNodesFromOptions(options)
		}

		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := fmt.Sprintf("%s%s", nodeConfig.Address, path)
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if nodeConfig.APIKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", nodeConfig.APIKey))
	}

	resp, err := im.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// parseRemoteResponse is a helper function to parse API responses
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

// ListRemoteInstances lists all instances on the remote node
func (im *instanceManager) ListRemoteInstances(nodeConfig *config.NodeConfig) ([]*instance.Process, error) {
	resp, err := im.makeRemoteRequest(nodeConfig, "GET", "/api/v1/instances/", nil)
	if err != nil {
		return nil, err
	}

	var instances []*instance.Process
	if err := parseRemoteResponse(resp, &instances); err != nil {
		return nil, err
	}

	return instances, nil
}

// CreateRemoteInstance creates a new instance on the remote node
func (im *instanceManager) CreateRemoteInstance(nodeConfig *config.NodeConfig, name string, options *instance.CreateInstanceOptions) (*instance.Process, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/", name)

	resp, err := im.makeRemoteRequest(nodeConfig, "POST", path, options)
	if err != nil {
		return nil, err
	}

	var inst instance.Process
	if err := parseRemoteResponse(resp, &inst); err != nil {
		return nil, err
	}

	return &inst, nil
}

// GetRemoteInstance retrieves an instance by name from the remote node
func (im *instanceManager) GetRemoteInstance(nodeConfig *config.NodeConfig, name string) (*instance.Process, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/", name)
	resp, err := im.makeRemoteRequest(nodeConfig, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var inst instance.Process
	if err := parseRemoteResponse(resp, &inst); err != nil {
		return nil, err
	}

	return &inst, nil
}

// UpdateRemoteInstance updates an existing instance on the remote node
func (im *instanceManager) UpdateRemoteInstance(nodeConfig *config.NodeConfig, name string, options *instance.CreateInstanceOptions) (*instance.Process, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/", name)

	resp, err := im.makeRemoteRequest(nodeConfig, "PUT", path, options)
	if err != nil {
		return nil, err
	}

	var inst instance.Process
	if err := parseRemoteResponse(resp, &inst); err != nil {
		return nil, err
	}

	return &inst, nil
}

// DeleteRemoteInstance deletes an instance from the remote node
func (im *instanceManager) DeleteRemoteInstance(nodeConfig *config.NodeConfig, name string) error {
	path := fmt.Sprintf("/api/v1/instances/%s/", name)
	resp, err := im.makeRemoteRequest(nodeConfig, "DELETE", path, nil)
	if err != nil {
		return err
	}

	return parseRemoteResponse(resp, nil)
}

// StartRemoteInstance starts an instance on the remote node
func (im *instanceManager) StartRemoteInstance(nodeConfig *config.NodeConfig, name string) (*instance.Process, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/start", name)
	resp, err := im.makeRemoteRequest(nodeConfig, "POST", path, nil)
	if err != nil {
		return nil, err
	}

	var inst instance.Process
	if err := parseRemoteResponse(resp, &inst); err != nil {
		return nil, err
	}

	return &inst, nil
}

// StopRemoteInstance stops an instance on the remote node
func (im *instanceManager) StopRemoteInstance(nodeConfig *config.NodeConfig, name string) (*instance.Process, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/stop", name)
	resp, err := im.makeRemoteRequest(nodeConfig, "POST", path, nil)
	if err != nil {
		return nil, err
	}

	var inst instance.Process
	if err := parseRemoteResponse(resp, &inst); err != nil {
		return nil, err
	}

	return &inst, nil
}

// RestartRemoteInstance restarts an instance on the remote node
func (im *instanceManager) RestartRemoteInstance(nodeConfig *config.NodeConfig, name string) (*instance.Process, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/restart", name)
	resp, err := im.makeRemoteRequest(nodeConfig, "POST", path, nil)
	if err != nil {
		return nil, err
	}

	var inst instance.Process
	if err := parseRemoteResponse(resp, &inst); err != nil {
		return nil, err
	}

	return &inst, nil
}

// GetRemoteInstanceLogs retrieves logs for an instance from the remote node
func (im *instanceManager) GetRemoteInstanceLogs(nodeConfig *config.NodeConfig, name string, numLines int) (string, error) {
	path := fmt.Sprintf("/api/v1/instances/%s/logs?lines=%d", name, numLines)
	resp, err := im.makeRemoteRequest(nodeConfig, "GET", path, nil)
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

	// Logs endpoint might return plain text or JSON
	// Try to parse as JSON first (in case it's wrapped in a response object)
	var logResponse struct {
		Logs string `json:"logs"`
	}
	if err := json.Unmarshal(body, &logResponse); err == nil && logResponse.Logs != "" {
		return logResponse.Logs, nil
	}

	// Otherwise, return as plain text
	return string(body), nil
}
