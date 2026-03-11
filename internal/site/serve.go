package site

import (
	"bytes"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
)

// htmlSanitizer allows standard user-generated content HTML while preventing XSS.
// It permits safe elements (bold, italic, links, images, code blocks, tables, lists)
// but strips scripts, style tags, event handlers, and other dangerous content.
var htmlSanitizer = bluemonday.UGCPolicy()

// markdownServer serves a bragctl markdown site with rendered HTML.
type markdownServer struct {
	sitePath string
	postsDir string
	md       goldmark.Markdown
}

func newMarkdownServer(sitePath string) *markdownServer {
	return &markdownServer{
		sitePath: sitePath,
		postsDir: filepath.Join(sitePath, "posts"),
		md:       goldmark.New(),
	}
}

func (s *markdownServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	if path == "" || path == "/" {
		s.serveIndex(w, r)
		return
	}

	// Only serve .md files from posts/
	if !strings.HasSuffix(path, ".md") {
		http.NotFound(w, r)
		return
	}

	s.servePost(w, r, path)
}

func (s *markdownServer) serveIndex(w http.ResponseWriter, _ *http.Request) {
	entries, err := os.ReadDir(s.postsDir)
	if err != nil {
		http.Error(w, "posts directory not found", http.StatusInternalServerError)
		return
	}

	type post struct {
		Name string
		Path string
	}

	var posts []post
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		posts = append(posts, post{
			Name: strings.TrimSuffix(e.Name(), ".md"),
			Path: e.Name(),
		})
	}

	// Newest first (date-prefixed filenames sort correctly)
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Name > posts[j].Name
	})

	cfg, _ := loadConfig(s.sitePath)
	title := "Brag Document"
	if cfg != nil && cfg.Title != "" {
		title = cfg.Title
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := indexTmpl.Execute(w, map[string]any{
		"Title": title,
		"Posts": posts,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *markdownServer) servePost(w http.ResponseWriter, _ *http.Request, name string) {
	data, err := os.ReadFile(filepath.Join(s.postsDir, filepath.Base(name))) //nolint:gosec // user-navigated file
	if err != nil {
		http.NotFound(w, nil)
		return
	}

	// Strip YAML frontmatter
	content := stripFrontmatter(data)

	var buf bytes.Buffer
	if err := s.md.Convert(content, &buf); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := postTmpl.Execute(w, map[string]template.HTML{
		"Title":   template.HTML(template.HTMLEscapeString(strings.TrimSuffix(filepath.Base(name), ".md"))), //nolint:gosec // explicitly escaped
		"Content": template.HTML(htmlSanitizer.SanitizeBytes(buf.Bytes())),                                  //nolint:gosec // sanitized with bluemonday
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// stripFrontmatter removes YAML frontmatter (--- delimited) from markdown.
func stripFrontmatter(data []byte) []byte {
	s := string(data)
	if !strings.HasPrefix(s, "---") {
		return data
	}
	// Find closing ---
	end := strings.Index(s[3:], "\n---")
	if end < 0 {
		return data
	}
	return []byte(s[end+3+4:]) // skip past closing --- and newline
}

var indexTmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html><head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; max-width: 800px; margin: 0 auto; padding: 2rem; background: #1a1a2e; color: #e0e0e0; }
  h1 { color: #e0e0e0; border-bottom: 2px solid #333; padding-bottom: 0.5rem; }
  ul { list-style: none; padding: 0; }
  li { padding: 0.5rem 0; border-bottom: 1px solid #2a2a3e; }
  a { color: #82aaff; text-decoration: none; font-size: 1.1rem; }
  a:hover { text-decoration: underline; color: #b4d0ff; }
  .empty { color: #888; font-style: italic; }
</style>
</head><body>
<h1>{{.Title}}</h1>
{{if .Posts}}
<ul>
{{range .Posts}}<li><a href="/{{.Path}}">{{.Name}}</a></li>
{{end}}
</ul>
{{else}}
<p class="empty">No posts yet. Use your AI assistant to create your first brag entry.</p>
{{end}}
</body></html>`))

var postTmpl = template.Must(template.New("post").Parse(`<!DOCTYPE html>
<html><head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; max-width: 800px; margin: 0 auto; padding: 2rem; background: #1a1a2e; color: #e0e0e0; }
  a { color: #82aaff; }
  a:hover { color: #b4d0ff; }
  .nav { margin-bottom: 2rem; }
  h1, h2, h3 { color: #e0e0e0; }
  code { background: #2a2a3e; padding: 0.2em 0.4em; border-radius: 3px; }
  pre { background: #2a2a3e; padding: 1rem; border-radius: 6px; overflow-x: auto; }
  pre code { background: none; padding: 0; }
  blockquote { border-left: 3px solid #82aaff; margin-left: 0; padding-left: 1rem; color: #aaa; }
  table { border-collapse: collapse; width: 100%; }
  th, td { border: 1px solid #333; padding: 0.5rem; text-align: left; }
  th { background: #2a2a3e; }
</style>
</head><body>
<div class="nav"><a href="/">&larr; Back to posts</a></div>
<h1>{{.Title}}</h1>
{{.Content}}
</body></html>`))
