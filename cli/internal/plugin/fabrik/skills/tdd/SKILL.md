---
name: tdd
description: Test-driven implementation guardrail. Red-green-refactor, minimal scope, backpressure before close.
---

# TDD Protocol

## Loop
1. Red: write failing test for one behavior only.
2. Green: minimal code to pass.
3. Refactor: ponytail rules, keep green.

## Rules
- One behavior per cycle. No speculative cases.
- Run project test command before `bd close`.
- Fix failures before closing. No broken-green claims.
