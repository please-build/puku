package sync

import (
	"fmt"
	"os"

	"github.com/please-build/buildtools/build"
	"github.com/please-build/buildtools/labels"
	"golang.org/x/mod/modfile"

	"github.com/please-build/puku/config"
	"github.com/please-build/puku/edit"
	"github.com/please-build/puku/graph"
	"github.com/please-build/puku/licences"
	"github.com/please-build/puku/please"
	"github.com/please-build/puku/proxy"
)

type syncer struct {
	plzConf  *please.Config
	graph    *graph.Graph
	licences *licences.Licenses
}

const REPLACE_LABEL = "go_replace_directive"

func newSyncer(plzConf *please.Config, g *graph.Graph) *syncer {
	p := proxy.New(proxy.DefaultURL)
	l := licences.New(p, g)
	return &syncer{
		plzConf:  plzConf,
		graph:    g,
		licences: l,
	}
}

// Sync constructs the syncer struct and initiates the sync.
// NB. the Graph is to be constructed in the calling code because it's useful
// for it to be available outside the package for testing.
func Sync(plzConf *please.Config, g *graph.Graph) error {
	s := newSyncer(plzConf, g)
	if err := s.sync(); err != nil {
		return err
	}
	return s.graph.FormatFiles()
}

// SyncToStdout constructs the syncer and outputs the synced build file to stdout.
func SyncToStdout(format string, plzConf *please.Config, g *graph.Graph) error { //nolint
	s := newSyncer(plzConf, g)
	if err := s.sync(); err != nil {
		return err
	}
	return s.graph.FormatFilesWithWriter(os.Stdout, format)
}

func (s *syncer) sync() error {
	if s.plzConf.ModFile() == "" {
		return nil
	}

	conf, err := config.ReadConfig(".")
	if err != nil {
		return err
	}

	file, err := s.graph.LoadFile(conf.GetThirdPartyDir())
	if err != nil {
		return err
	}

	existingRules, err := s.readModules(file)
	if err != nil {
		return fmt.Errorf("failed to read third party rules: %v", err)
	}

	if err := s.syncModFile(conf, file, existingRules); err != nil {
		return err
	}
	return nil
}

func (s *syncer) syncModFile(conf *config.Config, file *build.File, existingRules map[string]*build.Rule) error {
	outs, err := please.Build(conf.GetPlzPath(), s.plzConf.ModFile())
	if err != nil {
		return err
	}

	if len(outs) != 1 {
		return fmt.Errorf("expected exactly one out from Plugin.Go.Modfile, got %v", len(outs))
	}

	modFile := outs[0]
	bs, err := os.ReadFile(modFile)
	if err != nil {
		return err
	}
	f, err := modfile.Parse(modFile, bs, nil)
	if err != nil {
		return err
	}

	for _, req := range f.Require {
		reqVersion := req.Mod.Version
		var matchingReplace *modfile.Replace
		for _, r := range f.Replace {
			if r.Old.Path == req.Mod.Path {
				matchingReplace = r
				reqVersion = r.New.Version
			}
		}

		// Existing rule will point to the go_mod_download with the version on it so we should use the original path
		rule, ok := existingRules[req.Mod.Path]
		if ok {
			if matchingReplace != nil && matchingReplace.New.Path != req.Mod.Path && rule.Kind() == "go_repo" {
				// Looks like we've added in a replace directive for this module which changes the path, so we need to
				// delete the old go_repo rule and regenerate it with a go_mod_download and a go_repo.
				edit.RemoveTarget(file, rule)
			} else {
				// Make sure the version is up-to-date
				rule.SetAttr("version", edit.NewStringExpr(reqVersion))
				// Add label for the replace directive
				if matchingReplace != nil {
					err := edit.AddLabel(rule, REPLACE_LABEL)
					if err != nil {
						return fmt.Errorf("failed to add replace label to %v: %v", req.Mod.Path, err)
					}
				}
				// No other changes needed
				continue
			}
		}

		ls, err := s.licences.Get(req.Mod.Path, req.Mod.Version)
		if err != nil {
			return fmt.Errorf("failed to get licences for %v: %v", req.Mod.Path, err)
		}

		// If no replace directive, or replace directive is just replacing the version, add a simple rule
		if matchingReplace == nil || matchingReplace.New.Path == req.Mod.Path {
			file.Stmt = append(file.Stmt, edit.NewGoRepoRule(req.Mod.Path, reqVersion, "", ls, []string{REPLACE_LABEL}))
			continue
		}

		dl, dlName := edit.NewModDownloadRule(matchingReplace.New.Path, matchingReplace.New.Version, ls)
		file.Stmt = append(file.Stmt, dl)
		file.Stmt = append(file.Stmt, edit.NewGoRepoRule(req.Mod.Path, "", dlName, nil, []string{REPLACE_LABEL}))
	}

	return nil
}

func (s *syncer) readModules(file *build.File) (map[string]*build.Rule, error) {
	// existingRules contains the rules for modules. These are synced to the go.mod's version as necessary. For modules
	// that use `go_mod_download`, this map will point to that rule as that is the rule that has the version field.
	existingRules := make(map[string]*build.Rule)
	for _, repoRule := range append(file.Rules("go_repo"), file.Rules("go_module")...) {
		if repoRule.AttrString("version") != "" {
			existingRules[repoRule.AttrString("module")] = repoRule
		} else {
			// If we're using a go_mod_download for this module, then find the download rule instead.
			t := labels.ParseRelative(repoRule.AttrString("download"), file.Pkg)
			f, err := s.graph.LoadFile(t.Package)
			if err != nil {
				return nil, err
			}
			existingRules[repoRule.AttrString("module")] = edit.FindTargetByName(f, t.Target)
		}
	}

	return existingRules, nil
}
