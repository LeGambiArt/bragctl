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
}

// InitOpts holds parameters for creating a new site.
type InitOpts struct {
	Name   string
	Path   string
	Title  string
	Author string
	Engine string
}

// Config is the per-site configuration stored in _config.yaml.
type Config struct {
	Title       string `yaml:"title"`
	Author      string `yaml:"author"`
	Description string `yaml:"description"`
	Engine      string `yaml:"engine"`
}
