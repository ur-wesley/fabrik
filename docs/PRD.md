# PRD — Adopt oxc Suite (oxlint + oxfmt) for Linting & Formatting

## Problem Statement

The repository currently uses Biome (`@biomejs/biome` v2.5.0) for both linting and formatting, configured via `biome.json`. Biome works, but the user has requested the **oxc suite** — `oxlint` (linter) and `oxfmt` (formatter) — as the canonical tooling. The current setup is therefore inconsistent with the declared direction and should be replaced so:

- One toolchain (oxc) covers both lint and format.
- The codebase stops depending on `@biomejs/biome`.
- Future agents / contributors see the new toolchain reflected in scripts, configs, and docs.
- The project's existing strict-typing and no-fluff conventions (see `biome.json` rule set) are preserved.

## Solution

Replace Biome with the **oxc suite** end-to-end:

- Add `oxlint` and `oxfmt` as devDependencies.
- Add `.oxlintrc.json` and `.oxfmtrc.json` at the repo root that replicate the current Biome rule intent.
- Update `package.json` scripts so `lint`, `lint:fix`, and `format` invoke oxlint/oxfmt.
- Delete `biome.json` and remove `@biomejs/biome` from `devDependencies`.
- Add a small test that locks in the new toolchain (config presence + script wiring).
- Update `.fabrik/CONTEXT.md` glossary with the new tooling terms.

The change is **atomic** — one commit, no feature flags, no coexistence period. The project's surface area is small enough that staged rollout adds risk without benefit.

## User Stories

1. As a maintainer, I want `npm run lint` to invoke `oxlint`, so that linting runs on the chosen toolchain.
2. As a maintainer, I want `npm run format` to invoke `oxfmt`, so that formatting runs on the chosen toolchain.
3. As a maintainer, I want `npm run lint:fix` to auto-fix lint issues and re-format source, so that one command tidies a file.
4. As a maintainer, I want `oxlint` configured to enforce the same rules Biome enforced, so that code quality does not regress.
5. As a maintainer, I want `oxfmt` configured with the project's formatter conventions (2-space indent, single quotes, semicolons, trailing commas, 100-col line width), so that output looks identical to today.
6. As a maintainer, I want the unused-variable and unused-import rules enforced as errors, so that dead code fails CI.
7. As a maintainer, I want `noExplicitAny` enforced as an error, so that `any` cannot sneak in.
8. As a maintainer, I want `noNonNullAssertion` reported as a warning, so that `!` is discouraged but not a hard failure.
9. As a maintainer, I want `noConsole` to allow only `console.warn` and `console.error`, so that accidental `console.log` is caught.
10. As a maintainer, I want `useImportType` enforced as an error, so that type-only imports are explicit.
11. As a maintainer, I want organize-imports behavior preserved via oxlint's import rule, so that imports stay ordered.
12. As a maintainer, I want `biome.json` removed from the repo, so that there is a single source of truth for tooling config.
13. As a maintainer, I want `@biomejs/biome` removed from `devDependencies`, so that installs no longer pull it in.
14. As a maintainer, I want `bun.lock` regenerated cleanly, so that the lockfile reflects the new toolchain.
15. As a CI system, I want `npm run test`, `npm run typecheck`, `npm run lint`, and `npm run build` to all pass after migration, so that backpressure is intact.
16. As a build-loop agent, I want the backpressure commands in `.fabrik/AGENTS.md` and `.fabrik/PROMPT_build.md` to keep working unchanged, so that the agent loop does not break.
17. As a future contributor, I want the toolchain decision documented in `.fabrik/CONTEXT.md`, so that the rationale is discoverable.
18. As a future contributor, I want a test that fails if the oxc configs are missing, so that a sloppy cleanup is caught.
19. As a future contributor, I want a test that fails if the `package.json` scripts no longer reference oxlint/oxfmt, so that the wiring is locked in.
20. As a Windows user, I want oxlint/oxfmt to work from a PowerShell session via `bunx`/`npx`, so that the migration is portable.
21. As a reviewer, I want the diff to be reviewable in one pass, so that the migration is easy to accept or reject as a unit.
22. As a maintainer, I want the formatter to round-trip source files without changing semantics, so that a `format` run is a no-op on already-formatted files.
23. As a maintainer, I want the migration to leave no dead config files behind, so that `biome.json` is not silently reactivated by an old muscle memory.

## Implementation Decisions

