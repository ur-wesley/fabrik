# Spec: Fabrik v2 native stack

APPROVED. User: `/orchestrator start building this after laying out everything as todo until everything is implemented from the docs dir`.

Authoritative behaviour lives in `v2/*.md` and `v2/*.sql`. This spec is the land + build order. Do not invent features beyond those files plus the gaps listed here.

## Goal

Replace external `bd`, Engram, and Graphify with one Go `fabrik` binary: issues in repo Dolt, machine-wide SQLite memory, repo SQLite code graph, stdio MCP, loopback SSE for the GUI. Implement every command and tool specified under `v2/` until tests match the docs.

## Architecture

- One Go module: `cli/`. `fabrik mcp` is a subcommand, not a second process language.
- Memory: `~/.fabrik/memory.db` (`FABRIK_MEMORY_DIR`), [v2/MEMORY.sql](../../v2/MEMORY.sql). Not Dolt. Not synced.
- Issues: `.fabrik/dolt/` embedded Dolt, [v2/SCHEMA.sql](../../v2/SCHEMA.sql). `fabrik dolt push` issues only.
- Graph: `.fabrik/graph.db`. gotreesitter with **all** grammar blobs; v1 parses Go, Rust, TS, TSX, JS, JSX, Python only.
- SQL: `cli/internal/sqlmigrate/` applies numbered raw `.sql` files. Do not reuse `cli/internal/migrater` (that package deletes hub markdown).
- GUI: later `gui/` (today bd-neon). CLI JSON only. Subscribe via `fabrik listen`.
- Agents/skills: stay embedded templates. `fabrik show`.

## Gaps vs current `v2/` files (write these docs first)

1. Branch `v2` with the existing untracked `v2/` tree.
2. Patch [v2/STACK.md](../../v2/STACK.md): official MCP Go SDK, all grammars, sqlmigrate, monorepo map.
3. Add [v2/REPO.md](../../v2/REPO.md), [v2/MIGRATE.md](../../v2/MIGRATE.md), [v2/listen.md](../../v2/listen.md).
4. Point neon board/memory at subscribe-first refresh. `fabrik check` prints schema versions.

`fabrik listen`: bind `127.0.0.1`, SSE `/events`, bearer token, `~/.fabrik/listen.json`, topics `issues`|`memory`|`graph`. Writes notify best-effort. See plan `stack_recommendation`.

Do not add: secret scrub, dolt unlock, orient tool, issue undo, Beads import, embeddings, Cypher, Dolt server, wrapping other CLIs.

## Quality

Per slice: `cd cli && go test ./... && go vet ./... && CGO_ENABLED=1 go build ./cmd/fabrik`.
Tests as in the stack plan: migrator, CLI goldens, MCP default tools, graph fixtures, listen notify.

No git commit unless the user later asks. Conservative Beads profile: no `dolt push` / `git push`.

## Proposed Beads issues

Epic: `v2 native stack` (type epic, p1).

All tasks `-t task -p 1` unless noted. Acceptance: match the named `v2/` files; tests listed; quality gates pass.

1. `v2: branch + remaining spec docs` — create git branch `v2`; keep `v2/` files; write REPO.md, MIGRATE.md, listen.md; patch STACK, README, check, neon-*. **blocks all below.**
2. `v2: sqlmigrate` — `cli/internal/sqlmigrate` + `sql/{dolt,memory,graph}/001_*.sql` from SCHEMA.sql / MEMORY.sql / graph tables. Tests: empty apply, idempotent, newer-than-binary error, SQLite bak restore. Deps: 1.
3. `v2: store ids + open` — hash ids, paths (`-C`, `FABRIK_MEMORY_DIR`), open Dolt/SQLite with migrations. Deps: 2.
4. `v2: init setup check reset` — implement [v2/init.md](../../v2/init.md) [v2/setup.md](../../v2/setup.md) [v2/check.md](../../v2/check.md) [v2/reset.md](../../v2/reset.md). MCP registration replaces `engram setup`. Deps: 3.
5. `v2: issue CLI` — list, ready, show, create, claim, close, dep, note per `v2/issue-*.md`. DOLT_COMMIT + export. Deps: 4.
6. `v2: mem CLI` — context, search, get, save, session start/summary/end per `v2/mem-*.md`. Machine DB. Deps: 4.
7. `v2: vc dolt export history` — vc-*, dolt push/pull, export, history per docs. Deps: 5.
8. `v2: graph index + CLI` — build, status, query, path, explain, explore, context, impact, detect-changes. All grammars linked; parse six langs. Deps: 4.
9. `v2: mcp server` — `fabrik mcp`, official go-sdk, default tool list + FABRIK_MCP_TOOLS. Deps: 5, 6, 8.
10. `v2: listen SSE` — [v2/listen.md](../../v2/listen.md); notify after successful writes. Deps: 5, 6, 8.
11. `v2: hub prompts + installer` — [v2/hub-prompts.md](../../v2/hub-prompts.md), stop `engram setup`, `tools.fabrik_mcp`. Deps: 9.
12. `v2: loop` — [v2/loop.md](../../v2/loop.md) on native stores. Deps: 5, 6.
13. `gui/` scaffold (p3) — empty `gui/` + README pointing at neon specs. Do **not** copy bd-neon yet. Deps: 10.

Land with `bd create --acceptance` and `bd dep add`. Builder takes one ready issue at a time until none remain (except the p3 gui scaffold may stay if timeboxed; still land it).
