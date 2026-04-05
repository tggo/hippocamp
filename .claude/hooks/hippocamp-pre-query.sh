#!/bin/bash
# Hippocamp pre-query hook for Claude Code.
# Runs on UserPromptSubmit: queries the knowledge graph for context relevant
# to the user's prompt and outputs it to stderr (visible to Claude).
#
# Install in .claude/settings.json:
#   "hooks": {
#     "UserPromptSubmit": [{ "command": ".claude/hooks/hippocamp-pre-query.sh \"$PROMPT\"" }]
#   }
#
# Requires: hippocamp binary in PATH, persisted graph in data/default.trig

set -euo pipefail

PROMPT="${1:-}"
if [ -z "$PROMPT" ]; then
  exit 0
fi

# Find hippocamp binary (project-local build, then PATH)
HIPPOCAMP=""
if [ -x "./hippocamp" ]; then
  HIPPOCAMP="./hippocamp"
elif command -v hippocamp &>/dev/null; then
  HIPPOCAMP="hippocamp"
else
  exit 0  # No hippocamp binary — skip silently
fi

# Find config file
CONFIG=""
if [ -f "./config.yaml" ]; then
  CONFIG="./config.yaml"
elif [ -f "$HOME/.config/hippocamp/config.yaml" ]; then
  CONFIG="$HOME/.config/hippocamp/config.yaml"
else
  exit 0
fi

# Check if there's a persisted graph to query
GRAPH_FILE=$(grep -oP 'default_file:\s*"\K[^"]+' "$CONFIG" 2>/dev/null || echo "./data/default.trig")
if [ ! -f "$GRAPH_FILE" ]; then
  exit 0
fi

# Extract keywords from prompt (first 5 significant words, skip common ones)
KEYWORDS=$(echo "$PROMPT" | tr '[:upper:]' '[:lower:]' | \
  tr -cs '[:alnum:]' '\n' | \
  grep -vE '^(the|a|an|is|are|was|were|be|been|do|does|did|have|has|had|will|would|could|should|can|may|might|shall|not|and|or|but|in|on|at|to|for|of|with|from|by|as|it|this|that|what|how|why|where|when|which|who|i|me|my|we|us|our|you|your|they|them|their|please|help|want|need|make|show|get|find|use|look|let|add|change|fix|update|create|delete|remove|file|code)$' | \
  head -5 | tr '\n' ' ')

if [ -z "$KEYWORDS" ]; then
  exit 0
fi

# Query the graph
RESULT=$("$HIPPOCAMP" --config "$CONFIG" --query "$KEYWORDS" --limit 10 2>/dev/null) || exit 0

# Only output if we got meaningful results (not empty array)
if [ "$RESULT" != "[]" ] && [ -n "$RESULT" ]; then
  echo "--- Hippocamp knowledge graph context ---" >&2
  echo "$RESULT" >&2
  echo "--- end graph context ---" >&2
fi
