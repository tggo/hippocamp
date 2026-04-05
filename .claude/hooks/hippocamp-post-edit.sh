#!/bin/bash
# Hippocamp post-edit hook for Claude Code.
# Runs after Edit/Write tool use: marks the graph as needing a re-analyze
# for the modified file. Does NOT do full re-analysis (that's expensive) —
# instead it adds a hippo:status "stale" marker so the next /project-analyze
# knows which files to refresh.
#
# Install in .claude/settings.json:
#   "hooks": {
#     "PostToolUse": [{
#       "command": ".claude/hooks/hippocamp-post-edit.sh \"$TOOL_NAME\" \"$FILE_PATH\"",
#       "matcher": "Edit|Write"
#     }]
#   }
#
# Requires: hippocamp MCP server running (this uses the triple tool via the server)

set -euo pipefail

TOOL_NAME="${1:-}"
FILE_PATH="${2:-}"

# Only act on file-modifying tools
case "$TOOL_NAME" in
  Edit|Write) ;;
  *) exit 0 ;;
esac

if [ -z "$FILE_PATH" ]; then
  exit 0
fi

# Derive project name from git root or directory name
PROJECT_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
PROJECT_NAME=$(basename "$PROJECT_ROOT")

# Compute relative path
REL_PATH="${FILE_PATH#$PROJECT_ROOT/}"

# Log that this file needs re-indexing (for the next /project-analyze run)
# We write to a simple tracking file rather than calling MCP tools from a hook
STALE_FILE="${PROJECT_ROOT}/.claude/.hippocamp-stale"
echo "$REL_PATH" >> "$STALE_FILE" 2>/dev/null || true

# De-duplicate
if [ -f "$STALE_FILE" ]; then
  sort -u "$STALE_FILE" -o "$STALE_FILE" 2>/dev/null || true
fi
