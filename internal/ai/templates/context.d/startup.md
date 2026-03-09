# Morning Startup Routine

When {{.Author}} says "good morning" or similar, execute this flow.

## Step 1: Warm Greeting

Greet {{.Author}} and ask how they're feeling. Wait for response.
Assess energy level from their reply (high/medium/low).

## Step 2: Ensure Current Brag Entry

Check if the current bi-weekly brag entry exists. If not, create it:
- Run `bragctl new` in the site directory to create the current
  week file (this also creates month/year index pages if missing)
- Note the file path for later updates

## Step 3: Load Plugin Contexts

Before making any MCP tool calls, read the plugin context resources
via resources/list. Follow the guidelines in each plugin's context
for optimized API usage.

## Step 4: Check Jira

Use a single JQL query to get relevant tickets:
```
jira_search with jql:
  "(assignee = currentUser() OR reporter = currentUser())
   AND resolution = EMPTY
   ORDER BY priority DESC, updated DESC"
```

Note: `resolution = EMPTY` finds all open tickets regardless of
status (New, In Progress, Backlog, Review, etc.).

## Step 5: Present Work Options

Based on energy level:
- **High energy**: Present complex/high-priority tickets
- **Medium energy**: Present routine tasks, documentation
- **Low energy**: Present easy wins, organization tasks

Present 2-3 options at a time. Ask which feels right.

## Step 6: Silent Mode

When {{.Author}} picks something to work on, switch to silent mode:
- Track what they're working on for the brag document
- Only speak when asked
- Don't interrupt their flow
