---
name: wallop
description: Recommended operator protocol for the wallop toolbox. Load this skill on startup and whenever the user mentions wallop, tool registry, wallop toc, wallop guard, wallop register, wallop doctor, information wall, developer factory, calendar_gateway (demo-only), time_ops_reader, vault graph, an unknown tool, or wants an existing script indexed in this repo. Conventions only. Direct Python imports remain allowed. Ask before writing tools/ or modifying the YAML catalog.
license: MIT
metadata:
  version: "1.11"
  release: "1.0.001"
  repo: dan88c/wallop
---

# wallop operator skill

Project release: **1.0.001** (`VERSION`). Skill protocol version: 1.11.

This skill lets an agent discover, validate, and run local tools without stuffing full JSON schemas into the prompt.

**Trigger.** Load this file on session start, when the working directory is a wallop checkout, when the user names a wallop command, or when a requested capability is not in the last `wallop toc` output.

Recommendations only. Working directory must be the repo root. Set `WALLOP_ROOT` if the CLI cannot walk up to `config/tool_registry.yaml`. Adapt path separators and piping to the host OS (Bash vs PowerShell).

If `wallop` is not on PATH, invoke the binary from the repo: `./bin/wallop` on Unix, `.\\bin\\wallop.exe` on Windows. Do not assume the bare command `wallop` exists.

## Core directives

1. Read this skill, then run `wallop toc`. Do not assume any tool exists until it appears in the TOC.
2. When the user asks for an action, match it against TOC names. Use `wallop toc --full` only after a guard miss.
3. Validate before run: `wallop guard --tool <name> --payload '<json>'`.
   - Exit 0: safe to run `python tools/<name>.py` with the same JSON on stdin.
   - Exit 1: validation failed. Read stderr, inspect `wallop toc --full`, fix arguments, retry at most twice.
   - Exit 2: bad flags.
   - Exit 3: unknown tool. Stop. Ask: "No tool found. Register an existing script, run the developer factory, or abort?"
   - Exit 4: reserialize a flat JSON object.
4. Never invent tool names. Never write `tools/` or `config/tool_registry.yaml` without a yes.
5. After any register or factory write, run `wallop doctor` (or `python scripts/doctor.py` / `make doctor`). Do not treat the catalog as healthy until doctor exits 0.
6. `calendar_gateway` is demo-only, not a live calendar client.

## Binary setup (no Go toolchain)

Never ask the user to install Go. Do not copy binaries from `dist/` in git; that folder is empty on purpose. Fetch `nightly` from GitHub Releases.

**Linux / macOS**

```bash
python3 scripts/bootstrap.py --download
./bin/wallop toc
```

**Windows (PowerShell)**

```powershell
py -3 scripts\bootstrap.py --download
.\bin\wallop.exe toc
```

Bootstrap ends with the catalog health table from doctor. That table is the source of truth, not a `DEMO OK` line.

Supported CLI commands: `toc`, `guard`, `graph`, `register`, `doctor`, `version`.

`wallop doctor` shells out to `scripts/doctor.py`. If the binary is stale, run the Python script directly.

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
python3 scripts/doctor.py
```

**Windows (PowerShell)**

```powershell
.\bin\wallop.exe toc
.\bin\wallop.exe guard --tool TOOL_NAME --payload '{"param":"value"}'
'{"param":"value"}' | python tools\TOOL_NAME.py
.\bin\wallop.exe graph --vault .\sandbox\vault --query KEYWORD
py -3 scripts\doctor.py
```

## Exit code handling

| Exit | Classification | Agent action |
|------|----------------|--------------|
| 0 | Success | Run the target script with the payload on stdin |
| 1 | Invalid payload / doctor errors | Read stderr. For guard: fix payload, retry at most twice. For doctor: fix catalog before adding more tools |
| 2 | CLI usage | Check flag names |
| 3 | Unknown tool | Stop. Ask: (A) register an existing script, (B) generate via factory, or (C) abort |
| 4 | Bad JSON | Reserialize a valid flat JSON object |

Default catalog names: `calendar_gateway` (**demo-only**, not a live calendar client), `time_ops_reader`.

When the user asks to create, list, update, or delete a real calendar event, do **not** treat `calendar_gateway` as production. Say it is a harness sample. Real writes belong on an edge webhook outside this repo.

## Catalog health (`doctor`)

Run after clone, after `wallop register`, and after the factory. `scripts/bootstrap.py` already runs doctor and prints the health table.

```bash
python3 scripts/bootstrap.py --download
python3 scripts/doctor.py
make doctor
```

Checks:

1. Duplicate `name` entries in `config/tool_registry.yaml`.
2. Each `tools/<name>.py` imports and its `*Input` `BaseModel.model_json_schema()` matches YAML param names, types, and required flags.
3. `HARNESS_TZ` is a usable IANA zone (default `Asia/Hong_Kong`) and `VAULT_PATH` exists (default `./sandbox/vault`).

Exit 0 = no errors. Warnings (for example a Pydantic field missing from YAML) print to stderr but do not fail. Exit 1 = do not add another tool until the errors are fixed.

## Expanding the catalog (ask before writing)

When exit code is 3 or the user already has a script:

1. Never invent tool names. Never modify `tools/` or `config/tool_registry.yaml` without confirmation.
2. Ask: "Tool not found. Register an existing script, generate a new one via the factory, or abort?"
3. If they choose register:

Use `.\\bin\\wallop.exe` on Windows if `bin` is not on PATH.

```bash
./bin/wallop register --entry ./path/to/script.py --name my_tool --desc "Short description" --tag custom --risk read
./bin/wallop toc
python3 scripts/doctor.py
```

```powershell
.\bin\wallop.exe register --entry .\path\to\script.py --name my_tool --desc "Short description" --tag custom --risk read
.\bin\wallop.exe toc
py -3 scripts\doctor.py
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
4. Verify with `./bin/wallop toc` and `python3 scripts/doctor.py` (Windows: `.\\bin\\wallop.exe toc` and `py -3 scripts\\doctor.py`).
