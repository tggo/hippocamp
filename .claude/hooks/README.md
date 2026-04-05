# Hippocamp Hooks for Claude Code

These hooks integrate Hippocamp's knowledge graph with Claude Code.

## Setup

Copy these hooks to your project's `.claude/hooks/` directory and configure them in `.claude/settings.json`:

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "command": ".claude/hooks/hippocamp-pre-query.sh \"$PROMPT\""
      }
    ],
    "PostToolUse": [
      {
        "command": ".claude/hooks/hippocamp-post-edit.sh \"$TOOL_NAME\" \"$FILE_PATH\"",
        "matcher": "Edit|Write"
      }
    ]
  }
}
```

## Hook descriptions

### `hippocamp-pre-query.sh`

Runs before each prompt. Extracts keywords from the user's message, queries the Hippocamp graph, and outputs relevant knowledge as context for Claude.

**Requires:** `hippocamp` binary in PATH, persisted graph file.

### `hippocamp-post-edit.sh`

Runs after file edits. Tracks which files have been modified so the next `/project-analyze` run can refresh only stale entries. Writes to `.claude/.hippocamp-stale`.
