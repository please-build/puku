package glob

import (
	"os"
	"path/filepath"
)

type pattern struct {
	dir, glob string
}

type Globber struct {
	cache map[pattern][]string
}

type GlobArgs struct {
	Include, Exclude []string
}

func New() *Globber {
	return &Globber{cache: map[pattern][]string{}}
}

// Glob is a specialised version of the glob builtin from Please. It assumes:
// 1) globs should only match .go files as they're being used in go rules
// 2) go rules will never depend on files outside the package dir, so we don't need to support **
// 3) we don't want symlinks, directories and other non-regular files
func (g *Globber) Glob(dir string, args *GlobArgs) ([]string, error) {
	inc := map[string]struct{}{}
	for _, i := range args.Include {
		fs, err := g.glob(dir, i)
		if err != nil {
			return nil, err
		}

		for _, f := range fs {
			inc[f] = struct{}{}
		}
	}

	for _, i := range args.Exclude {
		fs, err := g.glob(dir, i)
		if err != nil {
			return nil, err
		}

		for _, f := range fs {
			delete(inc, f)
		}
	}

	ret := make([]string, 0, len(inc))
	for i := range inc {
		ret = append(ret, i)
	}
	return ret, nil
}

// glob matches all regular files in a directory based on a glob pattern
func (g *Globber) glob(dir, glob string) ([]string, error) {
	p := pattern{dir: dir, glob: glob}
	if res, ok := g.cache[p]; ok {
		return res, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		// Ignore dirs, symlinks etc.
		if !e.Type().IsRegular() {
			continue
		}

		// We're globbing for Go files to determine their imports. We can skip any other files.
		if filepath.Ext(e.Name()) != ".go" {
			continue
		}

		match, err := filepath.Match(glob, e.Name())
		if err != nil {
			return nil, err
		}

		if match {
			files = append(files, e.Name())
		}
	}

	g.cache[p] = files
	return files, nil
}
