package graph

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/labels"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/edit"
	"github.com/please-build/puku/fs"
)

type Dependency struct {
	From, To labels.Label
}

type Graph struct {
	buildFileNames []string
	files          map[string]*build.File
	deps           []*Dependency
}

func New(buildFileNames []string) *Graph {
	return &Graph{
		buildFileNames: buildFileNames,
		files:          map[string]*build.File{},
	}
}

func (g *Graph) LoadFile(path string) (*build.File, error) {
	if f, ok := g.files[path]; ok {
		return f, nil
	}

	f, err := g.loadFile(path)
	if err == nil {
		g.files[path] = f
	}
	return f, err
}

// EnsureVisibility registers a dependency between two targets in different packages. This is used to ensure the targets are
// visible to each other.
func (g *Graph) EnsureVisibility(from, to string) {
	if strings.HasPrefix(to, "///") {
		return // Can't update visibility in subrepos
	}

	fromLabel := labels.Parse(from)
	toLabel := labels.Parse(to)

	if strings.HasPrefix(to, ":") || fromLabel.Package == toLabel.Package {
		return // Don't need visibility between targets in the same package
	}

	g.deps = append(g.deps, &Dependency{
		From: fromLabel,
		To:   toLabel,
	})
}

func (g *Graph) loadFile(path string) (*build.File, error) {
	validFilename := ""
	for _, name := range g.buildFileNames {
		filePath := filepath.Join(path, name)
		if f, err := os.Lstat(filePath); os.IsNotExist(err) {
			// This file name is available. Use the first one we find in the list.
			if validFilename == "" {
				validFilename = filePath
			}
		} else if !f.IsDir() { // this is a common issue on macos where paths are case insensitive...
			bs, err := os.ReadFile(filePath)
			if err != nil {
				return nil, err
			}
			file, err := build.ParseBuild(filePath, bs)
			return file, err
		}
	}
	if validFilename == "" {
		return nil, fmt.Errorf("folders exist with the build file names in directory %v %v", path, g.buildFileNames)
	}

	// Otherwise returns a new empty file. We didn't find one.
	return build.ParseBuild(validFilename, nil)
}

func (g *Graph) FormatFiles(write bool, out io.Writer) error {
	if err := g.ensureVisibilities(); err != nil {
		return err
	}
	for _, file := range g.files {
		if err := saveAndFormatBuildFile(file, write, out); err != nil {
			return err
		}
	}
	return nil
}

func (g *Graph) ensureVisibilities() error {
	for _, dep := range g.deps {
		conf, err := config.ReadConfig(dep.To.Package)
		if err != nil {
			return err
		}
		if err := g.ensureVisibility(conf, dep); err != nil {
			return fmt.Errorf("failed to set visibility: %v", err)
		}
	}
	return nil
}

func getDefaultVisibility(f *build.File) []string {
	for _, r := range f.Rules("package") {
		if vis := r.AttrStrings("default_visibility"); len(vis) > 0 {
			return vis
		}
	}
	return nil
}

func (g *Graph) ensureVisibility(conf *config.Config, dep *Dependency) error {
	f, err := g.LoadFile(dep.To.Package)
	if err != nil {
		return err
	}

	t := findTargetByName(f, dep.To.Target)
	if t == nil {
		return fmt.Errorf("failed can't find target %v (depended on by %v)", dep.To.Format(), dep.From.Format())
	}

	kind := conf.GetKind(t.Kind())
	if kind == nil {
		return fmt.Errorf("%v tried to depend on %v, but %v is of unknown kind: %v", dep.From.Format(), dep.To.Format(), dep.To.Format(), t.Kind())
	}

	visibilities := t.AttrStrings("visibility")

	defaultVis := visibilities
	if len(defaultVis) == 0 {
		defaultVis = kind.DefaultVisibility
	}
	if len(defaultVis) == 0 {
		defaultVis = getDefaultVisibility(f)
	}
	if checkVisibility(dep.From, defaultVis) {
		return nil
	}

	vis := dep.From
	vis.Target = "all"
	t.SetAttr("visibility", edit.NewStringList(append(visibilities, vis.Format())))
	return nil
}

func checkVisibility(target labels.Label, visibilities []string) bool {
	for _, v := range visibilities {
		if v == "PUBLIC" {
			return true
		}

		visibility := labels.Parse(v)

		if filepath.Base(visibility.Package) == "..." {
			pkg := filepath.Dir(visibility.Package)
			if fs.IsSubdir(pkg, target.Package) {
				return true
			}
			continue
		}

		if visibility.Package != target.Package {
			continue
		}

		if visibility.Target == target.Target || visibility.Target == "all" {
			return true
		}
	}
	return false
}

func findTargetByName(file *build.File, name string) *build.Rule {
	for _, rule := range file.Rules("") {
		if rule.Name() == name {
			return rule
		}
	}
	return nil
}

func saveAndFormatBuildFile(buildFile *build.File, write bool, out io.Writer) error {
	if len(buildFile.Stmt) == 0 {
		return nil
	}

	if write {
		f, err := os.Create(buildFile.Path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(build.Format(buildFile))
		return err
	}
	target := build.Format(buildFile)
	actual, err := os.ReadFile(buildFile.Path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		actual = nil
	}

	if !bytes.Equal(target, actual) {
		_, err := out.Write(target)
		return err
	}

	return nil
}
