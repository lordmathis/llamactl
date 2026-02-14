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
	var models []CachedModel = []CachedModel{}

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return models, nil
	} else if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, err
	}

	fileManager := NewFileManager(cacheDir)

	// Scan for manifest files
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		// Look for manifest files: manifest={repo}={tag}.json
		if !strings.HasPrefix(filename, "manifest=") || !strings.HasSuffix(filename, ".json") {
			continue
		}

		repo, tag, err := parseManifestFilename(filename)
		if err != nil {
			continue
		}

		manifest, err := readManifest(filepath.Join(cacheDir, filename))
		if err != nil {
			continue
		}

		cachedModel := buildCachedModel(repo, tag, nodeName, manifest, fileManager)

		// Only include models that have at least one file
		if len(cachedModel.Files) > 0 {
			models = append(models, cachedModel)
		}
	}

	return models, nil
}

func readManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func buildCachedModel(repo, tag, nodeName string, manifest *Manifest, fm *FileManager) CachedModel {
	model := CachedModel{
		Node:  nodeName,
		Repo:  repo,
		Tag:   tag,
		Files: []File{},
	}

	addGGUFFiles(&model, manifest, repo, fm)
	addMMProjFile(&model, manifest, repo, fm)
	addPresetFile(&model, repo, fm)

	return model
}

func addGGUFFiles(model *CachedModel, manifest *Manifest, repo string, fm *FileManager) {
	if manifest.GGUFFile == nil {
		return
	}

	safeFilename := filepath.Base(manifest.GGUFFile.RFilename)
	filePath := fm.GetPath(repo, safeFilename)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return
	}

	model.Files = append(model.Files, File{
		Name:      filepath.Base(filePath),
		Path:      filePath,
		SizeBytes: fileInfo.Size(),
		Type:      "gguf",
	})
	model.SizeBytes += fileInfo.Size()

	// Check for split files
	addSplitFiles(model, safeFilename, repo, fm)
}

func addSplitFiles(model *CachedModel, baseFilename, repo string, fm *FileManager) {
	splitCount, _ := parseSplitCount(baseFilename)
	if splitCount <= 1 {
		return
	}

	for i := 2; i <= splitCount; i++ {
		splitFilename := fm.GetSplitFilename(baseFilename, i, splitCount)
		splitPath := fm.GetPath(repo, splitFilename)

		splitInfo, err := os.Stat(splitPath)
		if err != nil {
			continue
		}

		model.Files = append(model.Files, File{
			Name:      filepath.Base(splitPath),
			Path:      splitPath,
			SizeBytes: splitInfo.Size(),
			Type:      "gguf",
		})
		model.SizeBytes += splitInfo.Size()
	}
}

func addMMProjFile(model *CachedModel, manifest *Manifest, repo string, fm *FileManager) {
	if manifest.MMProjFile == nil {
		return
	}

	mmprojFilename := filepath.Base(manifest.MMProjFile.RFilename)
	mmprojPath := fm.GetPath(repo, mmprojFilename)

	mmprojInfo, err := os.Stat(mmprojPath)
	if err != nil {
		return
	}

	model.Files = append(model.Files, File{
		Name:      filepath.Base(mmprojPath),
		Path:      mmprojPath,
		SizeBytes: mmprojInfo.Size(),
		Type:      "mmproj",
	})
	model.SizeBytes += mmprojInfo.Size()
}

func addPresetFile(model *CachedModel, repo string, fm *FileManager) {
	presetPath := fm.GetPath(repo, "preset.ini")

	presetInfo, err := os.Stat(presetPath)
	if err != nil {
		return
	}

	model.Files = append(model.Files, File{
		Name:      filepath.Base(presetPath),
		Path:      presetPath,
		SizeBytes: presetInfo.Size(),
		Type:      "preset",
	})
	model.SizeBytes += presetInfo.Size()
}

// parseManifestFilename extracts repo and tag from manifest filename
// Format: manifest={part1}={part2}=...={tag}.json
// Example: manifest=bartowski=Qwen=Model-GGUF=Q4_K_M.json
//
//	-> repo: bartowski/Qwen/Model-GGUF, tag: Q4_K_M
func parseManifestFilename(filename string) (repo, tag string, err error) {
	// Strip "manifest=" prefix and ".json" suffix
	if !strings.HasPrefix(filename, "manifest=") || !strings.HasSuffix(filename, ".json") {
		return "", "", fmt.Errorf("invalid manifest filename: %s", filename)
	}

	trimmed := strings.TrimPrefix(filename, "manifest=")
	trimmed = strings.TrimSuffix(trimmed, ".json")

	// Split by "="
	parts := strings.Split(trimmed, "=")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("manifest filename must have at least 2 parts: %s", filename)
	}

	// Last part is tag
	tag = parts[len(parts)-1]

	// Everything before tag is repo (joined with "/")
	repo = strings.Join(parts[:len(parts)-1], "/")

	return repo, tag, nil
}

// parseSplitCount extracts split count from filename pattern
func parseSplitCount(filename string) (int, error) {
	matches := splitFilePattern.FindStringSubmatch(filename)
	if len(matches) == 2 {
		var count int
		if _, err := fmt.Sscanf(matches[1], "%d", &count); err == nil {
			return count, nil
		}
	}
	return 1, nil
}

func (m *Manager) ListCached(nodeName string) ([]CachedModel, error) {
	return ScanCache(m.cacheDir, nodeName)
}

func (m *Manager) DeleteModel(repo, tag string) error {
	models, err := m.ListCached("")
	if err != nil {
		return err
	}

	var filesToDelete []string
	for _, model := range models {
		if model.Repo != repo {
			continue
		}

		if tag != "" && model.Tag != tag {
			continue
		}

		for _, file := range model.Files {
			filesToDelete = append(filesToDelete, file.Path)
		}

		// Also delete manifest file
		manifestPath := m.fileManager.GetManifestPath(repo, model.Tag)
		filesToDelete = append(filesToDelete, manifestPath)

		// Delete ETag files
		for _, file := range model.Files {
			etagPath := file.Path + ".etag"
			filesToDelete = append(filesToDelete, etagPath)
		}
	}

	if len(filesToDelete) == 0 {
		return fmt.Errorf("model not found: %s:%s", repo, tag)
	}

	for _, filePath := range filesToDelete {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}