- **Toolchain**: Use `oxlint` (linting) and `oxfmt` (formatting). Do not include oxc transformer or minifier.
- **Package manager**: Keep `bun`. Invoke binaries via `bunx oxlint …` / `bunx oxfmt …` so a global install is not required.
- **Configs**:
  - `.oxlintrc.json` at repo root: enables the categories that mirror the current Biome rule set (`correctness`, `suspicious`, `style`, `pedantic`/`restriction` as needed). Each rule mapped by ESLint-equivalent name (`no-unused-vars`, `no-unused-imports` / `no-unused-vars` with `varsIgnorePattern`, `no-explicit-any`, `no-non-null-assertion`, `no-console` with allow list, `@typescript-eslint/consistent-type-imports`).
  - `.oxfmtrc.json` at repo root: 2-space indent, single quotes, semicolons, trailing commas, line width 100.
  - Organize-imports handled inside oxlint via the `import/order` (or oxlint's equivalent) rule. oxfmt is not expected to re-order imports independently.
- **`package.json` scripts**:
  - `lint` → `bunx oxlint src`
  - `lint:fix` → `bunx oxlint --fix src && bunx oxfmt src`
  - `format` → `bunx oxfmt src`
  - `typecheck`, `test`, `build` remain unchanged.
- **`devDependencies`**: add `oxlint` and `oxfmt` (caret-pinned to the latest at time of change); remove `@biomejs/biome`.
- **Cleanup**: delete `biome.json`. Run `bun install` to regenerate `bun.lock`.
- **Backpressure**: keep the three commands in `.fabrik/AGENTS.md` (`npm run test`, `npm run lint`, `npm run build`). They continue to work via the rewritten `lint` script.
- **Test seam**: add `src/toolchain.test.ts` that asserts:
  1. `.oxlintrc.json` exists and parses as JSON.
  2. `.oxfmtrc.json` exists and parses as JSON.
  3. `biome.json` does NOT exist (negative assertion — locks in the cleanup).
  4. `package.json` scripts `lint`, `lint:fix`, `format` each reference `oxlint` or `oxfmt`.
- **No source-code edits expected.** The migration is config-only; oxlint/oxfmt are expected to accept the existing `src/` without changes (or surface minor reformat suggestions which we apply in the same commit).
- **Docs**: Update `.fabrik/CONTEXT.md` with the new terms (`oxc suite`, `oxlint`, `oxfmt`) and the rule-mapping table. PLAYBOOK.md and README.md do not reference Biome today, so they need no edit.

### Rule-mapping table (decision artifact)

| Biome rule                              | oxlint equivalent                                                  | Severity   |
|-----------------------------------------|---------------------------------------------------------------------|------------|
| `noUnusedVariables`                     | `no-unused-vars`                                                   | error      |
| `noUnusedImports`                       | `no-unused-vars` (with `vars: 'all'` + `args: 'none'`) + import plugin | error  |
| `noExplicitAny`                         | `no-explicit-any`                                                   | error      |
| `noNonNullAssertion`                    | `no-non-null-assertion`                                             | warn       |
| `noConsole` (allow `warn`,`error`)      | `no-console` with allow-list                                         | warn       |
| `useImportType`                         | `@typescript-eslint/consistent-type-imports`                        | error      |
| `recommended` (catch-all)               | `oxlint`'s default rule categories                                  | error/warn |
| `organizeImports` (assist action)       | `import/order` (or oxlint's import-order rule)                      | error      |
| formatter: 2-space, single quotes, …    | `.oxfmtrc.json`                                                     | —          |

## Testing Decisions

- **What makes a good test here**: tests assert **tooling wiring**, not lint output. They check the contract between config files and `package.json` so a sloppy cleanup is caught.
- **Modules under test**: `src/toolchain.test.ts` (new). Uses `bun:test` like the rest of the suite (`src/orchestrator.test.ts`, `src/config.test.ts`).
- **Existing prior art**: `src/config.test.ts` validates `loadConfig` end-to-end via the file system — same pattern applies for asserting config presence.
- **Manual verification gate**: run `npm run lint` and `npm run format` on a representative `src/*.ts` file to confirm clean output (no diff). Document the expected exit code (0).

## Out of Scope

- Migrating to oxc's transformer or minifier.
- Adding custom ESLint plugins (oxlint rule set is final for this migration).
- Reformatting of generated files (`*.tsbuildinfo`, `bun.lock`) — oxlint/oxfmt ignore those.
- Cross-platform CI matrix testing beyond the existing Windows PowerShell target.
- Replacing `tsc --noEmit` with `oxc`'s type-aware checker — type-checking stays on `tsc` until oxc ships a stable type checker.
- Refactoring `src/` beyond what oxlint/oxfmt request automatically.

## Further Notes

- The migration is a single atomic commit. If oxlint/oxfmt surface reformat suggestions, they are folded into the same commit so the diff is one logical change.
- The user's instruction places this PRD at `docs/PRD.md` (repo root). The project's PLAYBOOK references `.fabrik/docs/PRD.md`. Both paths are valid; this PRD lives at the user-specified path. `.fabrik/docs/PRD.md` is left untouched unless the user requests relocation.
- `.fabrik/CONTEXT.md` is the canonical glossary location (per `.fabrik/AGENTS.md` and `PLAYBOOK.md`) and is updated alongside this PRD.