# Spec: Project Health & Thin-Hub Consolidation

## Summary
Fabrik recently underwent a major architectural migration from legacy shell scripts (`.fabrik/*.ps1`, `.fabrik/*.sh`, `install/*.ps1`, `install/*.sh`) to a unified, compiled Go CLI (`fabrik setup|init|show|list|run|loop|migrate`) with a "thin hub" model (`.fabrik/` containing only `config.yaml`, `docs/`, `specs/`).

However, the migration left significant drift across automated tests, documentation, and template synchronizations:
1. `cli/internal/installer/installer_test.go` hangs on interactive console confirmation prompt (`huh.NewForm`) in `TestSetupRepoInitBeforeWorkflowNote`.
2. `cli/internal/tmpl/tmpl_test.go` fails due to drift in `cli/opencode.json` and checking deleted legacy `.fabrik/` files instead of authoritative templates.
3. `e2e` tests fail because they assert deleted shell scripts and old workflow prompt contents.
4. `package.json` test scripts fail on Windows due to unexpanded glob patterns and broken `bunx` shims.
5. `PLAYBOOK.md`, `CLAUDE.md`, and `README.md` contain outdated commands and contradictory rules regarding memory vs issues (`bd remember` vs Engram).
6. TypeScript extension (`src/extension.ts`) still creates legacy hub directories (`.fabrik/styleguide`, `.fabrik/agents`) and points to deleted shell scripts.

## Key Changes Proposed

### 1. CLI & Test Stability (P0)
- **Injectable Confirmation Seam**: Add `Confirm func(question string, yes bool) bool` to `installer.Deps` (defaulting to `ui.Confirm`). In `ui.Confirm`, guard against running in automated test mode. Pass `Yes: true` in `TestSetupRepoInitBeforeWorkflowNote`.
- **Sync Tests Alignment**: Update `tmpl_test.go` so `pairs` assert sync against `install/templates/hub/*` instead of deleted `.fabrik/*` files. Sync `cli/opencode.json` with embedded `opencode.json`.
- **E2E Tests Update**: Update `e2e/helpers/repo.ts`, `e2e/install.e2e.test.ts`, and `e2e/workflow.e2e.test.ts` to test the thin hub model and Go CLI rather than deleted `.ps1` scripts.

### 2. Cross-Platform Scripts & Config (P1)
- In `package.json`:
  - Change `"test:e2e"` from `"bun test ./e2e/*.e2e.test.ts"` to `"bun test e2e"` for cross-platform compatibility.
  - Fix `"lint"` from `"bunx oxlint src"` to `"bun run oxlint src"` (or local bin).

### 3. Documentation & Spec Consistency (P1)
- **`PLAYBOOK.md`**: Update architecture diagram, machine setup, directory layout, and AFK orchestrator sections to reflect Go CLI (`fabrik setup`, `fabrik init`, `fabrik run`, `fabrik loop`).
- **`CLAUDE.md`**: Fix line 23 to align with `AGENTS.md` & `README.md` ("Beads owns tasks, Engram owns memory. Do not use bd remember for decisions").
- **`README.md`**: Remove reference to deleted `.opencode/plugins/fabrik.ts`.

### 4. Pi Extension Modernization (`src/`) (P2)
- Update `src/extension.ts` `fabrik-init` to only scaffold `docs/` and `specs/`, matching thin hub architecture.
- Update missing tool message from `Run install/setup.sh or setup.ps1` to `Run fabrik setup`.
- Clean up OxLint warnings in `src/`.

## Success Criteria
- `bun test ./src` passes 100%.
- `cd cli && go test ./...` passes 100% without timeouts or interactive hangs.
- `bun test e2e` passes 100%.
- All documentation accurately reflects the Go CLI architecture with zero contradictions.
