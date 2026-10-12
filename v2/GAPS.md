# Gaps vs Beads, Engram, Graphify / GitNexus

Fabrik v2 replaces the **daily agent loop** of those tools in one binary ([IDEA.md](IDEA.md)). This file is the inventory of what Fabrik **does not** copy. It is not a build backlog.

## Status legend

| Tag | Meaning |
| --- | --- |
| **skipped** | v2 explicitly said no ([IDEA.md](IDEA.md), [STACK.md](STACK.md)) |
| **absent** | Not in schema or MCP/CLI; never promised in v2 |
| **later** | Named optional in an existing v2 doc |

## Issues vs Beads (`bd`)

**Core covered:** `issue_ready` / `show` / `list` / `create` / `claim` / `close`, notes, deps `blocks` and `relates`, hash ids (`f-` / `n-`), Dolt in `.fabrik/dolt/`, `fabrik dolt push` / `pull`, JSONL export. See [issue-tools.md](issue-tools.md), [SCHEMA.sql](SCHEMA.sql).

| Source (Beads) | Status | Why |
| --- | --- | --- |
| Formulas, molecules, wisps, gates | **skipped** | Workflow chemistry not in v2 scope |
| `parent-child`, `discovered-from`, `duplicates`, `supersedes`, `replies-to` deps | **absent** | Only `blocks` \| `relates` ([issue-dep.md](issue-dep.md)) |
| Remove dependency | **absent** | No remove command in v1 |
| Labels, design field | **absent** | `acceptance` yes; no labels column |
| Message issue type, mail, threads | **absent** | Types: task, bug, feature, epic, chore only |
| Compaction of old closed issues | **absent** | No semantic decay of history |
| `bd remember`, `bd prime` as project memory | **skipped** | Decisions live in machine memory (Engram lane), not Dolt |
| defer, supersede, stale, orphans, lint, human, unclaim | **absent** | No hygiene commands |
| Dolt `sql-server` multi-writer mode | **skipped** | Single writer; MCP `store busy` |
| `bd backup`, prune, purge | **absent** | No equivalent |
| Beads import / JSONL as sync protocol | **skipped** | Dolt is source of truth; export is passive |

Keep using **Beads** when you need formulas, gates, rich dep kinds, or Beads-native hygiene. Keep using **Fabrik issues** for claim/ready/close inside the Fabrik MCP loop.

## Memory vs Engram

**Core covered:** `mem_current_project`, `mem_context`, `mem_search`, `mem_get_observation`, `mem_save`, `mem_update`, `mem_delete`, `mem_timeline`, `mem_session_start` / `summary` / `end`, `mem_save_prompt`, `mem_suggest_topic_key`, `mem_stats`; `~/.fabrik/memory.db`, FTS5; machine-wide reads unless `project` is passed. See [mem-tools.md](mem-tools.md), [MEMORY.sql](MEMORY.sql).

| Source (Engram) | Status | Why |
| --- | --- | --- |
| `mem_judge`, `mem_compare` | **skipped** | Conflict resolution not in v2 |
| `mem_merge_projects` | **skipped** | Project labels are manual / cwd-derived |
| Obsidian Brain export | **skipped** | No vault export |
| `mem_capture_passive` | **absent** | Not registered in Fabrik MCP |
| `mem_doctor` | **later** | [mem-tools.md](mem-tools.md): optional later |
| HTTP API (e.g. port 7437) | **skipped** | No Engram-compatible HTTP server; GUI uses [listen.md](listen.md) SSE |
| `<private>` secret scrub in bodies | **absent** | Not in v1 spec |
| `personal` vs `project` scope (separate tables) | **later** | `scope` reserved; v1 single observations table |

**Fabrik-only (not Engram):** `symbol_id` on observations and [graph_context](graph-context.md) joining memories to repo symbols.

Keep using **Engram** for judge/compare, merge_projects, Obsidian export, passive capture, or HTTP clients that expect `engram mcp` + HTTP.

## Code graph vs Graphify / CodeGraph / GitNexus

**Core covered:** `graph_explore`, `graph_context`, `graph_impact`, `graph_detect_changes`, `graph_path`, `graph_explain`, `graph_status`; SQLite `.fabrik/graph.db`; tree-sitter via gotreesitter; edges `extracted` \| `inferred` with candidates; incremental `graph_files` mtime refresh; languages Go, Rust, TS, TSX, JS, JSX, Python, Java, Kotlin, C#. See [graph-explore.md](graph-explore.md), [GRAPH.sql](GRAPH.sql).

| Source | Status | Why |
| --- | --- | --- |
| Cypher / KuzuDB (GitNexus) | **skipped** | Raw graph query language not in v1 |
| Embeddings, hybrid BM25 + vector search (GitNexus) | **skipped** | No vector index in v2 |
| God nodes, Leiden communities, process clusters | **absent** | No community/process tables |
| Docs, PDF, Office, image, video nodes (Graphify) | **skipped** | Code index only |
| Neo4j / FalkorDB push (Graphify) | **skipped** | Local SQLite only |
| GitNexus `rename` (multi-file coordinated) | **absent** | No rename tool |
| `route_map`, `tool_map`, `shape_check`, `api_impact` | **absent** | API/MCP route analysis not built |
| PDG / taint `explain` (GitNexus `--pdg`) | **absent** | `graph_explain` is symbol + snippet, not dataflow |
| PR triage, `list_prs`, `get_pr_impact` (Graphify / CodeGraph) | **absent** | No `gh` PR ranking in Fabrik graph |
| Multi-repo merge-graphs / global index | **absent** | One graph per repo checkout |
| `grep` / `read_file` MCP helpers inside graph server (GitNexus) | **absent** | Agent uses editor tools; graph is query-only |

Keep using **Graphify** for full-repo knowledge graphs including docs/PDFs and PR graph tools. Keep using **GitNexus** for Cypher, embeddings, PDG/taint, and coordinated rename.

## Three lanes (do not duplicate)

| Lane | Fabrik store | Do not put here |
| --- | --- | --- |
| Issues | `.fabrik/dolt/` | Decisions, session handoff |
| Memory | `~/.fabrik/memory.db` | Todos, blockers |
| Graph | `.fabrik/graph.db` | Non-code corpus, durable memory |

Cross-links only where v2 defines them: `graph_explore(issue_id=…)`, `mem_save` + `symbol_id`, `graph_context` observations for this repo path.

## When to stay on the old stack

| You need | Stay on |
| --- | --- |
| Formula pour, human/GitHub gates, rich dep graph | Beads (`bd`) |
| Memory judge, merge projects, Obsidian, passive capture | Engram |
| Docs+PDF graph, PR blast triage, god nodes | Graphify / CodeGraph |
| Cypher, embeddings, PDG, rename | GitNexus |

Use **Fabrik** when one MCP (`fabrik mcp`) should run ready → explore → save → push issues without three binaries.
