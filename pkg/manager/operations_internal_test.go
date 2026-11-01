package manager

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"llamactl/pkg/backends"
	"llamactl/pkg/config"
	"llamactl/pkg/database"
	"llamactl/pkg/instance"
)

// newInternalTestManager is createTestManager returning the concrete type,
// with a port range that covers ephemeral httptest ports.
func newInternalTestManager(t *testing.T) *instanceManager {
	t.Helper()

	appConfig := &config.AppConfig{
		Backends: config.BackendConfig{
			LlamaCpp: config.BackendSettings{
				Command: "sh",
				Args:    []string{"-c", "sleep 999999"},
			},
		},
		Instances: config.InstancesConfig{
			PortRange:            [2]int{1024, 65535},
			MaxInstances:         10,
			MaxRunningInstances:  10,
			DefaultAutoRestart:   true,
			DefaultMaxRestarts:   3,
			LogsDir:              t.TempDir(),
			DefaultRestartDelay:  5,
			DefaultIdleTimeout:   30,
			TimeoutCheckInterval: 5,
		},
		Database: config.DatabaseConfig{
			Path:               ":memory:",
			MaxOpenConnections: 25,
			MaxIdleConnections: 5,
			ConnMaxLifetime:    5 * time.Minute,
		},
		LocalNode: "main",
		Nodes:     map[string]config.NodeConfig{},
	}

	db, err := database.Open(&database.Config{
		Path:               appConfig.Database.Path,
		MaxOpenConnections: appConfig.Database.MaxOpenConnections,
		MaxIdleConnections: appConfig.Database.MaxIdleConnections,
		ConnMaxLifetime:    appConfig.Database.ConnMaxLifetime,
	})
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return New(appConfig, db).(*instanceManager)
}

func internalOpts(port int, model string) *instance.Options {
	return &instance.Options{
		BackendOptions: backends.Options{
			BackendType: backends.BackendTypeLlamaCpp,
			LlamaServerOptions: &backends.LlamaServerOptions{
				Model: model,
				Port:  port,
			},
		},
	}
}

func assertPortOwned(t *testing.T, im *instanceManager, port int) {
	t.Helper()
	if err := im.ports.allocateSpecific(port, "ownership-probe"); err == nil {
		im.ports.release(port)
		t.Errorf("port %d should still be owned by the instance", port)
	}
}

func assertPortFree(t *testing.T, im *instanceManager, port int) {
	t.Helper()
	if err := im.ports.allocateSpecific(port, "ownership-probe"); err != nil {
		t.Errorf("port %d should be free: %v", port, err)
		return
	}
	if err := im.ports.release(port); err != nil {
		t.Fatalf("failed to release probe on port %d: %v", port, err)
	}
}

func TestRemoteManager_NodeNameVsInstanceNameLookup(t *testing.T) {
	rm := newRemoteManager(map[string]config.NodeConfig{
		"node-a": {Address: "http://a.example"},
		"node-b": {Address: "http://b.example"},
	}, time.Second)

	// An instance named like the *other* node — the collision case.
	if err := rm.setInstanceNode("node-a", "node-b"); err != nil {
		t.Fatalf("setInstanceNode failed: %v", err)
	}

	node, ok := rm.getNodeByName("node-b")
	if !ok || node.Address != "http://b.example" {
		t.Errorf("getNodeByName(node-b) = %v, %v; want node-b config", node, ok)
	}

	node, ok = rm.getNodeForInstance("node-a")
	if !ok || node.Address != "http://b.example" {
		t.Errorf("getNodeForInstance(node-a) = %v, %v; want node-b config", node, ok)
	}

	// The old buggy call: no instance named "node-b" is mapped.
	if node, ok := rm.getNodeForInstance("node-b"); ok {
		t.Errorf("getNodeForInstance(node-b) = %v; want not found", node)
	}
}

func TestUpdateInstance_StopFailureKeepsConsistentState(t *testing.T) {
	im := newInternalTestManager(t)
	defer im.Shutdown()

	inst, err := im.CreateInstance("stop-fail", internalOpts(18100, "/old/model.gguf"))
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}
	if err := inst.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	im.stopFn = func(*instance.Instance) error { return errors.New("injected stop failure") }
	defer func() { im.stopFn = nil }()

	_, err = im.UpdateInstance("stop-fail", internalOpts(18101, "/new/model.gguf"))
	if err == nil || !strings.Contains(err.Error(), "stop") {
		t.Fatalf("UpdateInstance error = %v, want stop failure", err)
	}

	if !inst.IsRunning() {
		t.Error("instance must still be running after a failed stop")
	}
	if model := inst.GetOptions().BackendOptions.LlamaServerOptions.Model; model != "/old/model.gguf" {
		t.Errorf("model = %q, want old options preserved", model)
	}
	assertPortOwned(t, im, 18100)
	assertPortFree(t, im, 18101)
}

