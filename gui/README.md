# neon

Fabrik human UI: Kanban, memory, and code lookup. Spec: [v2/neon-board.md](../v2/neon-board.md), [v2/neon-memory.md](../v2/neon-memory.md), [v2/neon-lookup.md](../v2/neon-lookup.md).

Queries via `fabrik … --json`. Refresh via `fabrik listen` SSE (`~/.fabrik/listen.json`). Neon starts listen if needed.

```bash
cargo run --manifest-path gui/Cargo.toml -- /path/to/repo
```

From `gui/`, `cargo dev` runs watch + `cargo run` (aliases in `.cargo/config.toml`). Install `cargo-watch` via `mise install` at the repo root.

Requires `fabrik` on PATH (or `FABRIK_BIN`). Issue store needs `CGO_ENABLED=1` build of the CLI.
