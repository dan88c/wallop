# wallop call protocol (recommended)

Direct Python imports of `tools/*.py` remain supported. This file describes the cheap CLI path for a weak local operator.

## Guard payload shape

```json
{
  "tool": "calendar_gateway",
  "action": "create",
  "start": "2026-09-13T19:00:00+08:00",
  "title": "demo-block"
}
```

- One object. Prefer flat keys.
- `tool` required for `wallop guard`.
- Datetimes RFC3339. Dates `YYYY-MM-DD`.
- Timezone for past-date checks comes from `config/tool_registry.yaml` (default Asia/Hong_Kong).
- Writes are `risk: write` tools or `action` in create|update|delete|write|set.
- A write timestamp more than two minutes in the past fails with `date-drift`.

## TOC disclosure

- First prompt: `wallop toc` (names + tags).
- After a guard miss: `wallop toc --full` or a single tool slice.
- Do not preload the full YAML into a 64k operator context.

## Tool stdin/stdout

Each `tools/*.py` accepts JSON on stdin and prints one JSON object on stdout. In-process `run(*Input)` is fine when the host already has Python.

## Factory brief

`--brief` goes through `assert_safe_for_cloud` then `cloud_client.propose`. `MOCK_MODE=true` keeps the proposer local. `MOCK_MODE=false` posts only the sanitized abstract to `CLOUD_DEVELOPER_URL`. Factory output is always Python.

## Env

| Variable | Meaning |
|----------|---------|
| `WALLOP_ROOT` | Override repo root for the factory |
| `MOCK_MODE` | `true` skips sockets |
| `VAULT_PATH` | Markdown vault for `wallop graph` |
| `HARNESS_TZ` | Python tools clock |
| `TIME_OPS_LOG_PATH` | Exception log |
| `EDGE_CALENDAR_WEBHOOK` | Live calendar edge |
| `CLOUD_DEVELOPER_URL` | Live factory proposer |
