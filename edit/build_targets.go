package edit

import (
	"path/filepath"
	"strings"
)

func SubrepoTarget(module, thirdPartyFolder, packageName string) string {
	subrepoName := SubrepoName(module, thirdPartyFolder)

	name := filepath.Base(packageName)
	if packageName == "" {
		name = filepath.Base(module)
	}

	return BuildTarget(name, packageName, subrepoName)
}

func SubrepoName(module, thirdPartyFolder string) string {
	return filepath.Join(thirdPartyFolder, strings.ReplaceAll(module, "/", "_"))
}

func BuildTarget(name, pkgDir, subrepo string) string {
	bs := new(strings.Builder)
	if subrepo != "" {
		bs.WriteString("///")
		bs.WriteString(subrepo)
	}

	if pkgDir != "" || subrepo != "" {
		bs.WriteString("//")
	}

	if pkgDir == "." {
		pkgDir = ""
	}

	if pkgDir != "" {
		bs.WriteString(pkgDir)
		if filepath.Base(pkgDir) != name {
			bs.WriteString(":")
			bs.WriteString(name)
		}
	} else {
		bs.WriteString(":")
		bs.WriteString(name)
	}
	return bs.String()
}
