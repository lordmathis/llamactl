package models

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileManager struct {
	cacheDir string
}

func NewFileManager(cacheDir string) *FileManager {
	return &FileManager{
		cacheDir: cacheDir,
	}
}

// sanitizePathComponent removes dangerous path elements to prevent path traversal attacks.
// It removes path separators, parent directory references, and ensures the component
// is safe to use in a file path.
func sanitizePathComponent(component string) string {
	// Replace all path separators with safe characters
	component = strings.ReplaceAll(component, string(filepath.Separator), "_")
	component = strings.ReplaceAll(component, "/", "_")
	component = strings.ReplaceAll(component, "\\", "_")

	// Remove or replace parent directory references
	component = strings.ReplaceAll(component, "..", "_")

	// Remove any remaining path-related characters
	component = strings.TrimPrefix(component, ".")
	component = strings.TrimSpace(component)

	// If empty after sanitization, use a default
	if component == "" {
		component = "unknown"
	}

	return component
}

// sanitizeRepoPath sanitizes a repository path by removing path traversal attempts
// while preserving forward slashes for proper repo format (e.g., "org/model").
func sanitizeRepoPath(repo string) string {
	// Split by forward slash and sanitize each component
	parts := strings.Split(repo, "/")
	sanitized := make([]string, 0, len(parts))

	for _, part := range parts {
		// Remove dangerous patterns from each part
		part = strings.TrimSpace(part)
		// Remove parent directory references
		part = strings.ReplaceAll(part, "..", "_")
		// Remove backslashes
		part = strings.ReplaceAll(part, "\\", "_")
		// Remove leading dots (except for valid cases)
		for strings.HasPrefix(part, ".") {
			part = strings.TrimPrefix(part, ".")
		}

		// Skip empty parts
		if part != "" {
			sanitized = append(sanitized, part)
		}
	}

	// If nothing left after sanitization, use default
	if len(sanitized) == 0 {
		return "unknown"
	}

	return strings.Join(sanitized, "/")
}

func (fm *FileManager) GetPath(repo, filename string) string {
	return filepath.Join(fm.cacheDir, fm.getCacheFilename(repo, filename))
}

func (fm *FileManager) getCacheFilename(repo, filename string) string {
	// Sanitize repo (preserving slashes) then convert slashes to underscores for flat filename
	safeRepo := sanitizeRepoPath(repo)
	safeRepo = strings.ReplaceAll(safeRepo, "/", "_")

	// Sanitize filename (remove all path separators)
	safeFilename := sanitizePathComponent(filename)

	return safeRepo + "_" + safeFilename
}

func (fm *FileManager) GetManifestPath(repo, tag string) string {
	// Sanitize repo path and tag to prevent path traversal
	safeRepo := sanitizeRepoPath(repo)
	safeTag := sanitizePathComponent(tag)

	// Replace slashes with equals for manifest format (manifest=org=model=tag.json)
	repoWithEquals := strings.ReplaceAll(safeRepo, "/", "=")

	return filepath.Join(fm.cacheDir, fmt.Sprintf("manifest=%s=%s.json", repoWithEquals, safeTag))
}

func (fm *FileManager) GetETagPath(repo, filename string) string {
	return fm.GetPath(repo, filename) + ".etag"
}

func (fm *FileManager) GetSplitFilename(baseFilename string, part, total int) string {
	ext := filepath.Ext(baseFilename)
	base := strings.TrimSuffix(baseFilename, ext)
	return fmt.Sprintf("%s-%05d-of-%05d%s", base, part, total, ext)
}

func (fm *FileManager) RenameToFinal(tempPath, repo, filename string, cleanup *tempFileCleanup) error {
	finalPath := fm.GetPath(repo, filename)
	if err := os.Rename(tempPath, finalPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}
	cleanup.remove(tempPath)
	return nil
}

func (fm *FileManager) RenameSplitFiles(repo, baseFilename string, splitCount int, cleanup *tempFileCleanup) error {
	if splitCount <= 1 {
		return nil
	}

	for i := 2; i <= splitCount; i++ {
		splitFilename := fm.GetSplitFilename(baseFilename, i, splitCount)
		tempPath := fm.GetPath(repo, splitFilename+".tmp")
		finalPath := fm.GetPath(repo, splitFilename)

		if err := os.Rename(tempPath, finalPath); err != nil {
			return fmt.Errorf("failed to rename split file %d: %w", i, err)
		}
		cleanup.remove(tempPath)
	}

	return nil
}
