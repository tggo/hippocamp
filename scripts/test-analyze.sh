#!/bin/bash
# Test the /project-analyze skill against testdata projects using Claude CLI.
#
# Usage:
#   ./scripts/test-analyze.sh [project-name]
#   ./scripts/test-analyze.sh house-construction
#   ./scripts/test-analyze.sh                      # runs all projects
#
# Requires: claude CLI, hippocamp binary built

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HIPPOCAMP="$PROJECT_ROOT/hippocamp"
TESTDATA="$PROJECT_ROOT/testdata"

# Build hippocamp if needed.
if [ ! -x "$HIPPOCAMP" ]; then
  echo "Building hippocamp..."
  cd "$PROJECT_ROOT" && make build
fi

# Determine which projects to test.
if [ $# -gt 0 ]; then
  PROJECTS=("$@")
else
  PROJECTS=(house-construction tomato-garden sales-department accounting recipe-collection)
fi

PASS=0
FAIL=0

for proj in "${PROJECTS[@]}"; do
  proj_dir="$TESTDATA/$proj"
  if [ ! -d "$proj_dir" ]; then
    echo "SKIP: $proj (directory not found)"
    continue
  fi

  echo ""
  echo "════════════════════════════════════════════════"
  echo "  Analyzing: $proj"
  echo "════════════════════════════════════════════════"

  # Create a temp working directory with the project files + hippocamp config.
  work_dir=$(mktemp -d)
  cp -r "$proj_dir"/* "$work_dir/"
  cp "$PROJECT_ROOT/config.yaml" "$work_dir/"
  mkdir -p "$work_dir/data"

  # Copy the skill so Claude can use it.
  mkdir -p "$work_dir/.claude/skills"
  cp "$PROJECT_ROOT/internal/setup/embedded/claude/skills/project-analyze.md" "$work_dir/.claude/skills/"

  # Run Claude with the project-analyze skill prompt.
  echo "Running Claude /project-analyze in $work_dir..."

  ANALYZE_PROMPT="You have the hippocamp MCP server connected. Run the /project-analyze skill on this project directory. Read the files, extract knowledge, and populate the graph. After populating, run: graph action=dump file=./data/default.trig. Then report what you indexed."

  cd "$work_dir"
  if claude -p "$ANALYZE_PROMPT" \
    --allowedTools "mcp__hippocamp__triple,mcp__hippocamp__sparql,mcp__hippocamp__graph,mcp__hippocamp__search,Read,Glob" \
    --mcp-config <(cat <<EOF
{
  "mcpServers": {
    "hippocamp": {
      "command": "$HIPPOCAMP",
      "args": ["--config", "$work_dir/config.yaml"]
    }
  }
}
EOF
) 2>&1 | tee "$work_dir/claude-output.txt"; then
    echo ""
  else
    echo "WARNING: Claude exited with non-zero status"
  fi

  # Verify: check if the graph was populated.
  if [ -f "$work_dir/data/default.trig" ]; then
    trig_size=$(wc -c < "$work_dir/data/default.trig")
    echo "  TriG file created: $(echo $trig_size) bytes"

    # Query the graph for a basic sanity check — search for a common term.
    echo "  Querying graph..."
    # Try several queries to find anything in the graph.
    found=false
    for search_term in "project" "budget" "decision" "note" "topic"; do
      result=$("$HIPPOCAMP" --config "$work_dir/config.yaml" --query "$search_term" --limit 5 2>/dev/null) || true
      if [ "$result" != "[]" ] && [ -n "$result" ] && [ "$result" != "null" ]; then
        count=$(echo "$result" | python3 -c "import sys,json; print(len(json.load(sys.stdin)))" 2>/dev/null || echo "?")
        echo "  Search for '$search_term': $count results"
        found=true
        break
      fi
    done

    if $found; then
      echo "  PASS"
      PASS=$((PASS + 1))
    else
      echo "  FAIL — graph populated ($trig_size bytes) but no search results"
      FAIL=$((FAIL + 1))
    fi
  else
    echo "  FAIL — no TriG file created"
    FAIL=$((FAIL + 1))
  fi

  # Cleanup.
  rm -rf "$work_dir"
done

echo ""
echo "════════════════════════════════════════════════"
echo "  Results: $PASS passed, $FAIL failed (${#PROJECTS[@]} total)"
echo "════════════════════════════════════════════════"

[ "$FAIL" -eq 0 ]
