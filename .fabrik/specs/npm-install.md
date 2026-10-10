# Spec: Install Fabrik from npm (npm + bun, no postinstall)

## Status
Draft — for APPROVE. No Beads, no code.

## Goal
One publish path installs Fabrik everywhere:
`npm i -g @ur-wesley/fabrik`, `npx`, `bun add -g @ur-wesley/fabrik`, `bunx`,
local `npm i` / `bun add` — with **zero lifecycle scripts**, so bun's
postinstall block (`bun` ignores `postinstall` unless trusted) can never break
install. Never use a `curl|sh` postinstall download as the primary lane.

## Recon (reused, not re-run)
- Explore: no npm distribution existed at recon time. Root `package.json` had no
  `bin`, no `postinstall`, no `files`, `main: index.ts`, scripts used `bun`/`bunx`.
  `cli/package.json` (`@ur-wesley/fabrik-cli` v1.0.0, `private: true`) is a
  version-source mirror only. Dist lanes: `go install` + GH release binaries,
  `install/setup.sh|.ps1`, `pi install git:`.
- Researcher: `bin` + shebang to `./dist/index.js`, `files: [dist]`, zero lifecycle
  scripts (immune to the bun block). If the Go CLI ships: bin JS shim +
  `optionalDependencies` per-platform packages with `os`/`cpu` (esbuild/sharp
  pattern). No install script required. `curl|sh` postinstall never primary;
  bun `trustedDependencies` is consumer-side only and not a mechanism we can rely on.
- Delta since recon (already landed, commit `2839d02` + follow-ups — this spec
  ratifies + hardens, see Phase 2): root `bin/fabrik.js` shim,
  `npm/fabrik-{win32-x64,linux-x64,linux-arm64,darwin-x64,darwin-arm64}/`,
  `optionalDependencies`, `.github/workflows/release-cli.yml` build+publish,
  `scripts/sync-versions.mjs` with lifecycle-script guard, README quick-start.

## Phase 1 — Pure-npm publish contract (JS rules, still binding on the shim)
1. **bin + shebang.** Root `package.json`: `"bin": { "fabrik": "bin/fabrik.js" }`.
   `bin/fabrik.js` starts with `#!/usr/bin/env node`, is executable (`chmod +x`,
   preserved via git `core.fileMode` / CI `chmod`), CommonJS only (`require`,
   no ESM `import`, zero runtime deps) so it runs under both `node` and `bun`
   with no install step. (Researcher's `./dist/index.js` target applies if the
   CLI were TS; with the Go CLI in scope the same slot is the `bin/fabrik.js`
   launcher — one `bin` entry either way, never two.)
2. **No lifecycle scripts — ever.** No `postinstall`/`install`/`preinstall`/
   `preprepare` in root or any `npm/fabrik-*/package.json`.
   `scripts/sync-versions.mjs --check` fails if any exist (already enforced for
   root; extend to platform packages). No `curl|sh` download script as primary
   lane. Rationale: bun skips `postinstall` by default; any download-on-install
   breaks bun and trips `npm` supply-chain audits.
3. **dist build.** If JS ships: compile to `dist/` (`tsc`/`bun build`) and point
   `bin` at the built file; `dist/` rides the tarball via `files`, never built
   by a lifecycle script. With Go in scope: `dist/` equivalent is the
   prebuilt binary staged into `npm/<target>/bin/` by CI (Phase 2).
4. **files + publishConfig.**
   - Root `files`: `bin/fabrik.js`, `README.md` (+ `LICENSE` when added).
     Keeps the wrapper tarball dependency-free and tiny.
   - Platform `files`: `bin/fabrik` (or `bin/fabrik.exe` on win32-x64).
   - `publishConfig: { "access": "public" }` on root AND each platform package
     (scoped `@ur-wesley/*` defaults to restricted; CI also passes
     `--access public`, belt and suspenders). Keep `repository.url =
     https://github.com/ur-wesley/fabrik` everywhere (npm provenance requirement,
     already enforced by `sync-versions.mjs`).
5. **private-flag fix.**
   - Root `package.json` MUST NOT contain `"private": true` (publish would fail).
   - `cli/package.json` MUST KEEP `"private": true` — version mirror + Go
     module sidecar, never published to npm.
   - No `npm/fabrik-*/package.json` may be `private`.
   - `sync-versions.mjs --check` enforces all three (currently enforces
     versions + root lifecycle scripts only — extend to this).
6. **Remove `node -p require(...)` for bun-only compat.** Three occurrences must go
   (all break on a machine with only bun, or under `bun -e` semantics):
   - `package.json` script `cli:version`: `node -p "require('./package.json').version"`.
   - `.github/workflows/release-cli.yml` `version` step: same idiom.
   - `cli/internal/version/version.go` header comment documenting the ldflags stamp.
   Replace with one runtime-agnostic reader, e.g. `node scripts/get-version.mjs`
   (ESM `import` + `fs`, runs under `node` AND `bun`) or a
   `scripts/sync-versions.mjs --print-version` flag; scripts + CI + Go comment
   all call that. Acceptance: `bun scripts/get-version.mjs` (or chosen
   equivalent) prints `X.Y.Z` with `node` absent from `PATH`.
