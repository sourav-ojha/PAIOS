#!/usr/bin/env bash
# Copies the offenFlow architecture corpus into the shared brain's git repo.
# Source is untouched (read-only copy); brains/shared is what gbrain indexes.
set -euo pipefail

SRC_DOCS="/Users/sourav/workspace/offenlix/offenflow_project/offenFlow-main/docs/architecture"
SRC_SKILLS="/Users/sourav/workspace/offenlix/offenflow_project/offenFlow-main/.claude/skills"
DEST="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/brains/shared/offenflow"

mkdir -p "$DEST/architecture" "$DEST/skills"
cp -R "$SRC_DOCS/." "$DEST/architecture/"
cp -R "$SRC_SKILLS/." "$DEST/skills/"

echo "Copied to $DEST"
echo "Next: cd brains/shared && git init (if not already) && git add -A && git commit"
echo "Then: gbrain import ./brains/shared/offenflow/"
