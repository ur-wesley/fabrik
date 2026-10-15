# fabrik loop

## Purpose

Drive plan or build waves by reading Dolt in-process. Issue rules match [issue-ready.md](issue-ready.md) and [issue-claim.md](issue-claim.md). Each claim and any agent-driven close create Dolt commits like the CLI.

## Invocation

```text
fabrik loop plan [--max n]
fabrik loop build [--max n] [--push]
```

## Inputs

Requires `.fabrik/config.yaml` and `.fabrik/dolt/`. Does not require `bd` on PATH.

Recommended before a shared build wave: `fabrik dolt pull`. Recommended after: `fabrik dolt push` (loop does not push unless `--push` on git code branch only — issue sync stays manual in v1).

## Text result

```text
claim f-m3n4o5p6  Wire the build loop
```

```text
f-m3n4o5p6 still active
```

## bd-neon

Board refreshes when `.fabrik/dolt/` changes on disk.
