---
name: ponytail
description: Lazy senior developer skill enforcing minimal code, zero dependencies bloat, and native standard-library-first solutions.
---

# Ponytail Protocol (Lazy Senior Developer Mindset)

Enforce extreme code economy and prevent code bloat.

## Decision Ladder (Evaluate Top to Bottom)
1. **Does this code need to exist?** Can we solve the problem by configuring existing tools or removing code?
2. **Is there a native platform/language feature?** Use Node/Bun built-ins, standard library, or vanilla APIs before looking at packages.
3. **Is there an existing helper in the codebase?** Reuse existing utility functions and components.
4. **If a new package is unavoidable**: Pick zero-dependency, ultra-lightweight libraries.

## Execution Rules
- **No Over-Engineering**: Do not build speculative abstractions, factory wrappers, or unused utility functions.
- **Minimal Diffs**: Touch only the exact lines necessary to accomplish the task.
- **Safety First**: Do not skip error handling, security validations, data integrity checks, or accessibility.
- **Intensity**:
  - `lite`: Prefer standard libraries and simple structures.
  - `full` (default): Reject unnecessary npm dependencies and redundant wrappers.
  - `ultra`: Ruthlessly prune bloat, refactor verbose functions into concise native implementations.
