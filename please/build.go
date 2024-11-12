package please

import (
	"encoding/json"
	"strings"
)

// Build builds a target and returns the outputted files
func Build(plz string, targets ...string) ([]string, error) {
	out, err := execPlease(plz, append([]string{"build", "-p"}, targets...)...)
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}

// RecursivelyProvide queries the target, checking if it provides a different target for the given requirement, and
// returns that, repeating the operation if that target also provides a different target.
func RecursivelyProvide(plz string, target string, requires string) ([]string, error) {
	out, err := execPlease(plz, "query", "print", "--json", "--field=provides", target)
	if err != nil {
		return nil, err
	}
	// Returns a map of labels to fields. One of the fields in provides, which is a map of requests to targets, so we
	// index res[target]["provides"][requires] to get the provided target
	res := map[string]map[string]map[string][]string{}
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, err
	}

	t, ok := res[target]
	if !ok {
		return []string{target}, nil
	}

	provides, ok := t["provides"]
	if !ok {
		return []string{target}, nil
	}

	providedTargets := provides[requires]

	if len(providedTargets) == 0 || (len(providedTargets) == 1 && providedTargets[0] == target) {
		return []string{target}, nil
	}

	ret := make([]string, 0, len(providedTargets))
	for _, providedTarget := range providedTargets {
		if providedTarget == target {
			ret = append(ret, providedTarget) // Providing itself, don't recurse
		} else {
			r, err := RecursivelyProvide(plz, providedTarget, requires)
			if err != nil {
				return []string{}, err
			}
			ret = append(ret, r...)
		}
	}

	return ret, nil
}
