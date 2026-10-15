# fabrik mem context

## Purpose

Give an agent the recent **machine** session and the latest observations before it searches or greps. Default scope is **all projects** (machine-wide).

## Who calls it

Every agent, first. bd-neon Memory mode. Humans who want the last handoff.

## Invocation

```text
fabrik mem context [--project path] [--json]
fabrik [-C repo] mem context [--json]
```

`-C` only sets the `project` label on future writes; it does **not** filter this command unless `--project` is passed.

## Inputs

| Input | Rule |
| --- | --- |
| `--project` | Optional. Filter observations to this label (absolute repo path). |

Fixed limits: 1 session row, 20 observations.

## Stored rows

Read-only from `~/.fabrik/memory.db`. FTS is in the same file; no rebuild from Dolt.

MCP `mem_context` — same rules; optional `project` argument.

- Session: open global session (`ended_at` null), else latest ended session, else none.
- Observations: 20 newest by `created_at` across **all projects** when unfiltered.

## JSON result

```json
{
  "session": { "id": "s-m2n3o4p5", "started_at": "…", "ended_at": null, "goal": "…", "discoveries": "", "accomplished": "", "next_steps": "", "files": "" },
  "observations": [
    {
      "id": "o-w1x2y3z4",
      "session_id": "s-m2n3o4p5",
      "project": "/src/app",
      "kind": "decision",
      "title": "Hash ids",
      "body": "…",
      "created_at": "2026-10-11T18:00:00Z"
    }
  ],
  "stats": { "observation_count": 100, "session_count": 5, "projects": ["/src/app", "/other"] }
}
```

## Text result

Same as before; optional `project:` line per observation in text mode.

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Unknown `--project` | 1 | `fabrik: project not found in memory` + lists known labels |

Missing `memory.db` is created empty on first use; session null, observations `[]`.

## Examples

```text
fabrik mem context
fabrik mem context --project /src/app --json
```

## bd-neon

Memory mode on open/refresh. Search via [mem-search.md](mem-search.md).
