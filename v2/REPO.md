# Monorepo layout

| Path | Role |
| --- | --- |
| `cli/` | Go module. `fabrik` binary: setup, init, issue, mem, graph, mcp, listen, loop. |
| `v2/` | Behaviour spec until code matches. |
| `gui/` | Rust egui human UI (bd-neon port). CLI JSON + optional SSE subscribe. |
| `testdata/` | Shared JSON goldens for CLI, MCP, GUI decode tests. |
| `install/templates/` | Shims copied on init. |
| `cli/internal/tmpl/` | Embedded hub agents, skills, prompts (`fabrik show`). |

MCP is not a separate repo or binary: `fabrik mcp` in `cli/cmd/fabrik`.

Agents and skills ship inside the CLI. No npm install for hub content.