func TestUpdateInstance_StartFailureRollsBackToOldPort(t *testing.T) {
	im := newInternalTestManager(t)
	defer im.Shutdown()

	inst, err := im.CreateInstance("start-fail", internalOpts(18102, "/old/model.gguf"))
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}
	if err := inst.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	badOpts := internalOpts(18103, "/new/model.gguf")
	badOpts.CommandOverride = "definitely-not-a-real-command"

	_, err = im.UpdateInstance("start-fail", badOpts)
	if err == nil {
		t.Fatal("expected update to fail when the new options cannot start")
	}
	if !strings.Contains(err.Error(), "rolled back") {
		t.Errorf("error should mention the rollback, got: %v", err)
	}

	if !inst.IsRunning() {
		t.Error("instance must be running again on the old options")
	}
	if port := inst.GetPort(); port != 18102 {
		t.Errorf("port = %d, want rolled back to 18102", port)
	}
	if model := inst.GetOptions().BackendOptions.LlamaServerOptions.Model; model != "/old/model.gguf" {
		t.Errorf("model = %q, want old options restored", model)
	}
	assertPortOwned(t, im, 18102)
	assertPortFree(t, im, 18103)
}

func TestUpdateInstance_StartAndRollbackFailureLeavesStopped(t *testing.T) {
	im := newInternalTestManager(t)
	defer im.Shutdown()

	inst, err := im.CreateInstance("double-fail", internalOpts(18104, "/old/model.gguf"))
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}
	if err := inst.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	im.startFn = func(*instance.Instance) error { return errors.New("injected start failure") }
	defer func() { im.startFn = nil }()

	_, err = im.UpdateInstance("double-fail", internalOpts(18105, "/new/model.gguf"))
	if err == nil {
		t.Fatal("expected update to fail")
	}
	if !strings.Contains(err.Error(), "failed to start") || !strings.Contains(err.Error(), "rollback") {
		t.Errorf("error must mention both the start and rollback failures, got: %v", err)
	}

	if inst.IsRunning() {
		t.Error("instance must be left stopped when the rollback restart also fails")
	}
	if model := inst.GetOptions().BackendOptions.LlamaServerOptions.Model; model != "/old/model.gguf" {
		t.Errorf("model = %q, want old options restored", model)
	}
	assertPortOwned(t, im, 18104)
	assertPortFree(t, im, 18105)
}

// No process start here: the double-spawn guard would refuse a port the test backend already binds.
func TestUpdateInstance_PortChangeServesFromNewBackend(t *testing.T) {
	im := newInternalTestManager(t)
	defer im.Shutdown()

	backendA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "A")
	}))
	defer backendA.Close()
	backendB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "B")
	}))
	defer backendB.Close()

	portA := backendPort(t, backendA)
	portB := backendPort(t, backendB)

	inst, err := im.CreateInstance("proxy-int", internalOpts(portA, "/model.gguf"))
	if err != nil {
		t.Fatalf("CreateInstance failed: %v", err)
	}

	serve := func() string {
		rec := httptest.NewRecorder()
		if err := inst.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/models", nil)); err != nil {
			t.Fatalf("ServeHTTP failed: %v", err)
		}
		return rec.Body.String()
	}

	if body := serve(); body != "A" {
		t.Fatalf("served by %q, want backend A", body)
	}

	if _, err := im.UpdateInstance("proxy-int", internalOpts(portB, "/model.gguf")); err != nil {
		t.Fatalf("UpdateInstance failed: %v", err)
	}

	if body := serve(); body != "B" {
		t.Fatalf("served by %q after port change, want backend B (stale proxy target)", body)
	}
	if port := inst.GetPort(); port != portB {
		t.Errorf("port = %d, want %d", port, portB)
	}
	assertPortOwned(t, im, portB)
	assertPortFree(t, im, portA)
}

func backendPort(t *testing.T, srv *httptest.Server) int {
	t.Helper()
	_, portStr, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse backend address: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse backend port: %v", err)
	}
	return port
}
