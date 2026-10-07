# Brainer

Personal assignment-capture agent for messy work requests.

Paste Slack pings, meeting dumps, or verbal notes. Brainer **saves the raw text to disk first**, then asks GPT to fill a few fields. The goal is simple: **never lose an assignment**.

## Idea

Tasks arrive unstructured. Brainer is not a full project manager yet. It is a capture agent:

1. You paste one message in a web UI.
2. One YAML file is written under `memory/tasks/` immediately (auto-save).
3. GPT fills the same file (requester, open questions — in the task’s language) and links related teams/people from `memory/org.yaml` when names or nicknames match.

Living design notes live in [`idea/`](idea/). For a new AI/chat session, start with [`idea/CONTEXT.md`](idea/CONTEXT.md). Before any new feature:

1. Read `idea/` and check invariants.
2. If the feature breaks an invariant, reform the idea docs and append [`idea/CHANGELOG.md`](idea/CHANGELOG.md).
3. Then change `core/`.

## Structure

```
brainer/
├── idea/                 # concepts, invariants, decision log
├── core/                 # Go agent + embedded web UI
│   ├── cmd/brainer/      # binary entrypoint
│   ├── internal/
│   │   ├── api/          # HTTP routes
│   │   ├── config/       # .env loader
│   │   ├── fill/         # GPT field fill
│   │   ├── gpt/          # OpenAI-compatible GPT client
│   │   └── memory/       # YAML file store
│   └── web/              # UI (HTML/CSS/JS), served by the same binary
├── memory/               # runtime store (gitignored): tasks + org.yaml
├── .env                  # secrets + config (gitignored)
└── .env.example
```

Each task is one YAML file, e.g. `memory/tasks/20261005T141409Z.yaml`. Fill may set `related_teams` and `related_employees` to roster ids from `memory/org.yaml` (related mentions only).

## Requirements

- Go 1.23+
- GPT API token (OpenAI-compatible endpoint)

## Setup

```bash
cp .env.example .env
# edit .env: set GPT_BASE_URL and GPT_TOKEN
```

Relevant env vars:

| Key | Default | Purpose |
|-----|---------|---------|
| `GPT_BASE_URL` | _(set in `.env`)_ | OpenAI-compatible API base |
| `GPT_TOKEN` | _(required)_ | Bearer token |
| `GPT_MODEL` | `/gpt-120` | Model id |
| `MEMORY_DIR` | `memory` | Task store root |
| `HTTP_ADDR` | `:8080` | Listen address |

Restart the server after changing `.env`.

## Run

From the repo root:

```bash
cd core
go run ./cmd/brainer
```

Open [http://127.0.0.1:8080](http://127.0.0.1:8080). Teams and employees: [http://127.0.0.1:8080/teams](http://127.0.0.1:8080/teams).

Paste an assignment → **Record**. The task appears in the inbox even if GPT fails; use **Retry fill** on the detail pane if needed.

## API (v0)

| Method | Path | Role |
|--------|------|------|
| `POST` | `/api/captures` | Save raw text, start fill |
| `GET` | `/api/tasks` | List tasks (`?archived=0` default, `?archived=1` for archive) |
| `GET` | `/api/tasks/{id}` | Task detail |
| `POST` | `/api/tasks/{id}/fill` | Retry fill |
| `POST` | `/api/tasks/{id}/archive` | Move to archive |
| `DELETE` | `/api/tasks/{id}` | Delete archived task |
| `GET` | `/api/org` | Teams and employees |
| `POST` | `/api/teams` | Create team (name, optional nicknames) |
| `PATCH` | `/api/teams/{id}` | Update team name and/or nicknames |
| `DELETE` | `/api/teams/{id}` | Remove team |
| `POST` | `/api/teams/{id}/employees` | Add employee (name, optional nicknames) |
| `PATCH` | `/api/teams/{id}/employees/{eid}` | Update employee name and/or nicknames |
| `DELETE` | `/api/teams/{id}/employees/{eid}` | Remove employee |

## Invariants (short)

- Raw is written before GPT runs.
- One message → one file.
- Auto-save; no confirm gate.
- `memory/` is the source of truth and is gitignored.
