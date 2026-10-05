# Agent (`core/`)

Go module lives in `core/`. One binary serves API and UI.

```
core/
  cmd/brainer/          # main
  internal/api/         # HTTP JSON
  internal/memory/      # YAML store
  internal/gpt/         # Hooshyar / OpenAI-compatible client
  internal/enrich/      # prompt + map onto existing task
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
| `HOOSHYAR_BASE_URL` | default `http://api.hooshyar.systemgroup.net/abramad/gpt/v1` |
| `HOOSHYAR_TOKEN` | bearer token |
| `HOOSHYAR_MODEL` | model id (default `/gpt-120`) |
| `MEMORY_DIR` | override store path |
| `HTTP_ADDR` | default `:8080` |

## GPT

OpenAI-compatible `POST {base}/chat/completions` with `Authorization: Bearer {token}`.

Enrichment asks for JSON only: title, requester, due_at, priority, context, open_questions, needs_split, split_candidates, structured.

Do not invent due dates. Copy what the source said, or leave null.

If the call fails, keep the file, set `enrichment: failed`.

## HTTP

| Route | Role |
|-------|------|
| `POST /api/captures` | `{ "raw": "..." }` → write YAML, start enrich, return task |
| `GET /api/tasks` | list |
| `GET /api/tasks/{id}` | detail |
| `POST /api/tasks/{id}/enrich` | retry GPT enrich on an existing task |
| `GET /` | UI |
| `GET /static/` | UI assets |

`POST /api/captures` returns **201** once the file is on disk, even if enrichment is still `pending`.
