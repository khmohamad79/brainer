# Web UI (`core/web`)

Separate package under `core/web`, same binary. No second service. No SPA framework for v0.

## Screens

1. **Capture** (`/`) — textarea + Record. Auto-save on submit.
2. **List** — recent tasks: title (or first line), status, `needs_split` badge, enrichment warning.
3. **Detail** — raw + structured fields, including related teams/people from enrichment. Poll while `enrichment: pending`.
4. **Teams** (`/teams`) — define team and employee names plus optional nicknames; persist to `memory/org.yaml`. Name and nickname edits auto-save after a short pause (no Save button).

## Chrome

A sticky top bar on every page. Centered text is empty when everything is on disk. While a write is waiting or in flight it shows `Memorizing ...`.

## Rules

- Submit must not wait on GPT to feel “saved”. Show the new task immediately.
- Failed enrichment is visible; raw is always shown.
- No auth, boards, or calendar in v0.
