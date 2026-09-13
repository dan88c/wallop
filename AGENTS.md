# Agent entry (recommended, not required)

Wallop is a local toolbox plus optional CLI. You already have a model loop.

Skill protocol **1.12**, project release **1.1.000**.

**Do not read `skills/wallop/SKILL.md` on every startup.** Ask once at cold start (or when cwd is a wallop checkout) unless the user already named wallop / toc / guard / register / doctor / factory. After yes, read the skill, then run `wallop toc`. Do not guess catalog names. Load `skills/wallop/references/protocol.md` only for flags, exits, doctor, or factory retry.

`calendar_gateway` is **demo-only**. It exists so bootstrap/tests can exercise the pipe. Do not treat it as a real calendar client; do not point it at a production webhook unless the user explicitly asks for a demo against their edge.

Recommended conventions

- Prefer a binary from GitHub Releases via `python3 scripts/bootstrap.py --download` (or `make download` / `make build`). Git `dist/` is `.gitkeep` only. Do not require the user to install Go.
- Prefer `wallop toc` over dumping the YAML or reading `tools/*.py`. Use `--full` after a guard miss.
- Prefer `wallop guard` before `tools/<name>.py`. Direct imports stay allowed.
- After adding or registering a tool, run `wallop doctor` or `python scripts/doctor.py`.
- Do not invent tool names.
- If a tool is missing, ask whether to `wallop register --entry ... --name ...`, run the factory, or leave it outside the repo. Do not write `tools/` or the YAML without a yes.
- Factory pytest failure does not update the YAML. Retry the factory at most twice, then stop.
- Sanitize with `developer-factory/privacy_guard.py` before any cloud developer call.
- Keep `MOCK_MODE=true` unless the user enables a live edge webhook.
