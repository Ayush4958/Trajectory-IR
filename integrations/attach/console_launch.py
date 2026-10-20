"""Start trajir-console on 127.0.0.1:8787 when that port is free.

Any other address is refused. A listener that is already up is left alone.
"""

from __future__ import annotations

import os
import socket
import subprocess
from collections.abc import Callable
from dataclasses import dataclass
from pathlib import Path

CONSOLE_ADDR = "127.0.0.1:8787"


@dataclass(frozen=True)
class ConsoleLaunch:
    status: str
    process: subprocess.Popen[bytes] | None = None


def console_is_up(addr: str = CONSOLE_ADDR, timeout: float = 0.3) -> bool:
    """Return True when something accepts TCP on the console address."""
    if addr != CONSOLE_ADDR:
        raise ValueError("console address must be 127.0.0.1:8787")
    host, port_text = addr.rsplit(":", 1)
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.settimeout(timeout)
        return sock.connect_ex((host, int(port_text))) == 0


def console_binary(root: Path) -> Path:
    """Return go/trajir-console.exe, building it when the file is missing."""
    exe = Path(root) / "go" / "trajir-console.exe"
    if exe.is_file():
        return exe
    exe.parent.mkdir(parents=True, exist_ok=True)
    subprocess.check_call(
        ["go", "build", "-o", str(exe), "./cmd/trajir-console"],
        cwd=str(Path(root) / "go"),
    )
    return exe


def start_console_if_free(
    root: Path,
    *,
    data_dir: Path | None = None,
    detach: bool = True,
    probe: Callable[[], bool] | None = None,
    popen: Callable[..., subprocess.Popen[bytes]] | None = None,
) -> ConsoleLaunch:
    """Start the console, or report that 127.0.0.1:8787 is already taken."""
    listening = console_is_up if probe is None else probe
    if listening():
        return ConsoleLaunch("already")
    binary = console_binary(root)
    data = Path(data_dir) if data_dir is not None else Path(root) / ".console-data"
    data.mkdir(parents=True, exist_ok=True)
    cmd = [str(binary), "-addr", CONSOLE_ADDR, "-data", str(data)]
    kwargs: dict[str, object] = {
        "cwd": str(Path(root) / "go"),
        "stdout": subprocess.DEVNULL,
        "stderr": subprocess.DEVNULL,
    }
    if detach:
        if os.name == "nt":
            flags = getattr(subprocess, "DETACHED_PROCESS", 0)
            flags |= getattr(subprocess, "CREATE_NEW_PROCESS_GROUP", 0)
            kwargs["creationflags"] = flags
        else:
            kwargs["start_new_session"] = True
    spawn = subprocess.Popen if popen is None else popen
    process = spawn(cmd, **kwargs)
    return ConsoleLaunch("started", process)
