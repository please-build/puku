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
	"github.com/please-build/puku/logging"
	"github.com/please-build/puku/please"
	"github.com/please-build/puku/proxy"
)

var log = logging.GetLogger()

type syncer struct {
	plzConf  *please.Config
	graph    *graph.Graph
	licences *licences.Licenses
}

const ReplaceLabel = "go_replace_directive"

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

	// Remove "go_replace_directive" label from any rules which lack a replace directive
	for modPath, rule := range existingRules {
		// Find any matching replace directive
		var matchingReplace *modfile.Replace
		for _, replace := range f.Replace {
			if replace.Old.Path == modPath {
				matchingReplace = replace
			}
		}

		// Remove the replace label if not needed
		if matchingReplace == nil {
			err := edit.RemoveLabel(rule, ReplaceLabel)
			if err != nil {
				log.Warningf("Failed to remove replace label from %v: %v", modPath, err)
			}
		}
	}

	// Check all modules listed in go.mod
	for _, req := range f.Require {
		// Find any matching replace directive
		var matchingReplace *modfile.Replace
		for _, replace := range f.Replace {
			if replace.Old.Path == req.Mod.Path {
				matchingReplace = replace
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
				s.syncExistingRule(rule, req, matchingReplace)
				// No other changes needed
				continue
			}
		}

		// Add a new rule to the build file if one does not exist
		if err = s.addNewRule(file, req, matchingReplace); err != nil {
			return fmt.Errorf("failed to add new rule %v: %v", req.Mod.Path, err)
		}
	}

	return nil
}

func (s *syncer) syncExistingRule(rule *build.Rule, requireDirective *modfile.Require, replaceDirective *modfile.Replace) {
	reqVersion := requireDirective.Mod.Version
	// Add label for the replace directive
	if replaceDirective != nil {
		err := edit.AddLabel(rule, ReplaceLabel)
		if err != nil {
			log.Warningf("Failed to add replace label to %v: %v", requireDirective.Mod.Path, err)
		}
		// Update the requested version
		reqVersion = replaceDirective.New.Version
	}
	// Make sure the version is up-to-date
	rule.SetAttr("version", edit.NewStringExpr(reqVersion))
}

func (s *syncer) addNewRule(file *build.File, requireDirective *modfile.Require, replaceDirective *modfile.Replace) error {
	// List licences
	ls, err := s.licences.Get(requireDirective.Mod.Path, requireDirective.Mod.Version)
	if err != nil {
		return fmt.Errorf("failed to get licences for %v: %v", requireDirective.Mod.Path, err)
	}

	// If no replace directive, add a simple rule
	if replaceDirective == nil {
		file.Stmt = append(file.Stmt, edit.NewGoRepoRule(requireDirective.Mod.Path, requireDirective.Mod.Version, "", ls, []string{}))
		return nil
	}

	// If replace directive is just replacing the version, add a simple rule
	if replaceDirective.New.Path == requireDirective.Mod.Path {
		file.Stmt = append(file.Stmt, edit.NewGoRepoRule(requireDirective.Mod.Path, replaceDirective.New.Version, "", ls, []string{ReplaceLabel}))
		return nil
	}

	dl, dlName := edit.NewModDownloadRule(replaceDirective.New.Path, replaceDirective.New.Version, ls)
	file.Stmt = append(file.Stmt, dl)
	file.Stmt = append(file.Stmt, edit.NewGoRepoRule(requireDirective.Mod.Path, "", dlName, nil, []string{ReplaceLabel}))
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
