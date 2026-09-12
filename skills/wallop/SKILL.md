---
name: wallop
description: Recommended operator protocol for the wallop toolbox. Use when the user mentions wallop, tool registry, wallop toc, wallop guard, wallop register, information wall, developer factory, calendar_gateway, time_ops_reader, vault graph, or wants an existing script indexed in this repo. Conventions only. Direct Python imports remain allowed. Ask before writing tools/ or modifying the YAML catalog.
license: MIT
metadata:
  version: "1.9"
  repo: dan88c/wallop
---

# wallop operator skill

Recommendations only. Working directory must be the repo root. Set `WALLOP_ROOT` if the CLI cannot walk up to `config/tool_registry.yaml`. Adapt path separators and piping to the host OS (Bash vs PowerShell).

If `wallop` is not on PATH, invoke the binary from the repo: `./bin/wallop` on Unix, `.\bin\wallop.exe` on Windows. Do not assume the bare command `wallop` exists.

## Binary setup (no Go toolchain)

Never ask the user to install Go. Use pre-built binaries from `dist/` or GitHub Releases.

**Linux / macOS**

```bash
mkdir -p bin
cp dist/wallop-linux-amd64 bin/wallop && chmod +x bin/wallop
# macOS Apple Silicon:
# cp dist/wallop-darwin-arm64 bin/wallop && chmod +x bin/wallop
./bin/wallop toc
```

**Windows (PowerShell)**

```powershell
New-Item -ItemType Directory -Force bin | Out-Null
Copy-Item dist\wallop-windows-amd64.exe bin\wallop.exe
.\bin\wallop.exe toc
```

Supported CLI commands: `toc`, `guard`, `graph`, `register`.

## Catalog surface policy

1. Default prompt surface: load `wallop toc` only (short lines: names + tags).
2. Fallback: run `wallop toc --full` only after a guard miss or parameter uncertainty.
3. Model constraint: weak local models (14B–36B) must never write tool implementations, Go code, or shell wrappers directly.

## Recommended execution flow

Always validate payloads with `guard` before invoking Python scripts. Pass the same JSON on stdin.

**Linux / macOS (Bash)**

```bash
./bin/wallop toc
./bin/wallop guard --tool TOOL_NAME --payload '{"param":"value"}'
echo '{"param":"value"}' | python tools/TOOL_NAME.py
./bin/wallop graph --vault ./sandbox/vault --query KEYWORD
```

**Windows (PowerShell)**

```powershell
.\bin\wallop.exe toc
.\bin\wallop.exe guard --tool TOOL_NAME --payload '{"param":"value"}'
'{"param":"value"}' | python tools\TOOL_NAME.py
.\bin\wallop.exe graph --vault .\sandbox\vault --query KEYWORD
```

## Exit code handling

| Exit | Classification | Agent action |
|------|----------------|--------------|
| 0 | Success | Run the target script with the payload on stdin |
| 1 | Invalid payload | Read stderr, inspect `wallop toc --full`, fix payload. Retry at most twice |
| 2 | CLI usage | Check flag names |
| 3 | Unknown tool | Stop. Ask: (A) register an existing script, (B) generate via factory, or (C) abort |
| 4 | Bad JSON | Reserialize a valid flat JSON object |

Default catalog names: `calendar_gateway`, `time_ops_reader`.

## Expanding the catalog (ask before writing)

When exit code is 3 or the user already has a script:

1. Never invent tool names. Never modify `tools/` or `config/tool_registry.yaml` without confirmation.
2. Ask: "Tool not found. Register an existing script, generate a new one via the factory, or abort?"
3. If they choose register:

Use `.\bin\wallop.exe` on Windows if `bin` is not on PATH.

```bash
./bin/wallop register --entry ./path/to/script.py --name my_tool --desc "Short description" --tag custom --risk read
./bin/wallop toc
```

```powershell
.\bin\wallop.exe register --entry .\path\to\script.py --name my_tool --desc "Short description" --tag custom --risk read
.\bin\wallop.exe toc
```

Copies to `tools/<name>.py` and updates the YAML. `--keep-path` stores the original path instead of copying. Do not put secrets in the repo.

## Developer factory (on-demand synthesis)

Only if the user explicitly approves generating a new tool.

### Factory prerequisites

- Generating a *real* new tool requires a frontier coding model (Claude 3.5 Sonnet, GPT-4o, DeepSeek-Coder via OpenRouter/API, or a strong 70B+ local coder). Do not use the 14B–36B operator.
- Default `MOCK_MODE=true` needs no second LLM and no API key. It only writes a deterministic stub.
- If the user wants live synthesis (`MOCK_MODE=false`) and `CLOUD_DEVELOPER_URL` or `CLOUD_DEVELOPER_TOKEN` is unset in `.env`, **do not attempt generation and do not invent a key**. Prompt the user:

  > To synthesize new tools, configure a developer LLM in `.env` (`CLOUD_DEVELOPER_URL`, `CLOUD_DEVELOPER_TOKEN`) or keep `MOCK_MODE=true` for mock testing.

- Never hallucinate tokens. Never paste secrets into chat logs if the user pastes them once for setup.

1. Information wall: scrub PII, names, calendar titles, dates, file paths, and credentials from the brief.
2. Dispatch an abstract spec:

```bash
python developer-factory/factory_agent.py --brief "Abstract functional requirement without personal data or dates"
```

3. The factory runs local pytest. Only passing tools are registered.
4. Verify with `./bin/wallop toc` (Windows: `.\bin\wallop.exe toc`).
