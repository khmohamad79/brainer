# Web UI (`core/web`)

Separate package under `core/web`, same binary. No second service. No SPA framework for v0.

## Screens

1. **Capture** (`/`) — textarea + Record. Auto-save on submit. Inbox = non-archived tasks. Detail can archive a task.
2. **List** — recent non-archived tasks: first line of raw (or id), status.
3. **Detail** — full raw on top, light fields, refresh-fill icon. Poll while `fill: pending`.
4. **Archive** (`/archive`) — archived tasks only. Detail can delete (removes the file).
5. **Teams** (`/teams`) — define team and employee names plus optional nicknames; persist to `memory/org.yaml`. Name and nickname edits auto-save after a short pause (no Save button).

## Chrome

A sticky top bar on every page. Centered text is empty when everything is on disk. While a write is waiting or in flight it shows `Memorizing ...`.

## Rules

- Submit must not wait on GPT to feel “saved”. Show the new task immediately.
- Failed fill is visible; raw is always shown.
- No auth, boards, or calendar in v0.
