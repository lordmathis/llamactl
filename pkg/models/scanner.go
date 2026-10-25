package models

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type CachedModel struct {
	Node      string `json:"node"`
	Repo      string `json:"repo"`
	Tag       string `json:"tag,omitempty"`
	Files     []File `json:"files"`
	SizeBytes int64  `json:"size_bytes"`
}

type File struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	Type      string `json:"type"`
}

var (
	splitFilePattern = regexp.MustCompile(`-\d{5}-of-(\d{5})\.gguf$`)
	shaPattern       = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
)

func ScanCache(cacheDir, nodeName string) ([]CachedModel, error) {
	models := []CachedModel{}

	fileManager := NewFileManager(cacheDir)
	hubRoot := fileManager.HFHubRoot()

	// Scan new HF snapshot layout
	if _, err := os.Stat(hubRoot); err == nil {
		entries, err := os.ReadDir(hubRoot)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "models--") {
				continue
			}
			dirName := entry.Name()
			repo := parseHFRepoDirName(dirName)
			if repo == "" {
				continue
			}

			// Iterate over all refs
			refsDir := filepath.Join(hubRoot, dirName, "refs")
			if refEntries, err := os.ReadDir(refsDir); err == nil {
				for _, refEntry := range refEntries {
					if refEntry.IsDir() {
						continue
					}
					commit, err := readRefFile(filepath.Join(refsDir, refEntry.Name()))
					if err != nil {
						continue
					}
					cachedModel := scanHFRepo(hubRoot, dirName, repo, commit, nodeName)
					if len(cachedModel.Files) > 0 {
						cachedModel.Tag = refEntry.Name()
						models = append(models, cachedModel)
					}
				}
			} else {
				// Fallback: if no refs, scan snapshots directory
				snapshotRoot := filepath.Join(hubRoot, dirName, "snapshots")
				if snapEntries, err := os.ReadDir(snapshotRoot); err == nil {
					for _, snapEntry := range snapEntries {
						if !snapEntry.IsDir() {
							continue
						}
						commit := snapEntry.Name()
						cachedModel := scanHFRepo(hubRoot, dirName, repo, commit, nodeName)
						if len(cachedModel.Files) > 0 {
							models = append(models, cachedModel)
						}
					}
				}
			}
		}
	}

	// Backwards-compatible: scan old flat manifest= layout in cacheDir
	legacyModels, err := scanLegacyCache(cacheDir, nodeName)
	if err == nil {
		models = append(models, legacyModels...)
	}

	return models, nil
}

func parseHFRepoDirName(dirName string) string {
	if !strings.HasPrefix(dirName, "models--") {
		return ""
	}

	trimmed := strings.TrimPrefix(dirName, "models--")
	parts := strings.Split(trimmed, "--")
	if len(parts) < 2 {
		return ""
	}

	return strings.Join(parts, "/")
}

func readRefFile(refPath string) (string, error) {
	data, err := os.ReadFile(refPath)
	if err != nil {
		return "", err
	}
	commit := strings.TrimSpace(string(data))
	if !shaPattern.MatchString(commit) {
		return "", fmt.Errorf("invalid commit hash in ref file: %s", commit)
	}
	return commit, nil
}

func scanHFRepo(hubRoot, dirName, repo, commit, nodeName string) CachedModel {
	model := CachedModel{
		Node:  nodeName,
		Repo:  repo,
		Tag:   commit, // Default tag to commit hash
		Files: []File{},
	}

	if !shaPattern.MatchString(commit) {
		return model
	}

	snapshotDir := filepath.Join(hubRoot, dirName, "snapshots", commit)
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		return model
	}

	snapshotEntries, err := os.ReadDir(snapshotDir)
	if err != nil {
		return model
	}

	blobPaths := make(map[string]bool)

	for _, snapshotEntry := range snapshotEntries {
		if snapshotEntry.IsDir() {
			continue
		}

		symlinkPath := filepath.Join(snapshotDir, snapshotEntry.Name())
		blobPath, err := os.Readlink(symlinkPath)
		if err != nil {
			// If it's not a symlink, treat the file itself as the blob.
			// This handles hard links or regular files created by fallbacks.
			blobPath = symlinkPath
		} else if !filepath.IsAbs(blobPath) {
			blobPath = filepath.Join(filepath.Dir(symlinkPath), blobPath)
		}

		blobInfo, err := os.Stat(blobPath)
		if err != nil {
			continue
		}

		blobPaths[blobPath] = true

		model.Files = append(model.Files, File{
			Name:      snapshotEntry.Name(),
			Path:      symlinkPath,
			SizeBytes: blobInfo.Size(),
			Type:      classifyHFModelFileType(snapshotEntry.Name()),
		})
	}

	var totalSize int64
	for blobPath := range blobPaths {
		if info, err := os.Stat(blobPath); err == nil {
			totalSize += info.Size()
		}
	}
	model.SizeBytes = totalSize

	return model
}

