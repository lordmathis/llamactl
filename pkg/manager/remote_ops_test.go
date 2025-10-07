package manager

import (
	"llamactl/pkg/backends"
	"llamactl/pkg/instance"
	"testing"
)

func TestStripNodesFromOptions(t *testing.T) {
	im := &instanceManager{}

	// Test nil case
	if result := im.stripNodesFromOptions(nil); result != nil {
		t.Errorf("Expected nil, got %+v", result)
	}

	// Test main case: nodes should be stripped, other fields preserved
	options := &instance.CreateInstanceOptions{
		BackendType: backends.BackendTypeLlamaCpp,
		Nodes:       []string{"node1", "node2"},
		Environment: map[string]string{"TEST": "value"},
	}

	result := im.stripNodesFromOptions(options)

	if result.Nodes != nil {
		t.Errorf("Expected Nodes to be nil, got %+v", result.Nodes)
	}
	if result.BackendType != backends.BackendTypeLlamaCpp {
		t.Errorf("Expected BackendType preserved")
	}
	if result.Environment["TEST"] != "value" {
		t.Errorf("Expected Environment preserved")
	}
	// Original should not be modified
	if len(options.Nodes) != 2 {
		t.Errorf("Original options should not be modified")
	}
}
