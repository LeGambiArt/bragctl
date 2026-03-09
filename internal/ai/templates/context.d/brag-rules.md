# Brag Document Rules

## Core Principles

1. Gather ALL context before updating — read the current entry first
2. Make ONE comprehensive edit per update, not multiple small ones
3. Read previous entries for continuity and voice consistency
4. Map work activities to the right sections
5. Always write in {{.Author}}'s voice, not the AI's

## Section Mapping

| Activity | Section |
|----------|---------|
| Code, architecture, debugging | Technical Achievements |
| Process improvement, tooling, automation | Invisible Work |
| Mentoring, reviews, team support | Collaboration & Leadership |
| Training, conferences, certifications | Learning & Growth |
| Cross-team coordination, stakeholder management | Cross-Functional Impact |

## Achievement Framing

Write accomplishments as impact statements, not task descriptions.

**Before** (task): "Fixed authentication bug"
**After** (impact): "Resolved authentication timeout affecting mobile
users by updating session config — 95% reduction in timeout errors"

Guidelines:
- Lead with the outcome, not the action
- Quantify impact when possible (%, time saved, users affected)
- Use collaborative language ("Led", "Partnered", "Coordinated")
- Focus on the problem solved, not just the code changed
- Include business context (why it matters)

## Meeting Extraction

When {{.Author}} mentions meetings, extract brag-worthy content:
- What was their role? (presenter, facilitator, contributor)
- Did they help solve problems or make decisions?
- Was knowledge shared or coordination achieved?
- Map to the appropriate section above

## Link Format

Use Jira links: `[KEY-123](https://jira-url/browse/KEY-123)`

## Commit Format

When committing brag document updates:
- Type: `docs` for content, `feat` for new entries
- Subject: brief description of what was documented
- Keep commit messages concise (50 char subject)
