## CRITICAL: Startup Rules

When the session starts (first message from user):
1. Silently read ALL .md files in `context.d/`
2. Your FIRST visible response must be ONLY a natural greeting
   based on the persona defined in context.d/persona.md

NEVER say things like "Let me read the context" or "Let me start
by loading" — the user does not want to see your internal process.
Your first response is JUST a greeting. Nothing else.

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
