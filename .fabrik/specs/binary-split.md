# Spec: Binary Split — `fabrik` + `fabrik-lookup` (`fabrik-issue` only if Dolt returns)

**Issue:** `cli-wr4` · **Type:** plan-only spec · **No code, no build, no packaging changes in this issue.**

## 1. Background and context

- Embedded Dolt is **gone**. Completed work in `cli-x3a` / `cli-a35` replaced it with a
  pure-Go SQLite issue store at `.fabrik/issues.db` plus git + JSONL sync
  (`cli/internal/issuestore`, `cli/internal/export`, `fabrik sync` / `fabrik dolt`
  compatibility surface). `cli/go.mod` contains **no** `dolthub` / Dolt driver
  dependency — verify with `grep -i dolt cli/go.mod` (expect no hits outside comments).
- Binary size work in `cli-qf1` stripped the single binary from ~56.9 MiB to ~44.1 MiB
  (`-trimpath -ldflags="-s -w"`, see `scripts/build-cgo.mjs`).
- Today there is exactly **one** binary: `cli/cmd/fabrik` (package `main`), whose cobra
  root in `cli/cmd/fabrik/root.go` wires ~21 top-level commands via `new*Cmd()`
  constructors (`setup init reset check updatedeps version selfupdate show list run loop
  migrate listen mem issue graph mcp vc dolt sync history export`).
- Distribution today is single-file / single-asset:
  - `scripts/build-cgo.mjs` builds `./cmd/fabrik` (CGO optional now, pure-Go fallback
    `CGO_ENABLED=0 go build … ./cmd/fabrik`).
  - `npm/fabrik-*/bin` platform packages each ship one file (`bin/fabrik` or
    `bin/fabrik.exe`); the wrapper `bin/fabrik.js` resolves it (`FABRIK_BINARY`
    override → repo `npm/fabrik-<t>/bin` → local `cli/fabrik[.exe]` → installed
    `node_modules/@ur-wesley/fabrik-<target>/bin` → `require.resolve`).
  - `cli/internal/selfupdate/selfupdate.go` downloads **one** asset resolved by the
    single source of truth `cli/internal/platform` (`SelfUpdateAssetName(key)` →
    `fabrik-<suffix>` / `fabrik-<suffix>.exe`, fail-closed on missing `.sha256`).
  - GUI (`gui/src/fabrik/bin.rs`) and mise (`mise.toml` `gui:run`) shell
    `FABRIK_BIN` else `fabrik` from PATH; on Windows `FABRIK_BIN` must carry the
    `.exe` extension (CreateProcess semantics, cf. `cli-umi`).

This spec defines the **target** split. It does not implement it.

## 2. Goals / non-goals

Goals:

1. Separate the **agent hot path** (small, fast, sandbox-safe lookups) from the
   **admin/setup surface** (interactive setup, destructive reset, self-update).
2. Keep `fabrik mcp` working during and after the split as a **thin stdio router**
   that execs its siblings — no tool logic duplicated in the router.
3. Keep the **single-bundle install story**: one `go install`, one mise task, one npm
   meta-package, one GitHub release per platform, one `selfupdate` invocation that
   swaps every file.
4. State the `fabrik-issue` gate explicitly so nobody re-adds a third binary
   speculatively.

Non-goals (this issue): no `cmd/*` changes, no `release-cli.yml` changes, no npm
`bin/` changes, no `selfupdate.go` / `swap_*.go` changes, no `FABRIK_BIN` contract
changes, no MCP tool renames.

## 3. Target binary responsibilities

Recommended steady state: **2 binaries** (+ `fabrik mcp` as a mode, §4).

### 3.1 `fabrik` — admin / setup / mutation surface

Human-run, often interactive or destructive. Commands:

