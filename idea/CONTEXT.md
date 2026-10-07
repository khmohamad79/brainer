# Brainer — session context

Feed this file (and optionally `idea/00-core.md`) into new chats so work continues without re-deriving the design.

## What this is

Personal **assignment-capture** agent for a tech employee who gets unstructured tasks all day.

Success metric: **never lose an assignment.** Persist raw text first; GPT structure second.

Not yet: prioritization, calendar, auto-splitting into many files, multi-channel Slack ingest.

## Locked decisions (v0)

| Topic | Decision |
|-------|----------|
| Input | Simple web UI |
| Save | Auto-save on submit (no confirm) |
| Granularity | 1 message = 1 YAML file; only **mark** `needs_split` + `split_candidates` |
| Store | No DB — `memory/tasks/{id}.yaml` + `memory/org.yaml` (gitignored) |
| Format | YAML |
| UI | Separate package under `core/web`, same Go binary (no SPA); Capture + Teams pages |
| GPT | OpenAI-compatible `POST {GPT_BASE_URL}/chat/completions` |
| Env names | `GPT_BASE_URL`, `GPT_TOKEN`, `GPT_MODEL` (no provider prefix) |
| `.env.example` | No real API URL committed |

Working model id on the current provider: `/gpt-120` (not `gpt-4o-mini`).

## Invariants

1. Write `raw` to disk **before** GPT; keep it if GPT fails.
2. One message → one file.
3. Auto-save.
4. Split is a mark only (no auto child files).
5. `memory/` is source of truth and gitignored.
6. **Idea before code:** new feature → check `idea/` → if it breaks an invariant, reform idea docs + `CHANGELOG.md` → then change `core/`.

## Pipeline

```
UI submit → write YAML (enrichment: pending) → return id
         → GPT enrich (raw + org roster) → ok | failed (+ enrichment_error)
```

## Repo map

```
idea/          living design (read first)
core/          Go module
  cmd/brainer/
  internal/{api,config,enrich,gpt,memory}/
  web/         embedded HTML/CSS/JS
memory/        runtime store (gitignored): tasks/ + org.yaml
.env           secrets (gitignored)
README.md      human overview
```

## Run

```bash
cp .env.example .env   # set GPT_BASE_URL, GPT_TOKEN
cd core && go run ./cmd/brainer
# http://127.0.0.1:8080
```

Restart after any `.env` change. `.env` overrides empty/stale shell env for these keys.

## API

- `POST /api/captures` `{ "raw": "..." }` → 201 once file exists
- `GET /api/tasks`, `GET /api/tasks/{id}`
- `POST /api/tasks/{id}/enrich` → retry
- `GET /api/org` — teams and employees
- `POST /api/teams` `{ "name": "...", "nicknames": ["..."] }`
- `PATCH /api/teams/{id}` `{ "name": "...", "nicknames": ["..."] }`
- `DELETE /api/teams/{id}`
- `POST /api/teams/{id}/employees` `{ "name": "...", "nicknames": ["..."] }`
- `PATCH /api/teams/{id}/employees/{eid}` `{ "name": "...", "nicknames": ["..."] }`
- `DELETE /api/teams/{id}/employees/{eid}`

## Task YAML (essentials)

`id`, `status` (inbox|active|done), `enrichment` (pending|ok|failed), `enrichment_error`, `needs_split`, `split_candidates`, `title`, `requester`, `due_at` (as stated, never invent), `priority`, `context`, `open_questions`, `related_teams`, `related_employees`, `raw` (immutable), `structured`.

## How to continue in a new session

1. Read this file + `idea/00-core.md`.
2. For a feature: break-check against invariants; update `idea/` + `CHANGELOG.md` first if needed.
3. Implement only in `core/` (and docs).
4. Keep scope: capture reliability over new product surfaces.

## Intentionally deferred

Auth, boards, auto-split into files, Slack/email ingest, scheduling, multi-agent orchestration, task assignee roles (related mentions only for now).
