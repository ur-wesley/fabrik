# fabrik mem get

## Purpose

Print one observation in full.

## Invocation

```text
fabrik mem get <id> [--json]
```

## Inputs

| Input | Rule |
| --- | --- |
| `id` | `^o-[0-9a-f]+$` (hash id). |

## Stored rows

Read-only from `memory.db`.

## JSON result

Observation object including `project`, optional `topic_key`, `symbol_id`.

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Unknown id | 1 | `fabrik: observation o-… not found` |

## Examples

```text
fabrik mem get o-w1x2y3z4 --json
```

## bd-neon

Detail pane for selected row.