7. **Docs update.**
   - `README.md` Quick start: `npm i -g`, `bun add -g`, `npx`/`bunx`, `go install`,
     GH release binaries, `optionalDependencies` no-postinstall note, version
     single-source note (already present — keep in sync).
   - `PLAYBOOK.md` / hub docs: same lanes if they document install.
   - Legacy `install/setup.sh|.ps1` lane: `install/` currently holds only
     `deps.json` + `templates/` — confirm no shell-installer is reintroduced as
     a primary lane; shell scripts stay out of the npm path.

## Phase 2 — Go binary via shim + optionalDependencies (in scope, mostly built)
1. **Shim (`bin/fabrik.js`, landed — ratify).** Map
   `process.platform/arch` → `win32-x64 | linux-x64 | linux-arm64 | darwin-x64 |
   darwin-arm64`; binary name `fabrik.exe` on win32 else `fabrik`. Lookup order:
   `$FABRIK_BINARY` override → repo checkout (`npm/fabrik-<t>/bin/`,
   `cli/fabrik[.exe]` local build) → installed sibling
   (`../fabrik-<t>/bin/` under `node_modules`) → `require.resolve`
   (`@ur-wesley/fabrik-<t>/bin/...`, pnpm-safe). `spawnSync` with
   `stdio: inherit`. Unsupported platform and missing-binary errors name the
   expected `optionalDependency` and the `--no-optional` pitfall; exit nonzero.
2. **optionalDependencies (landed — ratify).** Five entries
   `@ur-wesley/fabrik-<target>` pinned to the exact root version
   (`sync-versions.mjs` enforces). Platform packages carry `os` + `cpu` so npm
   skips non-matching targets (bun honors `os`/`cpu` too); only the host binary
   downloads.
3. **CI publish matrix (landed — ratify + harden).**
   `.github/workflows/release-cli.yml`: on tag `v*`, build 5 Go targets
   (`CGO_ENABLED=0`, `-trimpath`, `-ldflags ...Version=<version>`), `sha256`,
   `go test ./...` gate, tag-vs-`package.json` gate (`sync-versions.mjs --check`),
   stage binaries into `npm/*/bin`, `chmod +x`, publish the 5 platform packages
   then the wrapper with `--provenance --access public` (OIDC trusted publishing,
   `NPM_TOKEN` fallback). Remaining hardening: switch the `version` step to the
   Phase-1 bun-safe reader; add `publishConfig` to platform packages.

## Test matrix (definition of done for install)
| # | Command | Runner | Asserts |
|---|---------|--------|---------|
| 1 | `npm i -g @ur-wesley/fabrik && fabrik version` | win32-x64, linux-x64 | version == tag; install log shows no lifecycle-script execution |
| 2 | `npx -p @ur-wesley/fabrik fabrik version` (no prior install) | linux-x64 | same version; one-shot run works |
| 3 | `bun add -g @ur-wesley/fabrik && fabrik version` | linux-x64 (+ darwin-arm64 if runner) | works with default (untrusted) bun config — proves no postinstall |
| 4 | `bunx --package @ur-wesley/fabrik fabrik version` | linux-x64 | same |
| 5 | `bun add @ur-wesley/fabrik` (local) + `bunx fabrik version` | linux-x64 | local lane works |
| 6 | `npm i -g --no-optional` then `fabrik` | linux-x64 | clean error naming the missing optionalDependency (negative path) |
| 7 | `sync-versions.mjs --check` + publish dry-run | CI | versions, lifecycle-script, private-flag, repository-url gates all green |
Bun-only check: cases 3–5 run with `node` removed from `PATH` on at least one
runner, plus the `get-version` acceptance in Phase 1 §6.

## Acceptance criteria
- [ ] `bin/fabrik.js` shebang + executable + zero-dep CJS; `files`/`publishConfig` per §4.
- [ ] No lifecycle scripts anywhere; `--check` guard extended to platform pkgs.
- [ ] Private flags: root + platform pkgs publishable, `cli/` stays `private`.
- [ ] Zero `node -p "require(...)"` occurrences (`package.json`, workflow, Go comment); bun-safe reader works with `node` absent.
- [ ] Docs (README + any install-lane docs) describe npm/bun/npx/bunx/go-install/binary lanes + no-postinstall note.
- [ ] Test matrix 1–7 green (or explicitly scoped-down with reason).

## Open questions (for APPROVE)
1. Ratify-vs-build: Phase 2 is ~90% landed — approve this spec as ratify+harden, or want deltas split into separate Beads issues?
2. Version reader: new `scripts/get-version.mjs` file, or `--print-version` flag on `sync-versions.mjs`? (Recommend flag — one fewer file.)
3. Add `publishConfig.access=public` (+ `LICENSE` to `files`) on the 5 platform packages now, or rely on CLI `--access public`?
4. `cli:version` / `cli:build` npm scripts: keep as node-invoked helpers or rewrite bun-first (`bun scripts/...`)? AGENTS.md quality gates call `go` directly, so risk is low either way.
5. Test runners: is linux-x64 + win32-x64 CI enough, or must darwin-arm64 be in-matrix before publish?
6. `npx`/`bunx` canonical form: document `npx -p @ur-wesley/fabrik fabrik ...` (bin name `fabrik` ≠ package short name) — confirm no package rename intended.
7. `trustedDependencies`: explicitly out of scope (zero scripts ⇒ nothing to trust) — confirm no consumer-side config docs needed.
