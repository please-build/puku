package harness

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/bazelbuild/buildtools/build"
	exec "golang.org/x/sys/execabs"

	"github.com/please-build/puku/config"
)

type TestHarness struct {
	Puku     string
	Please   string
	RepoRoot string
}

func MustNew() *TestHarness {
	readEnvAbs := func(envVar string) string {
		v, ok := os.LookupEnv(envVar)
		if !ok {
			log.Fatalf("$%v not set", envVar)
		}
		v, err := filepath.Abs(v)
		if err != nil {
			log.Fatalf("%v", err)
		}
		return v
	}

	puku := readEnvAbs("DATA_PUKU")
	please := readEnvAbs("DATA_PLEASE")
	repo := readEnvAbs("DATA_REPO")

	conf := config.Config{
		PleasePath: please,
	}
	bs, err := json.Marshal(conf)
	if err != nil {
		log.Fatalf("failed to write puku.json: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repo, "puku.json"), bs, 0644); err != nil {
		log.Fatalf("failed to write puku.json: %v", err)
	}

	return &TestHarness{
		Puku:     puku,
		Please:   please,
		RepoRoot: repo,
	}
}

func (h *TestHarness) Format(paths ...string) error {
	cmd := exec.Command(h.Puku, append([]string{"fmt"}, paths...)...)
	cmd.Dir = h.RepoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run puku fmt %v: %v:\n%v", paths, err, string(out))
	}
	return nil
}

func (h *TestHarness) ParseFile(path string) (*build.File, error) {
	absPath := filepath.Join(h.RepoRoot, path)
	file, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	return build.Parse(path, file)
}
