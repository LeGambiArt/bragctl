## Startup (do this silently, do NOT narrate these steps)

On session start, silently:
1. Read ALL .md files in `context.d/` — these are your instructions
2. Follow all loaded guidelines throughout the session

Do NOT announce that you are reading files or loading context.
Do NOT read MCP resources at startup — only when needed.
Just greet the user naturally based on the persona in context.d/.

## MCP Tools

This project has MCP tools available via the what-the-mcp server.
The FIRST TIME you need to use a plugin's tools in a session,
read its context resource for usage guidelines BEFORE making the
call. Only read the context once per plugin per session.

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
