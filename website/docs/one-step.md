# One step

Trajectory IR sits beside the agent you already run. One command builds the verify binary, points Claude Code at project-local hooks, and checks one pack.

```powershell
pwsh -File scripts\install_claude.ps1
pwsh -File scripts\install_claude.ps1 -Target C:\path\to\your\project
```

```bash
./scripts/install_claude.sh --target /path/to/your/project
./go/trajir.exe verify pack.tir
```

`--target` writes the config into that project. The hook scripts stay in this install. The user profile is not edited. The same command writes Cursor, VS Code, Gemini CLI, and Codex project MCP config. Those MCP tools are `trajectory_status`, `trajectory_export_tir`, `trajectory_import_tir`, and `trajectory_verify_signature`. They do not seal the host's own tools.

If nothing is listening on `127.0.0.1:8787`, the installer starts `trajir-console` there. If the port is taken, it leaves that process alone. Pass `--skip-console` to skip the start. There is no second console.

The first PreToolUse seals the decision and exits 0 so the host can run the tool. A second PreToolUse for the same `tool_use_id`, when that seal is `NON_IDEMPOTENT_WRITE`, exits 2. The hook does not call the tool and does not start DBOS. PostToolUse records the result. If that diary write fails, PostToolUse still exits 0 so the host is not asked to run the tool again. PostToolUseFailure records a failed tool and clears its pending row.

LangGraph still gates inside the tool wrap. The first body runs. A later resume does not run that gated body again.

```python
guard = TrajIRToolGuard(tenant_id="demo", trajectory_id="lg-live", work_dir=".local-evidence/lg-live")
tools = ToolNode(my_tools, wrap_tool_call=guard.wrap_tool_call)
```

Google ADK uses callbacks. They seal and record. They do not call the tool. A missing call id raises ValueError. This page does not claim a live ADK agent has been run.

```python
guard = AdkToolGuard(tenant_id="demo", trajectory_id="adk-live", work_dir=".local-evidence/adk-live")
agent = Agent(tools=my_tools, before_tool_callback=guard.before_tool_callback, after_tool_callback=guard.after_tool_callback)
```

Snippets for those two calls are written to `.trajir/` in the target project. Importing a snippet does not import LangGraph, ADK, or DBOS.

An unknown tool, and any tool named Bash, is `NON_IDEMPOTENT_WRITE`.

Token numbers on the console are `ceil(chars/4)` estimates. They are not a provider invoice. The adopter list is empty. This is not exactly-once, not a process sandbox, and not Sigstore. OpenTelemetry labels are copied only after the in-process seal.

The Article 12 field map is [Article 12 field map](article-12-field-map.md). It names the pack fields. It is not a compliance claim.

This page is published when GitHub Actions builds this docs folder. The link above stays inside this folder.