func classifyHFModelFileType(filename string) string {
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".gguf") {
		if strings.Contains(lower, "mmproj") {
			return "mmproj"
		}
		return "gguf"
	}
	if lower == "preset.ini" {
		return "preset"
	}
	return "other"
}

// scanLegacyCache reads old flat-layout manifest=*.json files and returns CachedModel
// entries for any models found. This provides backwards compatibility until users
// have migrated to the new HF snapshot layout.
func scanLegacyCache(cacheDir, nodeName string) ([]CachedModel, error) {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, err
	}

	type legacyFileRef struct {
		RFilename string `json:"rfilename"`
	}
	type legacyManifest struct {
		GGUFFile   *legacyFileRef `json:"ggufFile"`
		MMProjFile *legacyFileRef `json:"mmprojFile,omitempty"`
	}

	var models []CachedModel
	seen := map[string]bool{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "manifest=") || !strings.HasSuffix(name, ".json") {
			continue
		}

		// Parse repo and tag from "manifest=org=model=tag.json"
		trimmed := strings.TrimSuffix(strings.TrimPrefix(name, "manifest="), ".json")
		parts := strings.Split(trimmed, "=")
		if len(parts) < 2 {
			continue
		}
		tag := parts[len(parts)-1]
		repo := strings.Join(parts[:len(parts)-1], "/")

		key := repo + ":" + tag
		if seen[key] {
			continue
		}
		seen[key] = true

		data, err := os.ReadFile(filepath.Join(cacheDir, name))
		if err != nil {
			continue
		}
		var manifest legacyManifest
		if err := json.Unmarshal(data, &manifest); err != nil || manifest.GGUFFile == nil {
			continue
		}

		repoPrefix := strings.ReplaceAll(repo, "/", "_")
		ggufFilename := filepath.Base(manifest.GGUFFile.RFilename)
		ggufPath := filepath.Join(cacheDir, repoPrefix+"_"+ggufFilename)

		info, err := os.Stat(ggufPath)
		if err != nil {
			continue
		}

		files := []File{{
			Name:      ggufFilename,
			Path:      ggufPath,
			SizeBytes: info.Size(),
			Type:      classifyHFModelFileType(ggufFilename),
		}}
		totalSize := info.Size()

		if manifest.MMProjFile != nil {
			mmprojFilename := filepath.Base(manifest.MMProjFile.RFilename)
			mmprojPath := filepath.Join(cacheDir, repoPrefix+"_"+mmprojFilename)
			if mmprojInfo, err := os.Stat(mmprojPath); err == nil {
				files = append(files, File{
					Name:      mmprojFilename,
					Path:      mmprojPath,
					SizeBytes: mmprojInfo.Size(),
					Type:      "mmproj",
				})
				totalSize += mmprojInfo.Size()
			}
		}

		models = append(models, CachedModel{
			Node:      nodeName,
			Repo:      repo,
			Tag:       tag,
			Files:     files,
			SizeBytes: totalSize,
		})
	}

	return models, nil
}

func (m *Manager) ListCached(nodeName string) ([]CachedModel, error) {
	return ScanCache(m.cacheDir, nodeName)
}

