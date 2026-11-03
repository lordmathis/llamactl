package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func writeDotEnv(t *testing.T, path, content string, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("Failed to write %s: %v", path, err)
	}
	// chmod sets exact permissions; WriteFile's perm is trimmed by umask.
	if err := os.Chmod(path, perm); err != nil {
		t.Fatalf("Failed to chmod %s: %v", path, err)
	}
}

func TestVerifyDotEnvInfo_AcceptsOwnRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	writeDotEnv(t, path, "A=1\n", 0o600)

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if err := verifyDotEnvInfo(path, info); err != nil {
		t.Errorf("Expected own 0600 regular file to pass, got: %v", err)
	}
}

func TestVerifyDotEnvInfo_RejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.env")
	writeDotEnv(t, target, "A=1\n", 0o600)

	link := filepath.Join(dir, ".env")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("Symlink creation failed: %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if err := verifyDotEnvInfo(link, info); err == nil {
		t.Error("Expected a symlinked .env to be rejected")
	}
}

func TestVerifyDotEnvInfo_RejectsDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if err := verifyDotEnvInfo(path, info); err == nil {
		t.Error("Expected a directory named .env to be rejected")
	}
}

func TestVerifyDotEnvInfo_RejectsWritableByOthers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not enforced on Windows")
	}

	for _, perm := range []os.FileMode{0o620, 0o602, 0o666} {
		path := filepath.Join(t.TempDir(), ".env")
		writeDotEnv(t, path, "A=1\n", perm)

		info, err := os.Lstat(path)
		if err != nil {
			t.Fatalf("Lstat: %v", err)
		}
		if err := verifyDotEnvInfo(path, info); err == nil {
			t.Errorf("Expected .env with mode %s to be rejected", perm)
		}
	}
}

func TestLoadDotEnv_SymlinkRejected(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.env")
	writeDotEnv(t, target, "A=1\n", 0o600)

	link := filepath.Join(dir, ".env")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("Symlink creation failed: %v", err)
	}

	configPath := filepath.Join(dir, "config.yaml")
	writeDotEnv(t, configPath, "", 0o644)

	_, err := loadDotEnv(configPath)
	if err == nil {
		t.Fatal("Expected loadDotEnv to fail on a symlinked .env")
	}
	if !strings.Contains(err.Error(), link) {
		t.Errorf("Expected error to name the offending file %s, got: %v", link, err)
	}
}
