"""Local identity for the Claude hooks. No secrets."""

from __future__ import annotations

import json
import os
from pathlib import Path

_ROOT = Path(__file__).resolve().parents[2]
_CONFIG = _ROOT / ".local-evidence" / "claude-one-step.json"


def _file_cfg(path: Path) -> dict[str, str]:
    loaded = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(loaded, dict):
        raise ValueError(f"{path} must be a JSON object")
    return {str(key): str(value) for key, value in loaded.items()}


def _identity_file() -> Path | None:
    """TRAJIR_IDENTITY_FILE wins. Then the cwd file. Then this checkout."""
    env_file = os.environ.get("TRAJIR_IDENTITY_FILE")
    if env_file:
        path = Path(env_file)
        if not path.is_file():
            raise FileNotFoundError(f"TRAJIR_IDENTITY_FILE not found: {path}")
        return path
    cwd_file = Path.cwd() / ".local-evidence" / "claude-one-step.json"
    if cwd_file.is_file():
        return cwd_file
    if _CONFIG.is_file():
        return _CONFIG
    return None


def hook_identity() -> dict[str, str]:
    chosen = _identity_file()
    file_cfg = _file_cfg(chosen) if chosen is not None else {}
    work = os.environ.get("TRAJIR_WORK_DIR") or file_cfg.get("work_dir")
    if not work:
        work = str(_ROOT / ".local-evidence" / "claude-agent")
    return {
        "work_dir": work,
        "tenant_id": os.environ.get("TRAJIR_TENANT_ID") or file_cfg.get("tenant_id") or "local",
        "trajectory_id": os.environ.get("TRAJIR_TRAJECTORY_ID")
        or file_cfg.get("trajectory_id")
        or "claude-one-step",
    }


def tool_use_id(event: dict) -> str:
    raw = event.get("tool_use_id")
    if raw:
        return str(raw)
    raise ValueError("tool_use_id is required")
