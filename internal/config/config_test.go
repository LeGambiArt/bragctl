package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSiteName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid names
		{name: "simple name", input: "my-site", wantErr: false},
		{name: "with underscore", input: "site_2", wantErr: false},
		{name: "with dot", input: "prod.v1", wantErr: false},
		{name: "alphanumeric only", input: "mysite123", wantErr: false},
		{name: "starts with number", input: "123site", wantErr: false},
		{name: "complex valid", input: "my-site_v2.prod", wantErr: false},

		// Invalid names
		{name: "empty string", input: "", wantErr: true},
		{name: "current dir", input: ".", wantErr: true},
		{name: "parent dir", input: "..", wantErr: true},
		{name: "path traversal up", input: "../etc", wantErr: true},
		{name: "path traversal deep", input: "../../tmp/evil", wantErr: true},
		{name: "forward slash", input: "foo/bar", wantErr: true},
		{name: "backslash", input: "foo\\bar", wantErr: true},
		{name: "null byte", input: "foo\x00bar", wantErr: true},
		{name: "absolute path", input: "/etc/passwd", wantErr: true},
		{name: "starts with dot", input: ".hidden", wantErr: true},
		{name: "starts with hyphen", input: "-badname", wantErr: true},
		{name: "contains space", input: "my site", wantErr: true},
		{name: "contains special char", input: "site@prod", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSiteName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSiteName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestSitePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Valid names should produce paths under SitesDir()
		{name: "valid simple", input: "mysite", wantErr: false},
		{name: "valid complex", input: "my-site_v2.prod", wantErr: false},

		// Invalid names should fail validation
		{name: "traversal attempt", input: "../etc", wantErr: true},
		{name: "absolute path", input: "/etc/passwd", wantErr: true},
		{name: "empty", input: "", wantErr: true},
		{name: "dot", input: ".", wantErr: true},
		{name: "dotdot", input: "..", wantErr: true},
	}

	sitesDir := SitesDir()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := SitePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SitePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}

			if err == nil {
				// Verify the path is under sitesDir
				if !strings.HasPrefix(path, sitesDir) {
					t.Errorf("SitePath(%q) = %q, does not start with sitesDir %q", tt.input, path, sitesDir)
				}

				// Verify the path contains the site name
				expectedPath := filepath.Join(sitesDir, tt.input)
				if path != expectedPath {
					t.Errorf("SitePath(%q) = %q, want %q", tt.input, path, expectedPath)
				}

				// Verify the path doesn't escape sitesDir
				relPath, err := filepath.Rel(sitesDir, path)
				if err != nil {
					t.Errorf("SitePath(%q) produced path that can't be made relative to sitesDir: %v", tt.input, err)
				}
				if strings.HasPrefix(relPath, ".."+string(filepath.Separator)) || relPath == ".." {
					t.Errorf("SitePath(%q) produced path that escapes sitesDir: %q", tt.input, relPath)
				}
			}
		})
	}
}
