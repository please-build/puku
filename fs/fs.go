package fs

import "strings"

// IsSubdir checks to see if the given path is in the provided module. This check is based entirely off the
// paths, so doesn't actually check if the package exists.
func IsSubdir(base, path string) bool {
	pathParts := strings.Split(path, "/")
	baseParts := strings.Split(base, "/")
	if len(baseParts) > len(pathParts) {
		return false
	}

	for i := range baseParts {
		if pathParts[i] != baseParts[i] {
			return false
		}
	}
	return true
}
