# Memory

Store: `{repo}/memory/tasks/{id}.yaml` for assignments, `{repo}/memory/org.yaml` for the company roster.  
`memory/` is gitignored.

## File schema

```yaml
id: 20261005T173200Z
status: inbox          # inbox | active | done
archived: false        # independent of status; true → leave inbox
created_at: 2026-10-05T17:32:00Z
updated_at: 2026-10-05T17:32:00Z
source: web
fill: pending          # pending | ok | failed
requester: ""
open_questions: []
related_teams: []      # team ids from org.yaml (related mentions)
related_employees: []  # employee ids from org.yaml (related mentions)
raw: |
  original message
```

## ID and filename

UTC timestamp only: `20060102T150405Z`, with optional `_2`, `_3` on collision within the same second.

GPT must not be required to name the file. UI labels use the first line of `raw` (or the id).

## Write order

1. Create `memory/tasks/` if needed.
2. Atomic write of the YAML (`tmp` + `fsync` + rename) with full `raw` and `fill: pending`.
3. Fill in place. **Never overwrite `raw`.**

## List order

Newest `created_at` first. Inbox = `archived: false`. Archive list = `archived: true`.

## Archive and delete

Archive sets `archived: true` without changing `status`. Delete removes the YAML file and is only allowed when `archived: true`.

## Org roster

Official name plus optional nicknames. One YAML file, rewritten atomically (`tmp` + `fsync` + rename).

```yaml
teams:
  - id: platform
    name: Platform
    nicknames: [plat, infra]
    employees:
      - id: sara
        name: Sara
        nicknames: [sari]
```

Team and employee ids are slugs from the official name at create time (collision suffix `_2`, `_3`) and stay stable if the name is edited later. Nicknames are extra strings only. Roster must not be mixed into task files.

## Task ↔ org relations

Fill may set `related_teams` and `related_employees` to roster ids named or implied in `raw` (including nicknames). Mentions only — no assignee or other role. Unknown ids are discarded. Empty roster → leave both arrays empty.
