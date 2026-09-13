---
name: wallop
description: Local wallop tool governor. Load when the user names wallop, toc, guard, register, factory, or cwd is a wallop checkout.
license: MIT
metadata:
  version: "1.12"
  release: "1.1.000"
  repo: dan88c/wallop
---

# wallop operator skill

Project release **1.1.000** (`VERSION`). Skill protocol **1.12**.

Load on session start, when cwd is a wallop checkout, when the user names a wallop command, or when a requested capability is missing from the last `wallop toc`.

Working directory = repo root. Set `WALLOP_ROOT` if the CLI cannot walk up to `config/tool_registry.yaml`. Adapt path separators (Bash vs PowerShell).

If `wallop` is not on PATH, use `./bin/wallop` or `.\bin\wallop.exe`. Do not assume the bare command exists. Details, exit table, and host-specific commands live in `references/protocol.md` — read that file only when you need flags, doctor checks, factory retry, or env vars.

## Core directives

1. Run `wallop toc` before assuming any tool exists. Match the user request to TOC names only.
2. Use `wallop toc --full` or `wallop toc --tag <tag>` only after a guard miss or parameter uncertainty.
3. Validate then run. Same JSON on stdin to `python tools/<name>.py`.
   - `wallop guard --tool <name> --payload '<json>'`
   - Exit 0 — run the script.
   - Exit 1 — read stderr, fix args, retry guard at most twice.
   - Exit 2 — fix flags.
   - Exit 3 — unknown tool. Stop. Ask register, factory, or abort.
   - Exit 4 — reserialize a flat JSON object.
4. Never invent tool names. Never write `tools/` or `config/tool_registry.yaml` without a yes.
5. After register or factory, run `wallop doctor` (or `python scripts/doctor.py`). Do not add another tool until doctor exits 0. Warnings on stderr are OK.
6. `calendar_gateway` is demo-only, not a live calendar client.
7. Weak local models (14B–36B) must never write tool implementations, Go code, or shell wrappers.

## When to Load This Skill

- Cold start or cwd is a wallop checkout: ask once if this session has not approved yet.
- Explicit request (user named wallop, toc, guard, register, doctor, factory): load. Do not ask again this session.
- No other skill fits and you would otherwise guess: ask once, then load. Do not invent a command.
- Already loaded and the last `wallop toc` has no match: stay loaded; follow exit 3 (register / factory / abort).

## Binary (no Go required)

Do not ask the user to install Go. Do not copy from git `dist/` — that folder is `.gitkeep` only. Fetch Releases via bootstrap:

```bash
python3 scripts/bootstrap.py --download
./bin/wallop version
./bin/wallop toc
```

```powershell
py -3 scripts\bootstrap.py --download
.\bin\wallop.exe version
.\bin\wallop.exe toc
```

Bootstrap installs `.venv`, places `bin/wallop` (Go build if present, else `WALLOP_RELEASE` default `nightly` from GitHub Releases, else a leftover local `dist/wallop-*`), then prints the doctor catalog-health table. That table is the source of truth — there is no `DEMO OK` line.

Commands: `toc`, `guard` (alias `validate`), `graph`, `register`, `doctor`, `version` (`-v` / `--version`), `help`.

`wallop doctor` shells out to `scripts/doctor.py`. If the binary is stale, run the Python script.

## Minimal loop

```bash
./bin/wallop toc
./bin/wallop guard --tool TOOL_NAME --payload '{"param":"value"}'
echo '{"param":"value"}' | python tools/TOOL_NAME.py
```

Default catalog: `calendar_gateway` (demo), `time_ops_reader`.

Real calendar create/list/update/delete is not this demo tool. Say so. Live writes belong on an edge webhook outside this repo.

## Expand catalog (ask first)

On exit 3 or an existing script:

1. Ask: register an existing script, generate via factory, or abort?
2. Register path (then doctor):

```bash
./bin/wallop register --entry ./path/to/script.py --name my_tool --desc "Short description" --tag custom --risk read
./bin/wallop toc
python3 scripts/doctor.py
```

Copies to `tools/<name>.py` unless `--keep-path`. No secrets in the repo.

## Developer factory

Only after explicit approval. The 14B–36B operator is not the factory model.

- `MOCK_MODE=true` (default): local stub, no API key.
- Live (`MOCK_MODE=false`): needs `CLOUD_DEVELOPER_URL` / `CLOUD_DEVELOPER_TOKEN`. If unset, stop. Do not invent a key. Prompt the user to configure `.env` or stay in mock.
- Scrub PII, names, titles, dates, paths, credentials from `--brief` before any cloud call.

```bash
python developer-factory/factory_agent.py --brief "Abstract functional requirement without personal data or dates"
python3 scripts/doctor.py
```

Factory writes `tools/<name>.py` then pytest. **On test failure the YAML is not updated.** Retry at most twice with a cleaner abstract / explicit `--name --desc --param`. After two failures, stop and ask the human. Then doctor + `wallop toc`.

Full CLI, exits, doctor checks, payload shape, env: `references/protocol.md`.
