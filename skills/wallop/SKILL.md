---
name: wallop
description: Local wallop tool governor. Load when the user names wallop, toc, guard, run, log, or asks what went wrong.
license: MIT
metadata:
  version: "1.13"
  release: "1.2.000"
  repo: dan88c/wallop
---

# wallop operator skill

Project release **1.2.000** (`VERSION`). Skill protocol **1.13**.

Working directory = repo root. Set `WALLOP_ROOT` if needed. If `wallop` is not on PATH, use `./bin/wallop` or `.\\bin\\wallop.exe`.

Details live in `references/protocol.md`. Do not preload that file.

## Core directives

1. `wallop toc` before assuming any tool exists.
2. Prefer `wallop run --tool NAME --payload '<json>'` (embeds guard, writes `logs/`).
3. Exit 0 — done. Exit 1 — fix payload, retry ≤2. Exit 3 — unknown tool, ask human. Exit 5 — child failed; stop; do not edit the script.
4. Never invent tool names. Never write `tools/` without a yes.
5. Weak local models must never write tool implementations or Go.

## What is wrong

When the user asks what is wrong / 有什麼問題 / show the error / dump the log:

```powershell
.\\bin\\wallop.exe log --last
```

```bash
./bin/wallop log --last
```

Paste the **full** command output to the user (the `log=` line plus the file body). Do not summarize away the traceback. Do not open or patch `entry` / `tools/*.py`. The human uses that log to fix the script.

## Minimal loop

```powershell
.\\bin\\wallop.exe toc
.\\bin\\wallop.exe run --tool TOOL_NAME --payload '{"param":"value"}'
.\\bin\\wallop.exe log --last
```
