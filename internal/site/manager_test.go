package site

import (
	"testing"

	"github.com/LeGambiArt/bragctl/internal/config"
)

func TestNewManagerRegistersEngines(t *testing.T) {
	mgr := NewManager(&config.Config{})

	for _, name := range []string{"markdown", "hugo"} {
		if _, ok := mgr.engines[name]; !ok {
			t.Errorf("engine %q not registered", name)
		}
	}
}
