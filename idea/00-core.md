# Core

## Invariants

1. **Never lose an assignment.** Persist `raw` to a file under `memory/` before (and even if) GPT fails.
2. **One message = one file.** Do not auto-create extra task files.
3. **Auto-save.** Submit writes immediately. No confirm step.
4. **Split is a mark.** If a message contains multiple work items, set `needs_split` and `split_candidates`. Splitting into files is a later feature.
5. **`memory/` is the source of truth.** YAML files on disk. No database.
6. **`memory/` is gitignored.** Assignments stay local.
7. **Idea before code.** New feature → read `idea/` → break check → reform these docs → then change `core/`.

## Pipeline

```
Web UI submit
  → write YAML (raw, enrichment: pending)   # durable
  → return task id to UI
  → GPT enrich (same file)
  → enrichment: ok | failed
```

The HTTP response for capture may return as soon as the file exists. Enrichment can finish after that.

## Process

1. Discuss the feature.
2. Check invariants and related `idea/` notes.
3. If it breaks an invariant, reform the idea docs first and append `CHANGELOG.md`.
4. Implement in `core/`.
