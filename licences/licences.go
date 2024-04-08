package licences

import (
	"os"

	"github.com/google/go-licenses/licenses"
	"github.com/google/licenseclassifier/v2/assets"
	"github.com/please-build/buildtools/build"

	"github.com/please-build/puku/edit"
	"github.com/please-build/puku/graph"
	"github.com/please-build/puku/proxy"
)

var modCacheDir = "plz-out/puku/modcache"

type Licenses struct {
	graph *graph.Graph
	proxy *proxy.Proxy
}

func New(p *proxy.Proxy, g *graph.Graph) *Licenses {
	return &Licenses{
		graph: g,
		proxy: p,
	}
}

// getLicences returns a map of licences in the given directories
func getLicences(modPaths []string) (map[string][]string, error) {
	c, err := assets.DefaultClassifier()
	if err != nil {
		return nil, err
	}
	ret := make(map[string][]string)
	for _, modPath := range modPaths {
		paths, err := licenses.FindCandidates(modPath, modPath)
		if err != nil {
			return nil, err
		}

		var ls []string
		done := make(map[string]struct{})
		for _, path := range paths {
			bs, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			result := c.Match(bs)
			for _, m := range result.Matches {
				if m.MatchType != "License" {
					continue
				}
				if m.Confidence < 0.8 {
					continue
				}
				if _, ok := done[m.Name]; ok {
					continue
				}
				ls = append(ls, m.Name)
				done[m.Name] = struct{}{}
			}
		}
		ret[modPath] = ls
	}

	return ret, nil
}

func (l *Licenses) Update(paths []string) error {
	l.update(paths)
	return l.graph.FormatFiles()
}

func (l *Licenses) UpdateToStdout(format string, paths []string) error {
	l.update(paths)
	return l.graph.FormatFilesWithWriter(os.Stdout, format)
}

func (l *Licenses) update(paths []string) error {
	var mods []string
	rules := make(map[string]*build.Rule)

	for _, path := range paths {
		f, err := l.graph.LoadFile(path)
		if err != nil {
			return err
		}

		allRules := append(f.Rules("go_module"), append(f.Rules("go_mod_download"), f.Rules("go_repo")...)...)
		for _, r := range allRules {
			mod, ver := r.AttrString("module"), r.AttrString("version")
			// Only set the license on the rule that actually does the download
			if ver == "" {
				continue
			}

			// If the rule already has a license, skip it
			if len(r.AttrStrings("licences")) != 0 {
				continue
			}

			downloadPath, err := l.proxy.EnsureDownloaded(mod, ver, modCacheDir)
			if err != nil {
				if proxy.IsNotFound(err) {
					continue
				}
				return err
			}
			if downloadPath == "" {
				return nil
			}
			rules[downloadPath] = r
			mods = append(mods, downloadPath)
		}
	}

	licenseMap, err := getLicences(mods)
	if err != nil {
		return err
	}

	for mod, license := range licenseMap {
		if len(license) != 0 {
			rules[mod].SetAttr("licences", edit.NewStringList(license))
		}
	}
	return nil
}

func (l *Licenses) Get(mod, ver string) ([]string, error) {
	path, err := l.proxy.EnsureDownloaded(mod, ver, modCacheDir)
	if err != nil {
		if proxy.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if path == "" {
		return nil, nil
	}

	res, err := getLicences([]string{path})
	if err != nil {
		return nil, err
	}
	return res[path], nil
}
