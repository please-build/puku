package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/please-build/puku/kinds"
)

func TestGetKind(t *testing.T) {
	c := Config{
		LibKinds: map[string]*KindConfig{
			"go_binary": {},
		},
		TestKinds: map[string]*KindConfig{
			"service_acceptance_test": {},
		},
		ExcludeBuiltinKinds: []string{"proto_library"},
	}
	t.Run("default kind", func(t *testing.T) {
		kind := c.GetKind("go_library")
		require.NotNil(t, kind)
		assert.Equal(t, kinds.Lib, kind.Type)
	})

	t.Run("overridden default kind", func(t *testing.T) {
		kind := c.GetKind("go_binary")
		require.NotNil(t, kind)
		assert.Equal(t, kinds.Lib, kind.Type)
	})

	t.Run("excluded default kind", func(t *testing.T) {
		kind := c.GetKind("proto_library")
		assert.Nil(t, kind)
	})

	t.Run("custom kinds", func(t *testing.T) {
		kind := c.GetKind("service_acceptance_test")
		require.NotNil(t, kind)
		assert.Equal(t, kinds.Test, kind.Type)
	})
}

func TestGetStop(t *testing.T) {
	t.Run("when stop is true then stop", func(t *testing.T) {
		c := Config{Stop: true}
		assert.True(t, c.GetStop())
	})

	t.Run("when stop is false then don't stop", func(t *testing.T) {
		c := Config{Stop: false}
		assert.False(t, c.GetStop())
	})

	t.Run("when base stop is true then stop", func(t *testing.T) {
		base := Config{Stop: true}
		c := Config{base: &base, Stop: false}
		assert.True(t, c.GetStop())

		c = Config{base: &base, Stop: true}
		assert.True(t, c.GetStop())
	})
}
