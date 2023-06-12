package generate

import (
	"errors"
	"github.com/please-build/puku/proxy"
)

type FakeProxy struct {
	modules map[string]string
}

func (f FakeProxy) ResolveModuleForPackage(pattern string) (*proxy.Module, error) {
	if f.modules == nil {
		return nil, errors.New("not found")
	}

	if mod, ok := f.modules[pattern]; ok {
		// we don't really care about the version here
		return &proxy.Module{Module: mod, Version: "v1.0.0"}, nil
	}
	return nil, errors.New("not found")
}

func (f FakeProxy) ResolveDeps(mods, newMods []*proxy.Module) ([]*proxy.Module, error) {
	panic("not implemented")
}



