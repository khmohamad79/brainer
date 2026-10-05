# Idea changelog

## 2026-10-05 — Session context file

- Added `idea/CONTEXT.md` as the handoff primer for future chats (`@idea/CONTEXT.md`).

## 2026-10-05 — Rename GPT env vars

- Env keys are `GPT_BASE_URL`, `GPT_TOKEN`, `GPT_MODEL` (no provider prefix).
- `.env.example` leaves `GPT_BASE_URL` empty; do not commit provider URLs there.

## 2026-10-05 — Hooshyar model fix

- Working model id from `/models` is `/gpt-120` (not `gpt-4o-mini`).
- Persist `enrichment_error` on failure; UI no longer shows “Waiting on GPT…” after failure.
- Added `POST /api/tasks/{id}/enrich` retry.
- Server must be restarted after `.env` changes.

## 2026-10-05 — v0 capture agent

- Capture channel: simple web UI.
- Human-in-the-loop: auto-save; edit/split later.
- Granularity: one message = one file; mark split candidates only.
- Store: folder `memory/`, one YAML file per task, no DB.
- Success metric: never lose an assignment (raw-first write).
- Process: `idea/` first; reform core docs before code if a feature breaks an invariant.
- GPT: Hooshyar `http://api.hooshyar.systemgroup.net/abramad/gpt/v1` via `.env` token.
- UI package: `core/web`, served by the Go binary.
- `memory/` and `.env` are gitignored.
