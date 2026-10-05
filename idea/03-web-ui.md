# Web UI (`core/web`)

Separate package under `core/web`, same binary. No second service. No SPA framework for v0.

## Screens

1. **Capture** — textarea + Record. Auto-save on submit.
2. **List** — recent tasks: title (or first line), status, `needs_split` badge, enrichment warning.
3. **Detail** — raw + structured fields. Poll while `enrichment: pending`.

## Rules

- Submit must not wait on GPT to feel “saved”. Show the new task immediately.
- Failed enrichment is visible; raw is always shown.
- No auth, boards, or calendar in v0.
