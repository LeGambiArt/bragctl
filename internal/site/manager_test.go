package site

import (
	"context"
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

func TestCreateRejectsPathTraversal(t *testing.T) {
	mgr := NewManager(&config.Config{})

	tests := []struct {
		name     string
		siteName string
	}{
		{name: "parent dir traversal", siteName: "../evil"},
		{name: "deep traversal", siteName: "../../etc/passwd"},
		{name: "absolute path", siteName: "/etc/passwd"},
		{name: "with slash", siteName: "foo/bar"},
		{name: "with backslash", siteName: "foo\\bar"},
		{name: "dot", siteName: "."},
		{name: "dotdot", siteName: ".."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mgr.Create(context.Background(), InitOpts{
				Name:   tt.siteName,
				Title:  "Test",
				Author: "Test",
				Engine: "markdown",
			})
			if err == nil {
				t.Errorf("Create(%q) succeeded, expected error for path traversal attempt", tt.siteName)
			}
		})
	}
}

func TestResolveRejectsPathTraversal(t *testing.T) {
	mgr := NewManager(&config.Config{})

	tests := []struct {
		name     string
		siteName string
	}{
		{name: "parent dir traversal", siteName: "../evil"},
		{name: "deep traversal", siteName: "../../etc/passwd"},
		{name: "absolute path", siteName: "/etc/passwd"},
		{name: "with slash", siteName: "foo/bar"},
		{name: "dot", siteName: "."},
		{name: "dotdot", siteName: ".."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mgr.Resolve(tt.siteName)
			if err == nil {
				t.Errorf("Resolve(%q) succeeded, expected error for path traversal attempt", tt.siteName)
			}
		})
	}
}
