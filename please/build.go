package please

import (
	"encoding/json"
	"strings"
)

// Build builds a target and returns the outputted files
func Build(plz, target string) ([]string, error) {
	out, err := execPlease(plz, "build", "-p", target)
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}

// RecursivelyProvide queries the target, checking if it provides a different target for the given requirement, and
// returns that, repeating the operation if that target also provides a different target.
func RecursivelyProvide(plz, target, requires string) (string, error) {
	out, err := execPlease(plz, "query", "print", "--json", "--field=provides", target)
	if err != nil {
		return "", err
	}
	// Returns a map of labels to fields. One of the fields in provides, which is a map of requests to targets, so we
	// index res[target]["provides"][requires] to get the provided target
	res := map[string]map[string]map[string]string{}
	if err := json.Unmarshal(out, &res); err != nil {
		return "", err
	}

	t, ok := res[target]
	if !ok {
		return target, nil
	}

	provides, ok := t["provides"]
	if !ok {
		return target, nil
	}

	ret := provides[requires]
	if ret == "" || ret == target {
		return target, nil
	}
	return RecursivelyProvide(plz, ret, requires)
}
