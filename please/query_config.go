package please

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Config struct {
	Plugin struct {
		Go struct {
			ImportPath []string `json:"importpath"`
		} `json:"go"`
	} `json:"plugin"`
	Parse struct {
		BuildFileName      []string `json:"buildfilename"`
		PreloadSubincludes []string `json:"preloadsubincludes"`
	} `json:"parse"`
}

func (c *Config) ImportPath() string {
	paths := c.Plugin.Go.ImportPath
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func (c *Config) GoIsPreloaded() bool {
	for _, i := range c.Parse.PreloadSubincludes {
		if i == "///go//build_defs:go" {
			return true
		}
	}
	return false
}

func (c *Config) BuildFileNames() []string {
	return c.Parse.BuildFileName
}

func QueryConfig(plzTool string) (*Config, error) {
	cmd := exec.Command(plzTool, "query", "config", "--json")
	stdErr := new(bytes.Buffer)
	cmd.Stderr = stdErr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%v\n%v", err, stdErr.String())
	}

	c := new(Config)
	if err := json.Unmarshal(out, c); err != nil {
		return nil, err
	}
	return c, nil
}

func FindRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return findRepoRoot(dir)
}

func findRepoRoot(path string) (string, error) {
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
	return findRepoRoot(filepath.Dir(path))
}