| Group | Commands | Notes |
|---|---|---|
| setup | `setup`, `init`, `reset`, `check`, `updatedeps` | wizard / template / health; `reset` destructive confirm path stays here |
| lifecycle | `version`, `selfupdate` | version via `version.Get()` (ldflags → `cli/package.json` → `"dev"`); self-update swaps all N files (§6) |
| hub | `show`, `list` | embedded hub content (`cli/internal/tmpl`) |
| execution | `run`, `loop`, `listen` | agent loop runners, long-lived processes |
| repo data (write) | `migrate`, `export`, `vc`, `sync`, `dolt` (+ `issue` **writes**: `create claim unclaim close dep note`) | `dolt` is the git-backed sync compatibility surface over the SQLite store, not embedded Dolt |
| memory (write) | `mem save/update/delete/session/*`, `mem export` | Virtual memory backends (VP/MA/UB/TI) mount under `mem` |
| graph (write) | `graph build` | index rebuild; reads move to lookup |

Rationale: everything that prompts (`ui.Confirm` / BubbleTea), mutates machine state,
touches the network, or rewrites the repo index lives in the big binary that humans
upgrade explicitly.

### 3.2 `fabrik-lookup` — agent hot path (pure Go, read-only)

Small binary agents, MCP servers, GUI panels, and hooks shell out to at high frequency.
**Read-only**, pure Go (no CGO, no network, no prompts):

| Group | Commands (read-only subset) |
|---|---|
| memory reads | `mem context`, `mem search`, `mem get`, `mem current-project`, `mem stats`, `mem timeline` |
| graph reads | `graph query`, `graph explore`, `graph overview`, `graph context`, `graph impact`, `graph explain`, `graph path`, `graph status`, `graph detect-changes` |
| issue reads (SQLite) | `issue list`, `issue ready`, `issue show` against `.fabrik/issues.db` via `cli/internal/issuestore` (read paths only; **no** `create/claim/close/dep/note`) |

Constraints for `fabrik-lookup`:

- No `os.Stdin` prompts, no `ui.Confirm`, no network dials, no self-update code linked
  in (keeps the binary small and sandbox-approvable).
- Shares `cli/internal/{memory,graph,issuestore,store,clierr,version}` with `fabrik`
  — no forks, no duplicated query logic. Size win comes from **not linking**
  `installer/tmpl/selfupdate/mcp-sdk/loop-runner` trees, not from copying code.
- JSON output contract unchanged (`--json` flags behave identically in both binaries
  so callers can swap the argv[0] only).
- Exit codes and `fabrik: usage: …` error strings unchanged (`clierr.Usage`).

### 3.3 `fabrik-issue` — GATED, do not build

> **Dolt is gone** (`cli-x3a`/`cli-a35`: SQLite issue store + git+JSONL sync;
> `go.mod` has no Dolt driver). There is therefore **no reason** to build
> `fabrik-issue` today.

Gate for ever reconsidering: **only if** embedded Dolt (or any CGO / JVM / native
issue backend) returns to the tree and its link cost would regress the lookup binary
size/latency budget (§7). If that happens, `fabrik-issue` would own `issue`
reads+writes and `dolt`/`sync` porcelain; until then the `issue` read subset stays in
`fabrik-lookup` (§3.2) and writes stay in `fabrik` (§3.1). Any proposal to add this
binary must cite the commit that reintroduced the native dependency.

## 4. `fabrik mcp` as thin stdio router

`fabrik mcp` remains the registered stdio entry point (Cursor / OpenCode / Pi /
Antigravity configs keep pointing at `fabrik mcp`) but becomes a **router**, not a
tool host:

1. **Resolution order** for siblings (first hit wins):
   1. Directory of `os.Executable()` for `fabrik` (i.e. `fabrik-lookup[.exe]`,
      `fabrik-issue[.exe]` next to the running `fabrik`) — covers release zips,
      npm `bin/` dirs, and `cli/` local builds where all files sit together.
   2. `PATH` lookup (`exec.LookPath`) — covers `go install ./cmd/...` (all binaries
      land in the same `$GOBIN`) and split installs.
   3. Else: fail-closed error naming the missing sibling and the install command.
