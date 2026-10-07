# Idea changelog

## 2026-10-07 — Editable task relations

- Capture detail: teams, people, and story are badge chips (× to remove, + opens a selector of existing roster/story entries).
- `PATCH /api/tasks/{id}` updates `related_teams`, `related_employees`, and/or `story_id` (unknown ids dropped; story at most one).

## 2026-10-07 — Stories + Jira summary

- New roster `memory/stories.yaml`: id, title, optional `jira_key`, editable `summary`.
- Tasks have at most one `story_id` (empty = none). GPT fill picks it from the story roster (title / jira key); server drops unknowns.
- Stories UI at `/teams`-like `/stories` (sidebar list, add title, edit key + summary).
- Optional Jira pull: `JIRA_BASE_URL` + `JIRA_TOKEN` (PAT Bearer). Setting/changing key or refresh seeds `summary` from the issue; summary stays manually editable.
- Without Jira env, story CRUD still works; only refresh fails.

## 2026-10-07 — Teams badge editor

- New team: name only.
- Team nicknames, people, and person nicknames are badge chips with + / remove (no comma textboxes).

## 2026-10-07 — Minimal dark UI

- Dark theme only; modern sans typeface.
- Chrome: top bar (page links + Memorizing status), right sidebar (Inbox / Archive), center (new box / detail).
- `/archive` opens Capture with the Archive list selected.
- Drop redundant ledes, placeholders, and hint copy.

## 2026-10-07 — Archive + delete

- Tasks have `archived` (bool), separate from `status`. Archive from any status; status is left alone.
- Inbox lists non-archived tasks only. Archive page lists archived tasks and can delete them (removes the YAML file).
- APIs: `GET /api/tasks?archived=0|1`, `POST /api/tasks/{id}/archive`, `DELETE /api/tasks/{id}` (archived only).

## 2026-10-07 — No title; fill in task language

- Tasks have no `title` field. List/detail use the first line of `raw` (or id) as the label.
- GPT fill writes free-text fields (`requester`, `open_questions`) in the same language as `raw`.

## 2026-10-07 — Simpler tasks

- Task id is a UTC timestamp only (collision suffix `_2`, `_3`).
- GPT step renamed from enrichment to **fill** (`fill`, `fill_error`, `POST /api/tasks/{id}/fill`).
- Dropped fields: `due_at`, `priority`, `context`, `structured`, `needs_split`, `split_candidates`.
- No split marking. One message stays one file with fewer filled fields (requester, open questions, related teams/people).

## 2026-10-07 — Task ↔ org relations

- Fill links each task to related team and employee ids from `memory/org.yaml` (mentions only; no assignee role).
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
- Human-in-the-loop: auto-save; edit later.
- Granularity: one message = one file.
- Store: folder `memory/`, one YAML file per task, no DB.
- Success metric: never lose an assignment (raw-first write).
- Process: `idea/` first; reform core docs before code if a feature breaks an invariant.
- GPT: Hooshyar `http://api.hooshyar.systemgroup.net/abramad/gpt/v1` via `.env` token.
- UI package: `core/web`, served by the Go binary.
- `memory/` and `.env` are gitignored.
