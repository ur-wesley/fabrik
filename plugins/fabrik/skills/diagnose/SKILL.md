---
name: diagnose
description: Disciplined bug diagnosis. Reproduce, minimise, hypothesise, instrument, fix, regression-test.
---

# Diagnose Protocol

## Loop
1. Reproduce with minimal case.
2. Minimise scope (file / input / flag).
3. Hypothesise (1-2 candidates, evidence-backed).
4. Instrument (log / test / graph query).
5. Fix minimal diff.
6. Regression test added.

## Rules
- Evidence before synthesis. Read files, don't guess.
- If two hypotheses, test both, report outcome.
