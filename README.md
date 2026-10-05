# Brainer

Personal assignment-capture agent for messy work requests.

Paste Slack pings, meeting dumps, or verbal notes. Brainer **saves the raw text to disk first**, then asks GPT to structure it. The goal is simple: **never lose an assignment**.

## Idea

Tasks arrive unstructured. Brainer is not a full project manager yet. It is a capture agent:

1. You paste one message in a web UI.
2. One YAML file is written under `memory/tasks/` immediately (auto-save).
3. GPT enriches the same file (title, due phrase, context, open questions).
4. If the message looks like several work items, Brainer **marks** split candidates — it does not create extra files yet.

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
│   │   ├── enrich/       # GPT structuring
│   │   ├── gpt/          # OpenAI-compatible GPT client
│   │   └── memory/       # YAML file store
│   └── web/              # UI (HTML/CSS/JS), served by the same binary
├── memory/               # runtime task store (gitignored)
├── .env                  # secrets + config (gitignored)
└── .env.example
```

Each task is one YAML file, e.g. `memory/tasks/20261005T141409Z_fix-login-bug.yaml`.

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

Open [http://127.0.0.1:8080](http://127.0.0.1:8080).

Paste an assignment → **Record**. The task appears in the inbox even if GPT fails; use **Retry enrich** on the detail pane if needed.

## API (v0)

| Method | Path | Role |
|--------|------|------|
| `POST` | `/api/captures` | Save raw text, start enrich |
| `GET` | `/api/tasks` | List tasks |
| `GET` | `/api/tasks/{id}` | Task detail |
| `POST` | `/api/tasks/{id}/enrich` | Retry enrich |

## Invariants (short)

- Raw is written before GPT runs.
- One message → one file.
- Auto-save; no confirm gate.
- Split is metadata only.
- `memory/` is the source of truth and is gitignored.
