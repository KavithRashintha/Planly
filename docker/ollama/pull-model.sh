#!/bin/bash
# Ollama model pull script — runs after Ollama server starts
# This ensures the required model is available before the agent-svc connects.

set -e

MODEL="${OLLAMA_MODEL:-qwen2.5:7b}"
OLLAMA_HOST="${OLLAMA_HOST:-http://localhost:11434}"

echo "[ollama-init] Waiting for Ollama server to be ready..."
until curl -sf "$OLLAMA_HOST/api/tags" > /dev/null 2>&1; do
  sleep 2
done

echo "[ollama-init] Ollama is ready. Checking if model '$MODEL' is available..."

# Check if model already pulled
if ollama list 2>/dev/null | grep -q "^${MODEL}"; then
  echo "[ollama-init] Model '$MODEL' is already available."
else
  echo "[ollama-init] Pulling model '$MODEL' (this may take a few minutes on first run)..."
  ollama pull "$MODEL"
  echo "[ollama-init] Model '$MODEL' pulled successfully."
fi

echo "[ollama-init] Done."
