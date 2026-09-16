# Wallop CLI (`cmd/wallop`)

`main.go` is the compiled `wallop` binary. It is a thin dispatcher: frontline agents get a small, deterministic command set; catalog, spawn, privacy, and cloud synthesis live in `core-operator/pkg/*`.

Each tool is a JSON pointer at `~/.config/wallop/tools/<name>.json`. Payload validation runs inside `wallop run` before the child process starts.

Version: `2.0.000`.

## Layout

All paths are resolved relative to the user's home directory (`~`), determined cross-platform via `os.UserHomeDir()`:
- **POSIX (Linux/macOS):** `/home/<username>` or `/Users/<username>`
- **Windows:** `C:\Users\<username>` *(Note: Wallop uses explicit `~/.config/wallop` and does not use `%APPDATA%` or `%LOCALAPPDATA%`)*

| Path | Role | Permissions |
|---|---|---|
| `~/.config/wallop/` | Base configuration directory | `0755` (directory) |
| `~/.config/wallop/secrets.env` | Secrets & developer credentials | `0600` on POSIX (regular file on Windows) |
| `~/.config/wallop/config.json` | Local inference endpoint config | `0644` |
| `~/.config/wallop/tools/*.json` | One JSON pointer per tool | `0644` |
| `~/.local/state/wallop/events.jsonl`| Execution audit log | Appended (`0644`) |
| `~/.local/share/wallop/generated/`  | Atomically written synthesized scripts | `0755` (scripts marked `+x`) |

## Commands

```
wallop <check|run|escalate|doctor|init|register|version> [flags]
```

| Command | Purpose |
|---|---|
| `check [query]` | List pointers whose command still resolves. Optional substring filter on name or description. JSON to stdout. |
| `run --tool NAME --payload FILE` | Validate payload against the pointer schema, spawn the tool, audit the result. Payload is stdin to the child only. |
| `escalate --reason TEXT --context FILE` | Local de-identify, then cloud-synthesize a script. Requires `wallop init` and developer credentials. |
| `doctor` | Health of secrets perms, writable dirs, tool probes, and the local inference endpoint. Exit `1` on any `[FAIL]`. |
| `init` | Interactive setup of the local OpenAI-compatible masker endpoint and model. |
| `register --name ID --cmd PATH [--desc TEXT] [--workdir DIR] [--probe ARG]` | Resolve `PATH` and write `tools/<ID>.json`. |
| `version` | Print `wallop 2.0.000`. Aliases: `-v`, `--version`. |

### `run`

- `--tool` and `--payload` are required. `--payload` is a path to a JSON file.
- Tool names must match `^[A-Za-z0-9_-]+$`. The pointer file is forced to stay under `tools/`.
- Empty schema skips validation. Malformed schema fails closed.
- Recursion is capped (`WALLOP_DEPTH`, max 3). The child inherits `WALLOP_DEPTH` only; the payload is not copied into the environment.
- Exit codes: `2` usage / bad name, `3` missing tool or binary, `4` unreadable payload, `1` guard failure, otherwise the child exit code.

### `register`

- `--name` and `--cmd` are required.
- `cmd` is resolved the same way as `run` (workdir, direct path, then `PATH`). The stored command is the resolved absolute path.
- `--probe` is an optional healthcheck argument used by `doctor`, not by `check`.

### `escalate`

1. Confirm the local masker (`config.json` `confirmed`).
2. Mask `--reason` and `--context` using the local model. Empty model output fails closed.
3. Load credentials (`CLOUD_DEVELOPER_TOKEN`, `CLOUD_DEVELOPER_URL`) from the environment, or fall back to `~/.config/wallop/secrets.env`.
   - **Permission Boundary**: On POSIX systems, `secrets.env` **must** have strict file permissions (`0600`). If group/others have read access (e.g., `0644`), `wallop` rejects the file and aborts execution. On Windows, file regularity is verified.
4. Atomically commit the synthesized script to `~/.local/share/wallop/generated/` and output the execution summary.

## Packages

| Package | Used by |
|---|---|
| `pkg/pointer` | Name bounds, pointer I/O, schema guard, spawn resolution |
| `pkg/audit` | `run` JSONL events |
| `pkg/config` | `init` / `doctor` local endpoint |
| `pkg/masker` | `escalate` de-identification |
| `pkg/gateway` | Secrets load + cloud synthesis |
| `pkg/doctor` | `wallop doctor` |

## Build

```
cd core-operator && go build -o ../bin/wallop ./cmd/wallop
```
