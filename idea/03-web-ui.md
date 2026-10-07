# Web UI (`core/web`)

Separate package under `core/web`, same binary. No second service. No SPA framework for v0.

## Screens

1. **Capture** (`/`) — center is a simple new-assignment box, or a selected task’s detail. Right sidebar switches between **Inbox** (non-archived) and **Archive**. Detail can archive (inbox) or delete (archive).
2. **Teams** (`/teams`) — same chrome; right sidebar lists teams. New team asks for name only. Team/person nicknames and people are badge chips (+ to add, × to remove). Official names auto-save after a short pause.
3. **Archive** (`/archive`) — redirects to Capture with the Archive sidebar selected (`/?list=archive`).

## Chrome

- **Top bar** on every page: page links and centered status. Status is empty when everything is on disk; while a write is waiting or in flight it shows `Memorizing ...`.
- **Right sidebar**: Inbox / Archive (on Capture) or team list (on Teams).
- **Center**: primary content — new box, task detail, or team editor.

## Look

- Dark theme only. Minimal chrome: no marketing copy, no redundant labels/placeholders/hints.
- Modern sans typeface (not serif / “document” look).

## Rules

- Submit must not wait on GPT to feel “saved”. Show the new task immediately.
- Failed fill is visible; raw is always shown.
- No auth, boards, or calendar in v0.
