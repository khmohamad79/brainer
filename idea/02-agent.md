# Agent (`core/`)

Go module lives in `core/`. One binary serves API and UI.

```
core/
  cmd/brainer/          # main
  internal/api/         # HTTP JSON
  internal/memory/      # YAML store
  internal/gpt/         # OpenAI-compatible GPT client
  internal/jira/        # Jira REST (PAT Bearer)
  internal/fill/        # prompt + map onto existing task
  internal/config/      # .env
  web/                  # separate UI package, embedded
```

## Run

From repo root:

```bash
cd core && go run ./cmd/brainer
```

`config` loads `{repo}/.env` and stores tasks in `{repo}/memory`.

## Environment

| Key | Purpose |
|-----|---------|
| `GPT_BASE_URL` | OpenAI-compatible API base (required in `.env`; has a built-in default if empty) |
| `GPT_TOKEN` | bearer token |
| `GPT_MODEL` | model id (default `/gpt-120`) |
| `JIRA_BASE_URL` | Jira Server/DC base (e.g. `https://jira.abramad.com`); optional |
| `JIRA_TOKEN` | Jira personal access token (Bearer); optional |
| `MEMORY_DIR` | override store path |
| `HTTP_ADDR` | default `:8080` |

## GPT

OpenAI-compatible `POST {base}/chat/completions` with `Authorization: Bearer {token}`.

Fill asks for JSON only: requester, open_questions, related_teams, related_employees, story_id.

Free-text fields (`requester`, `open_questions`) must be in the same language as the assignment `raw`.

Before calling GPT, load `org.yaml` and `stories.yaml` and include a compact roster (team/employee ids, names, nicknames; story ids, titles, jira keys) in the user message. GPT may only return those ids for `related_teams` / `related_employees` / `story_id`. Server intersects with the current rosters and drops unknowns. Empty roster → leave relations empty (`story_id` `""`).

If the call fails, keep the file, set `fill: failed`.

## HTTP

| Route | Role |
|-------|------|
| `POST /api/captures` | `{ "raw": "..." }` → write YAML, start fill, return task |
| `GET /api/tasks` | list; `?archived=0` (default) or `?archived=1` |
| `GET /api/tasks/{id}` | detail |
| `PATCH /api/tasks/{id}` | `{ related_teams?, related_employees?, story_id? }` — edit relations (unknown ids dropped) |
| `POST /api/tasks/{id}/fill` | retry GPT fill on an existing task |
| `POST /api/tasks/{id}/archive` | set `archived: true` (status unchanged) |
| `DELETE /api/tasks/{id}` | delete file; only if archived |
| `GET /api/stories` | list stories |
| `POST /api/stories` | `{ "title", "jira_key?" }` → create; fetch summary if key set |
| `PATCH /api/stories/{id}` | `{ "title?", "jira_key?", "summary?" }`; key change re-fetches summary |
| `DELETE /api/stories/{id}` | delete story |
| `POST /api/stories/{id}/jira-refresh` | re-fetch summary from current jira_key |
| `GET /` | Capture UI |
| `GET /teams` | Teams UI |
| `GET /stories` | Stories UI |
| `GET /archive` | Archive UI |
| `GET /static/` | UI assets |

`POST /api/captures` returns **201** once the file is on disk, even if fill is still `pending`.
