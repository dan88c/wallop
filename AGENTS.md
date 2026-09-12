# Agent entry (recommended, not required)

Wallop is a local toolbox plus optional CLI. You already have a model loop.

**On startup and on any unknown tool request, read `skills/wallop/SKILL.md` first** (or the symlink at `~/.agents/skills/wallop`). Then run `wallop toc`. Do not guess catalog names.

Load `skills/wallop/SKILL.md` for vault-graph, registry, factory, `wallop register`, or "put my existing script in this repo" requests.

`calendar_gateway` is **demo-only**. It exists so bootstrap/tests can exercise the pipe. Do not treat it as a real calendar client; do not point it at a production webhook unless the user explicitly asks for a demo against their edge.

Recommended conventions

- Prefer a binary from `dist/` (or GitHub Releases). Do not require the user to install Go.
- Prefer `wallop toc` over dumping the YAML. Use `--full` after a guard miss.
- Prefer `wallop guard` before `tools/<name>.py`. Direct imports stay allowed.
- Do not invent tool names.
- If a tool is missing, ask whether to `wallop register --entry ... --name ...`, run the factory, or leave it outside the repo. Do not write `tools/` or the YAML without a yes.
- Sanitize with `developer-factory/privacy_guard.py` before any cloud developer call.
- Keep `MOCK_MODE=true` unless the user enables a live edge webhook.
