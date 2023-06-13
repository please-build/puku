package work

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// WalkDir behaves like filepath.WalkDir but it skips directories we always want to ignore
func WalkDir(path string, f fs.WalkDirFunc) error {
	return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if d.Name() == "plz-out" {
			return filepath.SkipDir
		}
		if d.Name() == ".git" {
			return filepath.SkipDir
		}
		if err := f(path, d, err); err != nil {
			return err
		}
		return nil
	})
}

// FindRoot finds the root of the workspace
func FindRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return findRoot(dir)
}

func findRoot(path string) (string, error) {
	if path == "." {
		return "", errors.New("failed to locate please repo root: no .plzconfig found")
	}
	info, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}

	for _, i := range info {
		if i.IsDir() {
			continue
		}
		if i.Name() == ".plzconfig" {
			return path, nil
		}
	}
	return findRoot(filepath.Dir(path))
}
