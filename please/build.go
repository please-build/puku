package please

import "strings"

// Build builds a target and returns the outputted files
func Build(plz string, target string) ([]string, error) {
	out, err := execPlease(plz, "build", "-p", target)
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}
