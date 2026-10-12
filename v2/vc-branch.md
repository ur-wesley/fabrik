# fabrik vc branch

## Purpose

List, create, or switch Dolt branches. Independent of git branches.

## Who calls it

Humans running experiments on issue data. Not used in the default agent loop.

## Invocation

```text
fabrik [-C repo] vc branch [--json]
fabrik [-C repo] vc branch create <name>
fabrik [-C repo] vc checkout <name>
```

`create` makes a branch from current HEAD. `checkout` switches the working branch.

## Inputs

Branch names: `[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}`. `main` is the default branch after init.

## Stored rows

`DOLT_BRANCH`, `DOLT_CHECKOUT` via SQL procedures.

## JSON result

List mode:

```json
{
  "current": "main",
  "branches": ["main", "experiment"]
}
```

Create/checkout return `{ "branch": "experiment" }`.

## Text result

```text
* main
  experiment
```

## Errors

Unknown branch on checkout: `fabrik: branch experiment not found`

## Examples

```text
fabrik vc branch
fabrik vc branch create experiment
fabrik vc checkout experiment
```

## bd-neon

Not shown in v1.
