// Package site provides the site engine abstraction and implementations
// for managing brag document sites.
package site

import (
	"context"
)

// Engine defines how a site type is initialized, served, and built.
type Engine interface {
	// Name returns the engine identifier ("hugo", "markdown").
	Name() string

	// Init creates a new site at the given path.
	Init(ctx context.Context, opts InitOpts) error

	// Validate checks that a site directory looks correct.
	Validate(sitePath string) error

	// Serve starts a dev server for previewing the site.
	Serve(ctx context.Context, sitePath string, opts ServeOpts) error
}

// ServeOpts holds parameters for running a dev server.
type ServeOpts struct {
	Port       int
	Bind       string // address to bind to (default: 127.0.0.1)
	Foreground bool   // run in foreground (default: background)
}

// InitOpts holds parameters for creating a new site.
type InitOpts struct {
	Name   string
	Path   string
	Title  string
	Author string
	Engine string
	AI     string // preferred assistant (claude, cursor, gemini)
}

// Config is the per-site configuration stored in _config.yaml.
type Config struct {
	Title       string `yaml:"title"`
	Author      string `yaml:"author"`
	Description string `yaml:"description,omitempty"`
	Engine      string `yaml:"engine"`
	AI          string `yaml:"ai,omitempty"` // preferred assistant (claude, cursor, gemini)
}
