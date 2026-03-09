## MCP Tools

This project has MCP tools available via the what-the-mcp server.

Before using any MCP plugin for the first time in a session:
1. Use resources/list to discover available plugin contexts
2. Read the plugin's context resource for usage guidelines
3. Follow those guidelines for all calls to that plugin

## Additional Context

Read all files in `context.d/` for workflow preferences, team
conventions, and custom instructions specific to this site.

## Site Structure
{{- if eq .Engine "hugo" }}

Posts are in `content/posts/`.
Use Hugo frontmatter format.
Run `hugo server -D` to preview.
{{- else if eq .Engine "markdown" }}

Posts are in `posts/`.
Use YAML frontmatter: title, date, tags, impact.
Files are plain markdown — no build step needed.
{{- end }}

## Writing Brag Entries

Each post should capture a professional accomplishment:
- What you did (the action)
- Why it matters (the impact)
- Who was involved (collaboration)
- Quantify when possible (metrics, numbers)
