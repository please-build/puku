package proxy

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

var DefaultURL = "https://proxy.golang.org"

var client = http.DefaultClient

type ModuleNotFound struct {
	Path string
}

func (err ModuleNotFound) Error() string {
	return fmt.Sprintf("can't find module %v", err.Path)
}

// Module is the module, and it's version returned from @latest
type Module struct {
	Module  string
	Version string
}

type Proxy struct {
	latestVer map[string]Module
	modFiles  map[Module]*modfile.File
	url       string
}

func New(url string) *Proxy {
	return &Proxy{
		latestVer: map[string]Module{},
		modFiles:  map[Module]*modfile.File{},
		url:       url,
	}
}

// GetLatestVersion returns the latest version for a module from the proxy. Will return an error of type ModuleNotFound
// if no module exists for the given path
func (proxy *Proxy) GetLatestVersion(modulePath string) (Module, error) {
	if result, ok := proxy.latestVer[modulePath]; ok {
		if result.Module != "" {
			return result, nil
		}
		return Module{}, ModuleNotFound{Path: modulePath}
	}

	resp, err := client.Get(fmt.Sprintf("%s/%s/@latest", proxy.url, strings.ToLower(modulePath)))
	if err != nil {
		return Module{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 || resp.StatusCode == 410 {
			proxy.latestVer[modulePath] = Module{}
			return Module{}, ModuleNotFound{Path: modulePath}
		}
		return Module{}, fmt.Errorf("unexpected status code getting module %v: %v", modulePath, resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return Module{}, err
	}

	version := struct {
		Version string
	}{}
	if err := json.Unmarshal(b, &version); err != nil {
		return Module{}, err
	}

	proxy.latestVer[modulePath] = Module{
		Module:  modulePath,
		Version: version.Version,
	}
	return proxy.latestVer[modulePath], nil
}

// ResolveModuleForPackage tries to determine the module name for a given package pattern
func (proxy *Proxy) ResolveModuleForPackage(pattern string) (*Module, error) {
	modulePath := strings.TrimSuffix(pattern, "/...")

	var paths []string
	for modulePath != "." {
		paths = append(paths, modulePath)
		// Try and get the latest version to determine if we've found the module part yet
		latest, err := proxy.GetLatestVersion(modulePath)
		if err == nil {
			for _, p := range paths {
				proxy.latestVer[p] = latest
			}
			return &latest, nil
		}
		if _, ok := err.(ModuleNotFound); !ok {
			return nil, err
		}

		modulePath = filepath.Dir(modulePath)
	}
	return nil, ModuleNotFound{Path: modulePath}
}

// getGoModWithFallback attempts to get a go.mod for the given module and
// version with fallback for supporting modules with case insensitivity.
func (proxy *Proxy) getGoModWithFallback(mod, version string) (*modfile.File, error) {
	modVersionsToAttempt := map[string]string{
		mod: version,
	}

	// attempt lowercasing entire mod string for packages like:
	// - `github.com/Sirupsen/logrus` -> `github.com/sirupsen/logrus`.
	// https://github.com/sirupsen/logrus/issues/543
	modVersionsToAttempt[strings.ToLower(mod)] = version

	var errs error
	for mod, version := range modVersionsToAttempt {
		modFile, err := proxy.getGoMod(mod, version)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		return modFile, nil
	}

	return nil, errs
}

// ResolveDeps will resolve the dependencies of a module list following the minimum viable version strategy
func (proxy *Proxy) ResolveDeps(mods, newMods []*Module) ([]*Module, error) {
	deps := map[string]string{}

	// Add all the mods as requirements
	for _, mod := range append(mods, newMods...) {
		deps[mod.Module] = mod.Version
	}

	// And then walk the requirements of the new modules updating the deps as we see higher version requirements
	for _, mod := range newMods {
		err := proxy.getDeps(deps, mod.Module, mod.Version)
		if err != nil {
			return nil, err
		}
	}

	// Then return a list of all resolved modules
	ret := make([]*Module, 0, len(deps))
	for mod, ver := range deps {
		ret = append(ret, &Module{Module: mod, Version: ver})
	}
	return ret, nil
}

func (proxy *Proxy) getDeps(deps map[string]string, mod, version string) error {
	modFile, err := proxy.getGoModWithFallback(mod, version)
	if err != nil {
		return err
	}

	for _, req := range modFile.Require {
		oldVer, ok := deps[req.Mod.Path]
		if !ok || semver.Compare(oldVer, req.Mod.Version) < 0 {
			deps[req.Mod.Path] = req.Mod.Version
			if err := proxy.getDeps(deps, req.Mod.Path, req.Mod.Version); err != nil {
				return err
			}
		}
	}

	return nil
}

func (proxy *Proxy) getGoMod(mod, ver string) (*modfile.File, error) {
	modVer := Module{mod, ver}
	if modFile, ok := proxy.modFiles[modVer]; ok {
		return modFile, nil
	}

	file := fmt.Sprintf("%s/%s/@v/%s.mod", proxy.url, mod, ver)
	resp, err := client.Get(file)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%v %v: \n%v", file, resp.StatusCode, string(body))
	}

	modFile, err := modfile.Parse(file, body, nil)
	if err != nil {
		return nil, err
	}

	proxy.modFiles[modVer] = modFile
	return modFile, nil
}

func (proxy *Proxy) EnsureDownloaded(mod, ver, dir string) (string, error) {
	modRoot := filepath.Join(dir, fmt.Sprintf("%v@%v", mod, ver))
	if _, err := os.Lstat(modRoot); err == nil {
		return modRoot, nil // seems to already exist
	}

	url := fmt.Sprintf("%v/%v/@v/%v.zip", proxy.url, mod, ver)
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ModuleNotFound{mod}
	}

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(bs), int64(len(bs)))
	if err != nil {
		return "", err
	}

	// Read all the files from zip archive
	for _, zipFile := range zipReader.File {
		path := filepath.Join(dir, zipFile.Name)
		if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
			return "", err
		}
		dest, err := os.Create(path)
		if err != nil {
			return "", err
		}
		src, err := zipFile.Open()
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(dest, src); err != nil {
			return "", err
		}
	}
	return modRoot, nil
}

// IsNotFound returns true if the error is ModuleNotFound
func IsNotFound(err error) bool {
	_, ok := err.(ModuleNotFound)
	return ok
}
