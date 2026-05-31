package runtimepath

import (
	"os"
	"path/filepath"
)

const DirName = "icoo_runtime"

func Root() string {
	workDir, err := os.Getwd()
	if err != nil {
		workDir = "."
	}
	if abs, err := filepath.Abs(workDir); err == nil {
		workDir = abs
	}
	return filepath.Join(workDir, DirName)
}

func Join(elem ...string) string {
	parts := append([]string{Root()}, elem...)
	return filepath.Join(parts...)
}
