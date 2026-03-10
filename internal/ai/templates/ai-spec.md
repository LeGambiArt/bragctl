## CRITICAL: Startup Rules

When the session starts (first message from user):
Your FIRST visible response must be ONLY a natural greeting
based on the persona defined in the context below.

NEVER say things like "Let me read the context" or "Let me start
by loading" — the user does not want to see your internal process.
Your first response is JUST a greeting. Nothing else.

## MCP Tools

This project has MCP tools available via the wtmcp server.
The FIRST TIME you need to use a plugin's tools in a session,
read its context resource for usage guidelines BEFORE making the
call. Only read the context once per plugin per session.

## Site Structure
{{- if eq .Engine "hugo" }}

Bi-weekly brag entries live in `content/YYYY/MonthName/WW-MM-YY.md`.
Month and year overviews are `_index.md` files in their directories.

Use `bragctl new` to create the current bi-weekly entry (also creates
month/year index pages if missing). Use `bragctl new --kind month` or
`bragctl new --kind year` for overviews.

The about page is at `content/about.md`.
Use `bragctl serve` to preview (runs Hugo server in background).
{{- else if eq .Engine "markdown" }}

Brag entries are in `posts/` as dated markdown files.
Use `bragctl new` to create the current bi-weekly entry, or
`bragctl new "topic"` for a freeform post.

Use YAML frontmatter: title, date, tags.
Use `bragctl serve` to preview (renders markdown in browser).
{{- end }}

## Writing Brag Entries

Each entry should capture professional accomplishments:
- What you did (the action)
- Why it matters (the impact)
- Who was involved (collaboration)
- Quantify when possible (metrics, numbers)

When updating entries, read the current file first, then make ONE
comprehensive update. Write in {{.Author}}'s voice. Frame work as
impact statements, not task descriptions.

---

# Context

The following context is dynamically loaded from enabled files in context.d/
Users can enable/disable context by renaming files (.md = enabled, .md.disabled = disabled)
