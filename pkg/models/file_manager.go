package models

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var (
	commitPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

type FileManager struct {
	cacheDir    string
	hubRootOnce sync.Once
	hubRootVal  string
}

func NewFileManager(cacheDir string) *FileManager {
	return &FileManager{cacheDir: cacheDir}
}

// sanitizePathComponent removes dangerous path elements to prevent path traversal.
func sanitizePathComponent(component string) string {
	component = strings.ReplaceAll(component, string(filepath.Separator), "_")
	component = strings.ReplaceAll(component, "/", "_")
	component = strings.ReplaceAll(component, "\\", "_")
	component = strings.ReplaceAll(component, "..", "_")
	component = strings.TrimPrefix(component, ".")
	component = strings.TrimSpace(component)
	if component == "" {
		component = "unknown"
	}
	return component
}

func (fm *FileManager) HFHubRoot() string {
	fm.hubRootOnce.Do(func() {
		for _, env := range []string{"LLAMA_CACHE", "HF_HUB_CACHE", "HUGGINGFACE_HUB_CACHE"} {
			if v := os.Getenv(env); v != "" {
				fm.hubRootVal = v
				return
			}
		}
		if v := os.Getenv("HF_HOME"); v != "" {
			fm.hubRootVal = filepath.Join(v, "hub")
			return
		}
		if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
			fm.hubRootVal = filepath.Join(v, "huggingface", "hub")
			return
		}
		if v := os.Getenv("HOME"); v != "" {
			fm.hubRootVal = filepath.Join(v, ".cache", "huggingface", "hub")
			return
		}
		fm.hubRootVal = filepath.Join(fm.cacheDir, "huggingface", "hub")
	})
	return fm.hubRootVal
}

func (fm *FileManager) HFRepoDirName(repo string) string {
	parts := strings.Split(repo, "/")
	safe := make([]string, 0, len(parts))
	for _, p := range parts {
		p = sanitizePathComponent(p)
		if p != "" {
			safe = append(safe, p)
		}
	}
	return "models--" + strings.Join(safe, "--")
}

func (fm *FileManager) HFRepoDir(repo string) string {
	return filepath.Join(fm.HFHubRoot(), fm.HFRepoDirName(repo))
}

func (fm *FileManager) HFBlobPath(repo, oid string) string {
	return filepath.Join(fm.HFRepoDir(repo), "blobs", oid)
}

// DigestHex derives a safe blob name from an untrusted string (e.g. an ETag).
func (fm *FileManager) DigestHex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// HFValidateBlobPath checks path is models--*/blobs/<one component> under the hub root.
func (fm *FileManager) HFValidateBlobPath(path string) error {
	rel, err := filepath.Rel(fm.HFHubRoot(), path)
	if err != nil {
		return fmt.Errorf("blob path %q cannot be related to hub root: %w", path, err)
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
		return fmt.Errorf("blob path %q escapes the hub root", path)
	}
	if filepath.Base(filepath.Dir(rel)) != "blobs" ||
		!strings.HasPrefix(filepath.Base(filepath.Dir(filepath.Dir(rel))), "models--") {
		return fmt.Errorf("blob path %q is not a repository blob path", path)
	}
	return nil
}

func (fm *FileManager) HFSnapshotPath(repo, commit, filename string) string {
	if !commitPattern.MatchString(commit) {
		commit = "legacy"
	}
	return filepath.Join(fm.HFRepoDir(repo), "snapshots", commit, sanitizePathComponent(filename))
}

func (fm *FileManager) HFRefPath(repo, branch string) string {
	return filepath.Join(fm.HFRepoDir(repo), "refs", sanitizePathComponent(branch))
}

func (fm *FileManager) HFCreateSymlink(blobPath, snapshotPath string) error {
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	relativePath, err := filepath.Rel(filepath.Dir(snapshotPath), blobPath)
	if err != nil {
		return fmt.Errorf("failed to compute relative path: %w", err)
	}

	err = os.Symlink(relativePath, snapshotPath)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		if strings.Contains(err.Error(), "operation not permitted") || strings.Contains(err.Error(), "symlink") {
			return fm.HFCreateSymlinkFallback(blobPath, snapshotPath)
		}
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	return nil
}

func (fm *FileManager) HFCreateSymlinkFallback(blobPath, snapshotPath string) error {
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	// random suffix + os.Link can't clobber a planted path; dot prefix marks scratch for the sweep
	for attempt := 0; attempt < 3; attempt++ {
		tmpPath := filepath.Join(filepath.Dir(snapshotPath), "."+filepath.Base(snapshotPath)+".lnk-"+rand.Text())
		err := os.Link(blobPath, tmpPath)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to create hard link fallback: %w", err)
		}
		if err := os.Rename(tmpPath, snapshotPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to rename to final path: %w", err)
		}
		return nil
	}
	return fmt.Errorf("failed to create hard link fallback for %s: temp name collision", snapshotPath)
}

func (fm *FileManager) ComputeFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
