package utils

import (
	"os"
	"path/filepath"
)

var inputPath string

func InitPath(i string) {
	inputPath = i
}

func RenameFolderMediaInputPath(oldPath string, newName string) error {
	dir := filepath.Dir(oldPath)
	parentDir := filepath.Dir(dir)
	newPath := filepath.Join(parentDir, newName)
	return os.Rename(dir, newPath)
}
