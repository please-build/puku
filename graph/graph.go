package graph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/please-build/buildtools/build"
	"github.com/please-build/buildtools/labels"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/edit"
	"github.com/please-build/puku/logging"
	"github.com/please-build/puku/options"
)

var log = logging.GetLogger()

type Dependency struct {
	From, To labels.Label
}

type Graph struct {
	buildFileNames   []string
	files            map[string]*build.File
	deps             []*Dependency
	experimentalDirs []string
	opts             options.Options
}

func New(buildFileNames []string, opts options.Options) *Graph {
	return &Graph{
		buildFileNames: buildFileNames,
		files:          map[string]*build.File{},
		opts:           opts,
	}
}

func (g *Graph) WithExperimentalDirs(dirs ...string) *Graph {
	g.experimentalDirs = dirs
	return g
}

func (g *Graph) LoadFile(path string) (*build.File, error) {
	if f, ok := g.files[path]; ok {
		return f, nil
	}

	f, err := g.loadFile(path)
	if err != nil {
		return nil, err
	}

	g.files[path] = f
	f.Pkg = path

	return f, nil
}

// SetFile can be used to override a filepath with a given build file. This is useful for testing.
func (g *Graph) SetFile(path string, file *build.File) {
	g.files[path] = file
}

func (g *Graph) isExperimental(label labels.Label) bool {
	for _, e := range g.experimentalDirs {
		if strings.HasPrefix(label.Package, e) {
			return true
		}
	}
	return false
}

// EnsureVisibility registers a dependency between two targets in different packages. This is used to ensure the targets are
// visible to each other.
func (g *Graph) EnsureVisibility(from, to string) {
	if strings.HasPrefix(to, "///") {
		return // Can't update visibility in subrepos
	}

	fromLabel := labels.Parse(from)
	toLabel := labels.Parse(to)

	if g.isExperimental(fromLabel) {
		return // Experimental dirs are given visibility to all other packages
	}

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

func (g *Graph) FormatFilesWithWriter(out io.Writer, format string) error {
	if err := g.ensureVisibilities(); err != nil {
		return err
	}
	for _, file := range g.files {
		if err := writeFormattedBuildFile(file, out, format, g.opts); err != nil {
			return err
		}
	}
	return nil
}

func (g *Graph) FormatFiles() error {
	if err := g.ensureVisibilities(); err != nil {
		return err
	}
	for _, file := range g.files {
		if err := saveFormattedBuildFile(file, g.opts); err != nil {
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
			log.Warningf("failed to set visibility: %v", err)
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

	t := edit.FindTargetByName(f, dep.To.Target)
	if t == nil {
		return fmt.Errorf("failed can't find target %v (depended on by %v)", dep.To.Format(), dep.From.Format())
	}

	kind := conf.GetKind(t.Kind())
	if kind == nil {
		return nil
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

		// We're cheating slightly by using buildtools' labels.Parse function to parse visibility
		// identifiers, because "..." is permitted in a visibility identifier but not in build target
		// names, but it'll work fine provided we handle the "..." case differently.
		visibility := labels.Parse(v)

		if filepath.Base(visibility.Package) == "..." {
			pkg := filepath.Dir(visibility.Package)
			// filepath.Dir returns "." if visibility.Package contains no package name component (i.e., if
			// the visibility identifier is "//...") - translate this into the empty package name.
			if pkg == "." {
				pkg = ""
			}
			if isSubpackage(pkg, target.Package) {
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

// isSubpackage returns true if pkg is the name of a package whose BUILD file would exist within the
// directory tree belonging to the package basePkg, or false otherwise.
//
// A basePkg consisting of the empty string denotes the top-level package; given the definition of
// "sub-package" above, it follows that all packages are sub-packages of the top-level package.
func isSubpackage(basePkg, pkg string) bool {
	if basePkg == "" {
		return true
	}
	basePkgParts := strings.Split(basePkg, "/")
	pkgParts := strings.Split(pkg, "/")
	if len(basePkgParts) > len(pkgParts) {
		return false
	}
	for i := range basePkgParts {
		if basePkgParts[i] != pkgParts[i] {
			return false
		}
	}
	return true
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

// writeFormattedBuildFile writes a build file to the given writer if puku has made meaningful changes.
//
// See the comment on outputFormattedBuildFile for more details.
func writeFormattedBuildFile(buildFile *build.File, out io.Writer, format string, opts options.Options) error {
	outFn := func() (io.WriteCloser, error) {
		return nopCloser{out}, nil
	}
	return outputFormattedBuildFile(buildFile, outFn, format, opts)
}

// saveFormattedBuildFile writes a build file to disk if puku has made meaningful changes.
//
// See the comment on outputFormattedBuildFile for more details.
func saveFormattedBuildFile(buildFile *build.File, opts options.Options) error {
	outFn := func() (io.WriteCloser, error) {
		return os.Create(buildFile.Path)
	}

	return outputFormattedBuildFile(buildFile, outFn, "text", opts)
}

// outputFormattedBuildFile writes a build file to the given writer if puku has made meaningful changes.
//
// To avoid churn and changes to files where puku has not changed anything, checking for changes is
// done by comparing the formatted build file without applying rewriting (which roughly means linter
// changes). If changes do exist and skipRewriting is not true, the rewriting is applied to ensure
// the resulting build file will satisfy `plz fmt`.
//
// This takes a function to obtain the writer because this needs to read the file to check if puku
// has made any changes before it writes to it. If saveFormattedBuildFile called os.Create
// proactively, the file would be truncated, and so we'd always try to write to it.
func outputFormattedBuildFile(buildFile *build.File, outFn func() (io.WriteCloser, error), format string, opts options.Options) error {
	if len(buildFile.Stmt) == 0 {
		return nil
	}

	content := build.FormatWithoutRewriting(buildFile)

	actual, err := os.ReadFile(buildFile.Path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		actual = nil
	}

	if bytes.Equal(content, actual) {
		return nil
	}

	w, err := outFn()
	if err != nil {
		return err
	}
	defer w.Close()

	if !opts.SkipRewriting {
		content = build.Format(buildFile)
	}

	switch format {
	case "text":
		_, err := w.Write(content)
		return err
	case "json":
		e := json.NewEncoder(w)
		return e.Encode(struct{ Path, Content string }{Path: buildFile.Path, Content: string(content)})
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
}
