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

var splitFilePattern = regexp.MustCompile(`-\d{5}-of-(\d{5})\.gguf$`)

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
	return strings.TrimSpace(string(data)), nil
}

func scanHFRepo(hubRoot, dirName, repo, commit, nodeName string) CachedModel {
	model := CachedModel{
		Node:  nodeName,
		Repo:  repo,
		Tag:   commit, // Default tag to commit hash
		Files: []File{},
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
	hubRoot := fileManager.HFHubRoot()
	repoDirName := fileManager.HFRepoDirName(repo)
	repoDir := filepath.Join(hubRoot, repoDirName)

	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		return fmt.Errorf("model not found: %s", repo)
	}

	if tag == "" {
		return os.RemoveAll(repoDir)
	}

	commit, err := readRefFile(filepath.Join(repoDir, "refs", tag))
	if err != nil {
		return fmt.Errorf("model not found: %s:%s", repo, tag)
	}

	snapshotDir := filepath.Join(repoDir, "snapshots", commit)
	if _, err := os.Stat(snapshotDir); os.IsNotExist(err) {
		return fmt.Errorf("model not found: %s:%s", repo, tag)
	}

	snapshotEntries, err := os.ReadDir(snapshotDir)
	if err != nil {
		return err
	}

	blobPathsInSnapshot := make(map[string]bool)
	for _, snapshotEntry := range snapshotEntries {
		if snapshotEntry.IsDir() {
			continue
		}
		symlinkPath := filepath.Join(snapshotDir, snapshotEntry.Name())
		blobPath, err := os.Readlink(symlinkPath)
		if err != nil {
			blobPath = symlinkPath
		} else if !filepath.IsAbs(blobPath) {
			blobPath = filepath.Join(filepath.Dir(symlinkPath), blobPath)
		}
		blobPathsInSnapshot[blobPath] = true
		os.Remove(symlinkPath)
	}

	otherSnapshots, _ := os.ReadDir(filepath.Join(repoDir, "snapshots"))
	blobPathsInOtherSnapshots := make(map[string]bool)
	for _, otherSnapshot := range otherSnapshots {
		if otherSnapshot.Name() == commit {
			continue
		}
		otherSnapshotDir := filepath.Join(repoDir, "snapshots", otherSnapshot.Name())
		otherEntries, _ := os.ReadDir(otherSnapshotDir)
		for _, otherEntry := range otherEntries {
			if otherEntry.IsDir() {
				continue
			}
			symlinkPath := filepath.Join(otherSnapshotDir, otherEntry.Name())
			blobPath, err := os.Readlink(symlinkPath)
			if err != nil {
				blobPath = symlinkPath
			} else if !filepath.IsAbs(blobPath) {
				blobPath = filepath.Join(filepath.Dir(symlinkPath), blobPath)
			}
			blobPathsInOtherSnapshots[blobPath] = true
		}
	}

	for blobPath := range blobPathsInSnapshot {
		if !blobPathsInOtherSnapshots[blobPath] {
			os.Remove(blobPath)
		}
	}

	os.Remove(snapshotDir)
	refPath := filepath.Join(repoDir, "refs", tag)
	os.Remove(refPath)

	return nil
}
