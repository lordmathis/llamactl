package manager_test

import (
	"bytes"
	"llamactl/pkg/manager"
	"strings"
	"testing"
)

func TestReadRemoteBody_CapsAtLimit(t *testing.T) {
	atCap := bytes.Repeat([]byte("a"), 32<<20)
	got, err := manager.ReadRemoteBody(bytes.NewReader(atCap))
	if err != nil {
		t.Fatalf("reading a response at exactly the cap: %v", err)
	}
	if len(got) != len(atCap) {
		t.Fatalf("read %d bytes, want %d", len(got), len(atCap))
	}

	overCap := bytes.Repeat([]byte("a"), (32<<20)+1)
	if _, err := manager.ReadRemoteBody(bytes.NewReader(overCap)); err == nil {
		t.Fatal("reading a response over the cap succeeded, want error")
	} else if !strings.Contains(err.Error(), "cap") {
		t.Fatalf("error %v, want it to name the cap", err)
	}
}
