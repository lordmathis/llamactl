package instance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"llamactl/pkg/backends"
	"net/http"
	"time"
)

// Model represents a model available in a llama.cpp instance
type Model struct {
	ID      string      `json:"id"`
	Object  string      `json:"object"`
	OwnedBy string      `json:"owned_by"`
	Created int64       `json:"created"`
	InCache bool        `json:"in_cache"`
	Path    string      `json:"path"`
	Status  ModelStatus `json:"status"`
}

// ModelStatus represents the status of a model in an instance
type ModelStatus struct {
	Value string   `json:"value"` // "loaded" | "loading" | "unloaded"
	Args  []string `json:"args"`
}

// IsLlamaCpp checks if this instance is a llama.cpp instance
func (i *Instance) IsLlamaCpp() bool {
	opts := i.GetOptions()
	if opts == nil {
		return false
	}
	return opts.BackendOptions.BackendType == backends.BackendTypeLlamaCpp
}

// GetModels fetches the models available in this llama.cpp instance
func (i *Instance) GetModels() ([]Model, error) {
	if !i.IsLlamaCpp() {
		return nil, fmt.Errorf("instance %s is not a llama.cpp instance", i.Name)
	}

	if !i.IsRunning() {
		return nil, fmt.Errorf("instance %s is not running", i.Name)
	}

	var result struct {
		Data []Model `json:"data"`
	}
	if err := i.doRequest("GET", "/models", nil, &result, 10*time.Second); err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}

	return result.Data, nil
}

// LoadModel loads a model in this llama.cpp instance
func (i *Instance) LoadModel(modelName string) error {
	if !i.IsLlamaCpp() {
		return fmt.Errorf("instance %s is not a llama.cpp instance", i.Name)
	}

	if !i.IsRunning() {
		return fmt.Errorf("instance %s is not running", i.Name)
	}

	// Make the load request
	reqBody := map[string]string{"model": modelName}
	if err := i.doRequest("POST", "/models/load", reqBody, nil, 30*time.Second); err != nil {
		return fmt.Errorf("failed to load model: %w", err)
	}

	return nil
}

// UnloadModel unloads a model from this llama.cpp instance
func (i *Instance) UnloadModel(modelName string) error {
	if !i.IsLlamaCpp() {
		return fmt.Errorf("instance %s is not a llama.cpp instance", i.Name)
	}

	if !i.IsRunning() {
		return fmt.Errorf("instance %s is not running", i.Name)
	}

	// Make the unload request
	reqBody := map[string]string{"model": modelName}
	if err := i.doRequest("POST", "/models/unload", reqBody, nil, 30*time.Second); err != nil {
		return fmt.Errorf("failed to unload model: %w", err)
	}

	return nil
}

// doRequest makes an HTTP request to this instance's backend
func (i *Instance) doRequest(method, path string, reqBody, respBody any, timeout time.Duration) error {
	url := fmt.Sprintf("http://%s:%d%s", i.GetHost(), i.GetPort(), path)

	var bodyReader io.Reader
	if reqBody != nil {
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