2. **Routing table**: memory tools + graph tools + `issue_ready/show(/list)` →
   `fabrik-lookup`; `issue_create/claim/close/dep/note` + everything else → `fabrik`
   itself (in-process, no re-exec). If `fabrik-issue` ever exists, the whole `issue_*`
   family routes there instead.
3. **Version coherence**: router reports `version.Get()` (same ldflags stamp in every
   binary built from the same release); on sibling version mismatch (sibling
   `--version` != router version) log a warning to stderr but keep serving — never
   refuse to start an agent session over a patch skew; refuse only on sibling missing
   or non-executable.
4. **Filtering preserved**: `FABRIK_MCP_TOOLS` semantics from
   `cli/internal/mcpserver/server.go` (`EnabledToolNames()`: empty → `defaultTools`,
   `all` → `defaultTools + extraTools`, comma list intersected with `knownTools`)
   apply in the router **before** spawning so disabled tools are never advertised in
   `ListTools`, regardless of which sibling implements them.
5. **Transport**: stdio only. Router forwards stdin/stdout frames byte-transparent to
   the chosen sibling per request (or re-execs per request — implementation choice,
   invisible to clients); stderr of siblings merges to router stderr.

Why a mode first (§7): keeping `mcp` inside `fabrik` avoids a 4th release asset, a 4th
npm file entry, and a 4th self-update swap unit while the tool set still fits one
process.

## 5. Source layout (`cmd/*` sharing `cli/internal`)

Target (illustrative — a future implement issue may adjust names):

```text
cli/cmd/fabrik/         # admin binary (package main): root.go minus mem/graph-read/issue-read cmds
cli/cmd/fabrik-lookup/  # lookup binary (package main): cobra root with mem-read + graph-read + issue-read cmds
cli/cmd/fabrik-mcp/     # OPTIONAL 4th binary, gated (§7); else `mcp` subcommand stays in fabrik
cli/internal/...        # shared: memory graph issuestore store export version clierr platform mcpserver selfupdate …
```

Rules:

- `cli/internal` stays dependency-light and UI-free at the leaves: `memory`, `graph`,
  `issuestore`, `store` must not import `installer`, `tmpl`, `selfupdate`, or the MCP
  SDK, so `fabrik-lookup` does not link them.
