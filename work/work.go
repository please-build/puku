package work

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/buildtools/labels"
	"github.com/please-build/puku/config"
)

func ExpandPaths(origWD string, paths []string) ([]string, error) {
	ret := make([]string, 0, len(paths))
	for _, path := range paths {
		// Handle using build label style syntax a bit like `plz build`
		if strings.HasPrefix(path, "//") {
			l := labels.Parse(path)
			path = l.Package
		} else {
			if strings.HasPrefix(path, ":") {
				path = "."
			}
			// Join the path with the original working directory. We would have cd'ed to the root of the plz repo by this
			// point
			path = filepath.Join(origWD, path)
		}

		if filepath.Base(path) != "..." {
			ret = append(ret, path)
		}

		path = filepath.Dir(path)
		err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !d.IsDir() {
				return nil
			}
			if d.Name() == "plz-out" {
				return filepath.SkipDir
			}
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			conf, err := config.ReadConfig(path)
			if err != nil {
				return err
			}

			if conf.GetStop() {
				return filepath.SkipDir
			}
			ret = append(ret, path)
			return nil
		})

		if err != nil {
			return nil, err
		}
	}
	return ret, nil
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
