package knownimports

import (
	_ "embed"
	"go/build"
	"os"
	"path/filepath"
	"strings"
)

//go:embed go_root_packages
var goRootPkgs string

// Optimisation to avoid going to disk, but also includes packages like unsafe and C that don't actually appear in
// the SDK. We still check in case a new Go version includes a new package.
var goRootImports = map[string]struct{}{
	"unsafe": {},
	"C":      {},
}

func init() {
	for _, pkg := range strings.Split(goRootPkgs, "\n") {
		pkg := strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}
		goRootImports[pkg] = struct{}{}
	}
}

func IsInGoRoot(i string) bool {
	if strings.HasPrefix(i, "crypto/") {
		return true
	}
	if _, ok := goRootImports[i]; ok {
		return true
	}

	// Attempt to look this up in the GOROOT incase the users is on a newer version of Go to us
	b := build.Default
	path := filepath.Join(b.GOROOT, "pkg", b.GOOS+"_"+b.GOARCH, i+".a")
	if _, err := os.Lstat(path); err == nil {
		goRootImports[i] = struct{}{}
		return true
	}
	return false
}
