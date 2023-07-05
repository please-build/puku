package please

import (
	"bytes"
	"fmt"
	"os/exec"
)

func execPlease(plz string, args ...string) ([]byte, error) {
	cmd := exec.Command(plz, args...)
	stdErr := new(bytes.Buffer)
	cmd.Stderr = stdErr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%v\n%v", err, stdErr.String())
	}
	return out, nil
}
