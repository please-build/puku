package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/please-build/puku/kinds"
)

// KindConfig represents the configuration for a custom kind. See kinds.Kind for more information on how kinds work.
type KindConfig struct {
	// NonGoSources indicates that this rule operates on non-go sources and we shouldn't attempt to parse them to
	// generate the deps list. This is the case for rules like proto_library that still output a go package, but we
	// shouldn't try to update their deps based on their sources.
	NonGoSources      bool     `json:"nonGoSources"`
	ProvidedDeps      []string `json:"providedDeps"`
	DefaultVisibility []string `json:"defaultVisibility"`
}

// Config represents a puku.json file discovered in the repo. These are loaded for each directory, and form a chain of
// configs all the way up to the root config. Configs at a deeper level in the file tree override values from configs at
// a shallower level. The shallower cofig file is stored in (*Config).base` and the methods on this struct will recurse
// into this base config where appropriate.
type Config struct {
	base          *Config
	ThirdPartyDir string                 `json:"thirdPartyDir"`
	PleasePath    string                 `json:"pleasePath"`
	KnownTargets  map[string]string      `json:"knownTargets"`
	LibKinds      map[string]*KindConfig `json:"libKinds"`
	TestKinds     map[string]*KindConfig `json:"testKinds"`
	BinKinds      map[string]*KindConfig `json:"binKinds"`
	Stop          bool                   `json:"stop"`
}

// TODO we should reload this during plz watch so this probably needs to become a member of Update
// configs contains a cache of configs for a given directory
var configs = map[string]*Config{}

// ReadConfig builds up the config for a given path
func ReadConfig(dir string) (*Config, error) {
	var parts []string
	if dir != "." {
		parts = strings.Split(dir, "/")
	}

	c, err := readConfigs(nil, ".", parts)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return new(Config), nil
	}
	return c, nil
}

// readConfigs descends through the parts reading any config files it finds, building up the config chain.
func readConfigs(base *Config, path string, rest []string) (*Config, error) {
	c, err := readOneConfig(path)
	if err != nil {
		return nil, err
	}

	if c != nil {
		c.base = base
		base = c
	}

	if len(rest) == 0 {
		return base, nil
	}

	return readConfigs(base, filepath.Join(path, rest[0]), rest[1:])
}

// readOneConfig reads a config in a directory
func readOneConfig(path string) (*Config, error) {
	if config, ok := configs[path]; ok {
		return config, nil
	}
	f, err := os.ReadFile(filepath.Join(path, "puku.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	c := new(Config)
	if err := json.Unmarshal(f, c); err != nil {
		return nil, err
	}

	configs[path] = c
	return c, nil
}

func (c *Config) GetThirdPartyDir() string {
	if c.ThirdPartyDir != "" {
		return c.ThirdPartyDir
	}
	if c.base != nil {
		return c.base.GetThirdPartyDir()
	}
	return "third_party/go"
}

func (c *Config) GetStop() bool {
	return c.Stop
}

func (c *Config) GetKnownTarget(importPath string) string {
	if t, ok := c.KnownTargets[importPath]; ok {
		return t
	}
	if c.base != nil {
		return c.base.GetKnownTarget(importPath)
	}
	return ""
}

func (c *Config) GetPlzPath() string {
	if c.PleasePath != "" {
		return c.PleasePath
	}
	if c.base != nil {
		return c.base.GetPlzPath()
	}
	return "plz"
}

func (c *Config) GetKind(kind string) *kinds.Kind {
	if k, ok := kinds.DefaultKinds[kind]; ok {
		return k
	}

	if k, ok := c.LibKinds[kind]; ok {
		return &kinds.Kind{
			Name:              kind,
			Type:              kinds.Lib,
			ProvidedDeps:      k.ProvidedDeps,
			DefaultVisibility: k.DefaultVisibility,
		}
	}
	if k, ok := c.TestKinds[kind]; ok {
		return &kinds.Kind{
			Name:         kind,
			Type:         kinds.Test,
			ProvidedDeps: k.ProvidedDeps,
		}
	}
	if k, ok := c.BinKinds[kind]; ok {
		return &kinds.Kind{
			Name:         kind,
			Type:         kinds.Bin,
			ProvidedDeps: k.ProvidedDeps,
		}
	}
	if c.base != nil {
		return c.base.GetKind(kind)
	}
	return nil
}
