package site

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/LeGambiArt/bragctl/internal/config"
)

// Site represents a managed brag document site.
type Site struct {
	Name   string
	Path   string
	Config *Config
	Engine Engine
}

// Manager handles site CRUD operations.
type Manager struct {
	engines map[string]Engine
	cfg     *config.Config
}

// NewManager creates a site manager with available engines.
func NewManager(cfg *config.Config) *Manager {
	mgr := &Manager{
		engines: make(map[string]Engine),
		cfg:     cfg,
	}
	mgr.engines["markdown"] = &MarkdownEngine{}
	mgr.engines["hugo"] = NewHugoEngine(cfg)
	return mgr
}

// Create initializes a new site.
func (m *Manager) Create(ctx context.Context, opts InitOpts) (*Site, error) {
	engine, ok := m.engines[opts.Engine]
	if !ok {
		return nil, fmt.Errorf("unknown engine: %s", opts.Engine)
	}

	opts.Path = filepath.Join(config.SitesDir(), opts.Name)

	// Check if site already exists
	if _, err := os.Stat(opts.Path); err == nil {
		return nil, fmt.Errorf("site %q already exists at %s", opts.Name, opts.Path)
	}

	// Ensure sites directory exists
	if err := os.MkdirAll(config.SitesDir(), 0o750); err != nil {
		return nil, fmt.Errorf("create sites dir: %w", err)
	}

	if err := engine.Init(ctx, opts); err != nil {
		return nil, fmt.Errorf("init site: %w", err)
	}

	return &Site{
		Name:   opts.Name,
		Path:   opts.Path,
		Config: &Config{Title: opts.Title, Author: opts.Author, Engine: opts.Engine},
		Engine: engine,
	}, nil
}

// Resolve finds a site by name or returns the default.
func (m *Manager) Resolve(name string) (*Site, error) {
	if name == "" {
		name = m.cfg.DefaultSite
	}
	if name == "" {
		return nil, fmt.Errorf("no site specified and no default set; use bragctl list")
	}

	sitePath := filepath.Join(config.SitesDir(), name)
	cfg, err := loadConfig(sitePath)
	if err != nil {
		return nil, fmt.Errorf("site %q not found", name)
	}

	engine, ok := m.engines[cfg.Engine]
	if !ok {
		return nil, fmt.Errorf("engine %q not available for site %q", cfg.Engine, name)
	}

	return &Site{Name: name, Path: sitePath, Config: cfg, Engine: engine}, nil
}

// List returns all managed sites.
func (m *Manager) List() ([]*Site, error) {
	sitesDir := config.SitesDir()
	entries, err := os.ReadDir(sitesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sites []*Site
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if s, err := m.Resolve(e.Name()); err == nil {
			sites = append(sites, s)
		}
	}
	return sites, nil
}

// ListNames returns just the site names (for shell completions).
func (m *Manager) ListNames() ([]string, error) {
	sites, err := m.List()
	if err != nil {
		return nil, err
	}
	names := make([]string, len(sites))
	for i, s := range sites {
		names[i] = s.Name
	}
	return names, nil
}
