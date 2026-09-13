# wallop call protocol

Direct `python tools/<name>.py` imports remain valid. This file is the cheap CLI path for a weak local operator. Read it when SKILL.md is not enough.

Binary: `./bin/wallop` or `.\bin\wallop.exe`. Windows uses `py -3` and backslashes.

## Commands

```text
wallop toc [--full] [--tag name] [--registry path]
wallop guard --tool <name> [--payload <json>]     # alias: validate
wallop graph --vault <path> [--query keyword] [--ext .md]
wallop register --entry <script.py> --name <id> [--desc ...] [--tag t]
            [--param name:type[:required]] [--risk read|write] [--keep-path]
wallop doctor [--registry path]
wallop version | -v | --version
wallop help | -h | --help
```

`wallop` walks up from cwd (or `WALLOP_ROOT`) to `config/tool_registry.yaml`.

Guard payload is a **flat JSON object of tool arguments**. Pass the name with `--tool`. The CLI injects `tool` into the object. Do not require a `tool` key inside `--payload`.

Legacy: `--call '{"tool":"calendar_gateway","action":"list"}'` still works.

```bash
./bin/wallop guard --tool calendar_gateway --payload '{"action":"list"}'
echo '{"action":"list"}' | python tools/calendar_gateway.py
./bin/wallop graph --vault ./sandbox/vault --query home
python3 scripts/doctor.py
```

```powershell
.\bin\wallop.exe guard --tool calendar_gateway --payload '{"action":"list"}'
'{"action":"list"}' | python tools\calendar_gateway.py
.\bin\wallop.exe graph --vault .\sandbox\vault --query home
py -3 scripts\doctor.py
```

## Exit codes

| Exit | Meaning | Agent action |
|------|---------|--------------|
| 0 | Guard accepted / doctor healthy / version printed | Run `tools/<name>.py` with the same JSON on stdin, or continue |
| 1 | Invalid payload or doctor errors | Guard: fix payload, retry ≤2. Doctor: fix catalog before adding tools |
| 2 | Usage / bad flags | Check command names |
| 3 | Unknown tool or missing registry | Stop. Ask register, factory, or abort |
| 4 | Unreadable JSON | Reserialize one flat object |

Writes (`risk: write` or action in create\|update\|delete\|write\|set) reject a timestamp more than two minutes in the past (`date-drift`). Datetimes RFC3339. Dates `YYYY-MM-DD`. Timezone from YAML / `HARNESS_TZ` (default `Asia/Hong_Kong`).

Tools read JSON on stdin and print one JSON object on stdout.

## Bootstrap and doctor

```bash
python3 scripts/bootstrap.py --download
python3 scripts/doctor.py
make doctor
make download
```

Bootstrap source of truth is the catalog health table, not `DEMO OK`.

Binary resolution order: Go on PATH → leftover `dist/wallop-*` → GitHub Releases tag `WALLOP_RELEASE` (default `nightly`). Git `dist/` is empty on purpose.

Doctor checks:

1. No duplicate `name` in `config/tool_registry.yaml`.
2. Each `tools/<name>.py` imports; `*Input` `BaseModel.model_json_schema()` matches YAML param names, types, required flags.
3. `HARNESS_TZ` is a usable IANA zone; `VAULT_PATH` exists (default `./sandbox/vault`).

Exit 0 with warnings is fine (demo `calendar_gateway` may warn about an extra Pydantic field). Exit 1 = stop adding tools.

## Factory retry

```bash
python developer-factory/factory_agent.py --name wiki_search --desc "Keyword search over a vault" --tag read --param query:string:required
python developer-factory/factory_agent.py --brief "Abstract functional requirement without personal data or dates"
```

`--brief` goes through `assert_safe_for_cloud` then `cloud_client.propose`. `MOCK_MODE=true` keeps the proposer local. `MOCK_MODE=false` posts only the sanitized abstract to `CLOUD_DEVELOPER_URL`.

Failure rules:

1. Missing live URL/token with `MOCK_MODE=false` — do not invent credentials. Ask the user or drop back to mock.
2. pytest fail — factory prints `factory: tests failed; registry not updated` and leaves YAML unchanged. The `.py` file may already exist.
3. Retry at most **twice**. Tighten the abstract; prefer explicit `--name --desc --param`. Same `--name` overwrites the stub.
4. After two failures — abort. Show stderr. Ask the human. Do not hand-edit the stub as the 14B–36B operator.
5. On success — `wallop toc` then `python3 scripts/doctor.py`.

## Env

| Variable | Meaning |
|----------|---------|
| `WALLOP_ROOT` | Override repo root |
| `WALLOP_RELEASE` | Release tag for `--download` (default `nightly`) |
| `MOCK_MODE` | `true` skips sockets |
| `VAULT_PATH` | Markdown vault for `wallop graph` and doctor |
| `HARNESS_TZ` | Clock / past-date boundary |
| `TIME_OPS_LOG_PATH` | Exception log |
| `EDGE_CALENDAR_WEBHOOK` | Demo edge URL only |
| `CLOUD_DEVELOPER_URL` | Live factory proposer |
| `CLOUD_DEVELOPER_TOKEN` | Optional bearer |
