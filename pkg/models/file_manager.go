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

func (fm *FileManager) GetPath(repo, filename string) string {
	return filepath.Join(fm.cacheDir, fm.getCacheFilename(repo, filename))
}

func (fm *FileManager) getCacheFilename(repo, filename string) string {
	return strings.ReplaceAll(repo, "/", "_") + "_" + filename
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
