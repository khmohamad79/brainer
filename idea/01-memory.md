# Memory

Store: `{repo}/memory/tasks/{id}.yaml`  
`memory/` is gitignored.

## File schema

```yaml
id: 20261005T173200Z_fix-login-redirect
status: inbox          # inbox | active | done
created_at: 2026-10-05T17:32:00Z
updated_at: 2026-10-05T17:32:00Z
source: web
enrichment: pending    # pending | ok | failed
needs_split: false
split_candidates: []   # [{title, excerpt}]
title: ""
requester: ""
due_at: null           # string as stated in source; do not invent a calendar date
priority: null         # string as stated, or null
context: []
open_questions: []
raw: |
  original message
structured: ""         # GPT summary; never a substitute for raw
```

## ID and filename

`{UTC timestamp}_{slug}` with optional `_2`, `_3` on collision.

Slug comes from the first line of `raw` (lowercase, safe chars, max 40). GPT must not be required to name the file.

## Write order

1. Create `memory/tasks/` if needed.
2. Atomic write of the YAML (`tmp` + `fsync` + rename) with full `raw` and `enrichment: pending`.
3. Enrich in place. **Never overwrite `raw`.** If title is still empty after failure, use the first line of `raw`.

## List order

Newest `created_at` first.
