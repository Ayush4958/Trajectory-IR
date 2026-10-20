#!/usr/bin/env bash
# One-step local attach for Claude Code.
# Does not edit the user profile. Does not git commit.
# Starts trajir-console on 127.0.0.1:8787 only when that port is free.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

TARGET=""
SKIP_CONSOLE=0
EXTRA=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --target)
      if [[ $# -lt 2 ]]; then
        echo "missing directory after --target" >&2
        exit 1
      fi
      TARGET="$2"
      EXTRA+=(--target "$2")
      shift 2
      ;;
    --skip-console)
      SKIP_CONSOLE=1
      EXTRA+=(--skip-console)
      shift
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

echo "== One-step Claude attach =="
echo "  root: $ROOT"
if [[ -n "$TARGET" ]]; then
  echo "  target: $TARGET"
fi

# go/trajir is the package directory. The binary is go/trajir.exe so the
# build does not write go/trajir/trajir and then treat the directory as OK.
# trajir-mcp.exe and trajir-console.exe use the same suffix on every OS.
(
  cd "$ROOT/go"
  go build -o trajir.exe ./cmd/trajir
  go build -o trajir-mcp.exe ./cmd/trajir-mcp
  go build -o trajir-console.exe ./cmd/trajir-console
)
test -f "$ROOT/go/trajir.exe"
test -x "$ROOT/go/trajir.exe" || chmod +x "$ROOT/go/trajir.exe"
test -f "$ROOT/go/trajir-mcp.exe"
test -f "$ROOT/go/trajir-console.exe"

if [[ ${#EXTRA[@]} -gt 0 ]]; then
  python "$ROOT/scripts/write_claude_local_config.py" "${EXTRA[@]}"
else
  python "$ROOT/scripts/write_claude_local_config.py"
fi
python "$ROOT/scripts/local_one_step_smoke.py"

echo ""
if [[ -n "$TARGET" ]]; then
  echo "Attached $TARGET. Hook scripts stay in this install."
else
  echo "Attached inside this worktree only."
fi
echo "  Hooks: .claude/settings.local.json"
echo "  MCP:   .mcp.json .cursor/mcp.json .vscode/mcp.json .gemini/settings.json .codex/config.toml"
echo "  Verify a pack with: ./go/trajir.exe verify <pack.tir>"
if [[ "$SKIP_CONSOLE" -eq 1 ]]; then
  echo "Console start skipped."
fi
echo "The user profile was not changed."
echo "Console is 127.0.0.1:8787 only. A listener already on that port is left alone."
echo "INSTALL OK"