- Cobra `new*Cmd()` constructors move to shared `cli/internal/cli` (or stay in
  `cmd/fabrik` and get imported — implementer's choice) so both roots wire the same
  command objects; no copy-paste of flag definitions.
- `go install ./cmd/...` installs the whole suite; `go build ./...` + `go vet ./...`
  + `go test ./...` keep covering all mains.

## 6. Install matrix (must stay single-bundle)

Every row must deliver **all** binaries together; never ship a channel that installs
only `fabrik`:

| Channel | Today (single) | After split (N files) |
|---|---|---|
| `go install` | `go install github.com/ur-wesley/fabrik/cli/cmd/fabrik@latest` | `go install github.com/ur-wesley/fabrik/cli/cmd/...@latest` (all mains in one `$GOBIN`); keep the single-path form working as admin-only fallback |
| mise | `mise run cli:build-cgo` → `cli/fabrik[.exe]`; `gui:run` sets `FABRIK_BIN` | build task emits `cli/fabrik[.exe]` + `cli/fabrik-lookup[.exe]` (+ issue/mcp if gated in); `FABRIK_BIN` keeps pointing at `fabrik`, siblings resolve via `os.Executable` dir (§4) |
| npm meta-package `@ur-wesley/fabrik` (`bin/fabrik.js`) | resolves one `bin/fabrik[.exe]` from `npm/fabrik-<target>/bin` | launcher resolves sibling names from the **same directory** as the found binary (`fabrik-lookup[.exe]`, …); one platform package per `npm/fabrik-*` still, but with N files in `files[]` + `bin[]` |
| GoReleaser / `release-cli.yml` matrix (5 targets: win-amd64, linux-amd64/arm64, darwin-amd64/arm64; no win-arm64 runner) | builds + uploads `fabrik-<suffix>[.exe]` + `.sha256`; stages into `npm/<dir>/bin` | build loop emits N binaries per target; release attaches N assets + N `.sha256` files; stage step copies all N into each `npm/<dir>/bin` |
| `fabrik selfupdate` (`selfupdate.go` + `platform.SelfUpdateAssetName`) | downloads 1 asset, fail-closed without `<asset>.sha256`, swaps via `swap_unix.go` (rename aside) / `swap_windows.go` (stage `.new`, backup `.old` by copy, fast-path rename else `MoveFileExW` `DELAY_UNTIL_REBOOT`) | **N-file swap**: resolve N asset names from the same platform key, verify N checksums **before** swapping any, then swap each with the existing per-file `swap()`; Windows locking applies per file (a locked `fabrik.exe` must not block swapping `fabrik-lookup.exe`); keep `.old` rollback per file |

`FABRIK_BIN` / `FABRIK_BINARY` contract: unchanged meanings (explicit binary path
override). GUI shells `FABRIK_BIN` for admin ops; lookup fast paths may shell the
sibling directly when already resolved, else go through `fabrik mcp` routing (§4).

## 7. Decision: 3 binaries max, MCP as mode first

- **Cap: 3 binaries** (`fabrik`, `fabrik-lookup`, reserved `fabrik-issue` slot) and
  **no 4th binary for MCP** in the first implementation: `fabrik mcp` ships as a mode
  (router, §4) inside `fabrik`.
- **Promote MCP to a 4th binary (`fabrik-mcp`) only if** sandboxing / allow-listing
  demands it — i.e. a concrete agent sandbox (macOS seatbelt, Windows AppContainer,
  corporate EDR policy) blocks an admin-capable `fabrik` from holding the stdio
  socket even when it only routes, and accepts a router binary that links **only**
  `mcpserver` + `version` + exec-shim. The promotion proposal must name the sandbox
  and attach its denial log.
- **Never more than 4**: if `fabrik-issue` is gated out (§3.3) and MCP stays a mode,
  the suite is 2 files; the matrix above must still handle N generically so a future
  gated addition is mechanical.

## 8. Acceptance criteria (for a future implement issue, not this one)

1. `go install ./cmd/...` yields all suite binaries; each `--version` prints the same
   `version.Get()`.
2. `fabrik-lookup --help` lists only the §3.2 read subset; every listed command works
   with `--json` byte-identically to today's `fabrik … --json`.
3. `fabrik mcp` serves the full `FABRIK_MCP_TOOLS` matrix (`""`, `"all"`,
   comma-list) with siblings present, and errors clearly naming the missing sibling
   when one is removed from PATH + exe-dir.
4. Single-bundle install verified per §6 row (go, mise, npm pack contents, release
   assets, selfupdate dry-run + real swap on unix and Windows-locked path).
5. `fabrik-lookup` links no network/self-update/prompt symbols (implementer defines
   the assertion, e.g. `go list` import closure check); size delta vs. today recorded.
6. No `fabrik-issue` binary exists; `grep -ri dolt cli/go.mod` still clean.

## 9. Risks / open questions

- Windows self-update locking (§6, `swap_windows.go`): N-file swap must be
  per-file atomic with per-file reboot-staging; a half-swapped suite (new lookup +
  old admin) must still interoperate (version-skew tolerance, §4.3).
- npm optional-dependency skew: a stale global `fabrik-<target>` shadowing a fresh
  meta-package could mix versions; router warns (§4.3), `selfupdate` and npm remain
  the blessed upgrade paths.
- GUI `FABRIK_BIN` on Windows must keep the `.exe` (CreateProcess appends `.exe` to
  extensionless spawns — stale-binary hijack, `cli-umi`); split changes nothing here
  but every doc/step must repeat it.
- Open question for implementer: in-process vs re-exec-per-request MCP routing
  latency on large `graph explore` payloads — measure, default to whichever keeps
  p50 unchanged.
