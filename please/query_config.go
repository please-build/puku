package please

import (
	"encoding/json"
)

type Config struct {
	Plugin struct {
		Go struct {
			ImportPath []string `json:"importpath"`
			Modfile    []string `json:"modfile"`
		} `json:"go"`
	} `json:"plugin"`
	Parse struct {
		BuildFileName      []string `json:"buildfilename"`
		PreloadSubincludes []string `json:"preloadsubincludes"`
		ExperimentalDir    []string `json:"experimentaldir"`
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

func (c *Config) ModFile() string {
	if c == nil {
		return ""
	}

	if len(c.Plugin.Go.Modfile) == 0 {
		return ""
	}

	return c.Plugin.Go.Modfile[0]
}

func QueryConfig(plzTool string) (*Config, error) {
	out, err := execPlease(plzTool, "query", "config", "--json")
	if err != nil {
		return nil, err
	}
	c := new(Config)
	if err := json.Unmarshal(out, c); err != nil {
		return nil, err
	}
	return c, nil
}
