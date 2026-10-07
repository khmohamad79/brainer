# Idea changelog

## 2026-10-07 — Task ↔ org relations

- Enrichment links each task to related team and employee ids from `memory/org.yaml` (mentions only; no assignee role).
- New task fields: `related_teams`, `related_employees` (arrays of roster ids).
- GPT receives the roster (names + nicknames); server drops unknown ids. Requester stays free text.

## 2026-10-06 — Memory status bar

- Every page has a sticky top bar. Centered `Memorizing ...` while anything is not yet written to `memory/`.

## 2026-10-06 — Editable roster names

- Team and employee official names are editable; same delayed autosave as nicknames.
- Record ids stay the slug from create time.

## 2026-10-06 — Roster nickname autosave

- Nickname textboxes auto-save after a short idle delay. No Save button on those fields.

## 2026-10-06 — Roster nicknames

- Teams and employees may have zero or more nicknames (strings only).
- Nicknames live on the same `memory/org.yaml` records; ids still come from the official name.

## 2026-10-06 — Company roster (teams + employees)

- New UI page `/teams` to define teams and employees by name only.
- Roster lives in `memory/org.yaml` (same gitignored store as tasks; no DB).
- Capture pipeline is unchanged: tasks stay one YAML file each under `memory/tasks/`.
- No emails, roles, or assignment-to-person yet.

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
