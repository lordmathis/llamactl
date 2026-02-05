package models

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type CachedModel struct {
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

var modelFilenamePattern = regexp.MustCompile(`^([^_]+)_([^_]+)_(.+)$`)
var splitFilePattern = regexp.MustCompile(`^(.+)-\d{5}-of-\d{5}(\..+)$`)

func ScanCache(cacheDir string) ([]CachedModel, error) {
	var models []CachedModel

	_, err := os.Stat(cacheDir)
	if os.IsNotExist(err) {
		return models, nil
	}

	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, err
	}

	fileMap := make(map[string]*CachedModel)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		if filepath.Ext(filename) == ".tmp" {
			continue
		}

		matches := modelFilenamePattern.FindStringSubmatch(filename)
		if matches == nil {
			continue
		}

		org := matches[1]
		repo := matches[2]
		baseFilename := matches[3]

		repoName := org + "/" + repo

		fileInfo, err := entry.Info()
		if err != nil {
			continue
		}

		fileType := "unknown"
		if filepath.Ext(baseFilename) == ".gguf" {
			fileType = "gguf"
		} else if baseFilename == "preset.ini" {
			fileType = "preset"
		} else if baseFilename == "" && filename != "" {
			baseFilename = filename
		}

		modelKey := repoName
		tag := extractTag(baseFilename)
		modelKey = repoName + ":" + tag

		if _, exists := fileMap[modelKey]; !exists {
			fileMap[modelKey] = &CachedModel{
				Repo:  repoName,
				Tag:   tag,
				Files: []File{},
			}
		}

		model := fileMap[modelKey]

		filePath := filepath.Join(cacheDir, filename)
		model.Files = append(model.Files, File{
			Name:      filename,
			Path:      filePath,
			SizeBytes: fileInfo.Size(),
			Type:      fileType,
		})

		model.SizeBytes += fileInfo.Size()
	}

	for _, model := range fileMap {
		models = append(models, *model)
	}

	return models, nil
}

func extractTag(filename string) string {
	if filename == "preset.ini" {
		return "latest"
	}

	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)

	// Handle split files: remove -00001-of-00003 pattern
	if splitMatches := splitFilePattern.FindStringSubmatch(base); splitMatches != nil {
		base = splitMatches[1]
	}

	// For cache filenames like "Llama-3.2-3B-Instruct-GGUF_Q4_K_M",
	// the tag is everything after the first underscore
	// But the filename already has org_repo prefix stripped at this point
	if idx := strings.Index(base, "_"); idx != -1 {
		// Everything after first underscore is the tag
		return base[idx+1:]
	}

	// If no underscore, the whole base is the tag
	if base != "" {
		return base
	}

	return "latest"
}

func (m *Manager) ListCached() ([]CachedModel, error) {
	return ScanCache(m.cacheDir)
}

func (m *Manager) DeleteModel(repo, tag string) error {
	models, err := m.ListCached()
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
