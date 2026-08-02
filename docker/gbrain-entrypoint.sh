#!/usr/bin/env sh
# Initializes the brain against Postgres on first boot (idempotent — gbrain
# doctor tells us if it's already set up), then starts the HTTP MCP server.
set -eu

if ! gbrain doctor --fast >/dev/null 2>&1; then
  echo "[entrypoint] no existing brain state — initializing against Postgres"
  gbrain init --url "$DATABASE_URL" \
    --embedding-model "$GBRAIN_EMBEDDING_MODEL" \
    --embedding-dimensions "$GBRAIN_EMBEDDING_DIMENSIONS"
fi

exec gbrain serve --http --port 7333 --bind 0.0.0.0
