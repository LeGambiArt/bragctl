package site

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testMarkdownSite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create site config
	cfg := &Config{Title: "Test Brags", Author: "Alice", Engine: "markdown"}
	if err := writeConfig(dir, cfg); err != nil {
		t.Fatal(err)
	}

	// Create posts
	postsDir := filepath.Join(dir, "posts")
	if err := os.MkdirAll(postsDir, 0o750); err != nil {
		t.Fatal(err)
	}

	post1 := `---
title: "First Post"
date: 2026-03-01
---

# My First Brag

I did **something great** today.
`
	post2 := `---
title: "Second Post"
date: 2026-03-08
---

# Week Two

- Fixed a critical bug
- Shipped a feature
`
	if err := os.WriteFile(filepath.Join(postsDir, "2026-03-01-first.md"), []byte(post1), 0o644); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(postsDir, "2026-03-08-second.md"), []byte(post2), 0o644); err != nil { //nolint:gosec // test
		t.Fatal(err)
	}

	return dir
}

func TestMarkdownServerIndex(t *testing.T) {
	dir := testMarkdownSite(t)
	srv := newMarkdownServer(dir)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/") //nolint:gosec // test server
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET / status = %d, want 200", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(bodyBytes)

	// Should list both posts
	if !strings.Contains(body, "2026-03-01-first") {
		t.Error("index missing first post")
	}
	if !strings.Contains(body, "2026-03-08-second") {
		t.Error("index missing second post")
	}

	// Should have site title
	if !strings.Contains(body, "Test Brags") {
		t.Error("index missing site title")
	}

	// Second post should appear before first (newest first)
	idx1 := strings.Index(body, "2026-03-08-second")
	idx2 := strings.Index(body, "2026-03-01-first")
	if idx1 > idx2 {
		t.Error("posts not sorted newest first")
	}
}

func TestMarkdownServerPost(t *testing.T) {
	dir := testMarkdownSite(t)
	srv := newMarkdownServer(dir)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/2026-03-01-first.md") //nolint:gosec // test server
	if err != nil {
		t.Fatalf("GET post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET post status = %d, want 200", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	body := string(bodyBytes)

	// Should render markdown to HTML
	if !strings.Contains(body, "<strong>something great</strong>") {
		t.Error("markdown not rendered: missing <strong>")
	}

	// Should have back link
	if !strings.Contains(body, "Back to posts") {
		t.Error("post missing back link")
	}

	// Should NOT contain frontmatter
	if strings.Contains(body, "title:") {
		t.Error("post contains raw frontmatter")
	}
}

func TestMarkdownServerNotFound(t *testing.T) {
	dir := testMarkdownSite(t)
	srv := newMarkdownServer(dir)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	// Non-.md path
	resp, err := http.Get(ts.URL + "/something.txt") //nolint:gosec // test server
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("non-.md path status = %d, want 404", resp.StatusCode)
	}

	// Non-existent .md
	resp, err = http.Get(ts.URL + "/nope.md") //nolint:gosec // test server
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("missing .md status = %d, want 404", resp.StatusCode)
	}
}

func TestMarkdownServerEmptySite(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "posts"), 0o750); err != nil {
		t.Fatal(err)
	}

	srv := newMarkdownServer(dir)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/") //nolint:gosec // test server
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(bodyBytes), "No posts yet") {
		t.Error("empty site should show 'No posts yet' message")
	}
}

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "with frontmatter",
			input: "---\ntitle: test\ndate: 2026-01-01\n---\n# Hello",
			want:  "\n# Hello",
		},
		{
			name:  "without frontmatter",
			input: "# Hello\nWorld",
			want:  "# Hello\nWorld",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(stripFrontmatter([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("stripFrontmatter() = %q, want %q", got, tt.want)
			}
		})
	}
}