func (m *Manager) DeleteModel(repo, tag string) error {
	fileManager := NewFileManager(m.cacheDir)
	repoDir := fileManager.HFRepoDir(repo)

	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		return fmt.Errorf("model not found: %s", repo)
	}

	if tag == "" {
		return os.RemoveAll(repoDir)
	}

	refPath := fileManager.HFRefPath(repo, tag)
	commit, err := readRefFile(refPath)
	if err != nil {
		return fmt.Errorf("model not found: %s:%s", repo, tag)
	}

	shared, err := commitReferencedByOtherRef(repoDir, filepath.Base(refPath), commit)
	if err != nil {
		return err
	}
	if shared {
		// Another ref points at the same snapshot: only drop this ref.
		return os.Remove(refPath)
	}

	snapshotDir := filepath.Join(repoDir, "snapshots", commit)
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		// dangling ref: snapshot already gone; drop the ref instead of 404 forever
		return os.Remove(refPath)
	}

	referenced, err := blobsReferencedByOtherSnapshots(repoDir, commit)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(snapshotDir)
	if err != nil {
		return err
	}

	blobDir := filepath.Join(repoDir, "blobs")
	var entryPaths, blobPaths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		entryPath := filepath.Join(snapshotDir, entry.Name())
		entryPaths = append(entryPaths, entryPath)
		if blobPath, ok := containedBlobPath(blobDir, entryPath); ok {
			blobPaths = append(blobPaths, blobPath)
		}
	}

	for _, entryPath := range entryPaths {
		if err := os.Remove(entryPath); err != nil {
			return fmt.Errorf("failed to remove snapshot entry %s: %w", entryPath, err)
		}
	}
	for _, blobPath := range blobPaths {
		if referenced[blobPath] {
			continue
		}
		if err := os.Remove(blobPath); err != nil {
			return fmt.Errorf("failed to remove blob %s: %w", blobPath, err)
		}
	}
	if err := os.Remove(snapshotDir); err != nil {
		return fmt.Errorf("failed to remove snapshot directory: %w", err)
	}
	if err := os.Remove(refPath); err != nil {
		return fmt.Errorf("failed to remove ref file: %w", err)
	}
	return nil
}

// commitReferencedByOtherRef reports whether another valid ref resolves to commit; invalid siblings protect nothing.
func commitReferencedByOtherRef(repoDir, excludeRef, commit string) (bool, error) {
	refsDir := filepath.Join(repoDir, "refs")
	entries, err := os.ReadDir(refsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == excludeRef {
			continue
		}
		other, err := readRefFile(filepath.Join(refsDir, entry.Name()))
		if err != nil {
			continue
		}
		if other == commit {
			return true, nil
		}
	}
	return false, nil
}

// containedBlobPath returns the blob target only if it is a regular file directly inside blobDir.
func containedBlobPath(blobDir, entryPath string) (string, bool) {
	resolvedEntry, err := filepath.EvalSymlinks(entryPath)
	if err != nil {
		return "", false
	}
	resolvedBlobDir, err := filepath.EvalSymlinks(blobDir)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(resolvedEntry)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	rel, err := filepath.Rel(resolvedBlobDir, resolvedEntry)
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") ||
		filepath.IsAbs(rel) || strings.Contains(rel, "/") {
		return "", false
	}
	return resolvedEntry, true
}

// blobsReferencedByOtherSnapshots collects blobs used by other snapshots; an unreadable sibling fails closed.
func blobsReferencedByOtherSnapshots(repoDir, commit string) (map[string]bool, error) {
	snapshotsRoot := filepath.Join(repoDir, "snapshots")
	blobDir := filepath.Join(repoDir, "blobs")
	snapshots, err := os.ReadDir(snapshotsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}
		return nil, err
	}

	referenced := map[string]bool{}
	for _, snapshot := range snapshots {
		if !snapshot.IsDir() || snapshot.Name() == commit {
			continue
		}
		snapshotDir := filepath.Join(snapshotsRoot, snapshot.Name())
		entries, err := os.ReadDir(snapshotDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read sibling snapshot %s: %w", snapshot.Name(), err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if blobPath, ok := containedBlobPath(blobDir, filepath.Join(snapshotDir, entry.Name())); ok {
				referenced[blobPath] = true
			}
		}
	}
	return referenced, nil
}
