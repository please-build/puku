package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/muhammadmuzzammil1998/jsonc"
	"github.com/please-build/puku/kinds"
	"github.com/please-build/puku/logging"
)

var log = logging.GetLogger()

// KindConfig represents the configuration for a custom kind. See kinds.Kind for more information on how kinds work.
type KindConfig struct {
	// NonGoSources indicates that this rule operates on non-go sources and we shouldn't attempt to parse them to
	// generate the deps list. This is the case for rules like proto_library that still output a go package, but we
	// shouldn't try to update their deps based on their sources.
	NonGoSources      bool     `json:"nonGoSources"`
	ProvidedDeps      []string `json:"providedDeps"`
	DefaultVisibility []string `json:"defaultVisibility"`
	SrcsArg           string   `json:"srcsArg"`
}

func (kc *KindConfig) srcsArg() string {
	if kc.SrcsArg == "" {
		return "srcs"
	}
	return kc.SrcsArg
}

// Config represents a puku.json file discovered in the repo. These are loaded for each directory, and form a chain of
// configs all the way up to the root config. Configs at a deeper level in the file tree override values from configs at
// a shallower level. The shallower config file is stored in (*Config).base` and the methods on this struct will recurse
// into this base config where appropriate.
type Config struct {
	base                *Config
	ThirdPartyDir       string                 `json:"thirdPartyDir"`
	PleasePath          string                 `json:"pleasePath"`
	KnownTargets        map[string]string      `json:"knownTargets"`
	LibKinds            map[string]*KindConfig `json:"libKinds"`
	TestKinds           map[string]*KindConfig `json:"testKinds"`
	BinKinds            map[string]*KindConfig `json:"binKinds"`
	Stop                *bool                  `json:"stop"`
	EnsureSubincludes   *bool                  `json:"ensureSubincludes"`
	ExcludeBuiltinKinds []string               `json:"excludeBuiltinKinds"`
}

// TODO we should reload this during plz watch so this probably needs to become a member of Update
// configs contains a cache of configs for a given directory
var configs = map[string]*Config{}

// ReadConfig builds up the config for a given path
func ReadConfig(dir string) (*Config, error) {
	dir = filepath.Clean(dir)
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
			c := new(Config)
			configs[path] = c
			return c, nil
		}
		return nil, err
	}

	c := new(Config)
	if err := json.Unmarshal(f, c); err != nil {
		return nil, fmt.Errorf("in %s: %w", path, err)
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
	if c.Stop != nil {
		return *c.Stop
	}
	return c.base != nil && c.base.GetStop()
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

func (c *Config) ShouldEnsureSubincludes() bool {
	if c.EnsureSubincludes != nil {
		return *c.EnsureSubincludes
	}
	if c.base != nil {
		return c.base.ShouldEnsureSubincludes()
	}
	return true
}

func (c *Config) isExcludedDefaultKind(kind string) bool {
	for _, c := range c.ExcludeBuiltinKinds {
		if c == kind {
			return true
		}
	}
	if c.base == nil {
		return false
	}
	return c.base.isExcludedDefaultKind(kind)
}

func (c *Config) GetKind(kind string) *kinds.Kind {
	if k, ok := c.LibKinds[kind]; ok {
		return &kinds.Kind{
			Name:              kind,
			Type:              kinds.Lib,
			ProvidedDeps:      k.ProvidedDeps,
			SrcsAttr:          k.srcsArg(),
			DefaultVisibility: k.DefaultVisibility,
			NonGoSources:      k.NonGoSources,
		}
	}
	if k, ok := c.TestKinds[kind]; ok {
		return &kinds.Kind{
			Name:         kind,
			Type:         kinds.Test,
			ProvidedDeps: k.ProvidedDeps,
			SrcsAttr:     k.srcsArg(),
			NonGoSources: k.NonGoSources,
		}
	}
	if k, ok := c.BinKinds[kind]; ok {
		return &kinds.Kind{
			Name:         kind,
			Type:         kinds.Bin,
			ProvidedDeps: k.ProvidedDeps,
			SrcsAttr:     k.srcsArg(),
			NonGoSources: k.NonGoSources,
		}
	}
	if c.base != nil {
		return c.base.GetKind(kind)
	}

	if k, ok := kinds.DefaultKinds[kind]; ok {
		if !c.isExcludedDefaultKind(kind) {
			return k
		}
	}
	return nil
}

// TSConfig represents a tsconfig.json file discovered in the repo.
type TSConfig struct {
	Dir             string
	CompilerOptions struct {
		Paths map[string][]string `json:"paths"`
	} `json:"compilerOptions"`
}

var tsconfigs = map[string]*TSConfig{}

// ReadTSConfig finds the closest tsconfig by walking up the directory tree.
// Note: we don't try and resolve the full config inheritance, it just resolves
// to the first tsconfig.json file that we find.
func ReadTSConfig(dir string) (*TSConfig, error) {
	origDir := dir
	dir = filepath.Clean(dir)

	for true {
		if c, ok := tsconfigs[dir]; ok {
			return c, nil
		}

		f, err := os.ReadFile(filepath.Join(dir, "tsconfig.json"))
		if err != nil {
			if os.IsNotExist(err) {
				if dir == "." {
					break
				}

				// try parent directory
				dir = filepath.Dir(dir)
				continue
			}
			return nil, err
		}

		c := new(TSConfig)
		c.Dir = dir
		// tsconfig files are jsonc: https://code.visualstudio.com/docs/languages/json#_json-with-comments
		if err := jsonc.Unmarshal(f, c); err != nil {
			return nil, fmt.Errorf("in %s: %w", dir, err)
		}
		tsconfigs[dir] = c
		return c, nil
	}

	log.Debugf("Can't find tsconfig for dir: %s", origDir)
	return nil, nil
}
