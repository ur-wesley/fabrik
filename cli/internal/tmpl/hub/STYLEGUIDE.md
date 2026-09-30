# Fabrik Code Style & Engineering Guide

This guide establishes the engineering rules for writing code in this repository. AI agents must check all modifications against these principles.

---

## 1. Modularity & Architecture

### Deep Modules vs. Shallow Modules
*   **Deep Modules (Simple interface, complex logic):** Write modules (classes, functions, files) that hide their internal complexity behind a small, simple API. This keeps the rest of the codebase insulated from changes.
*   **Avoid Shallow Modules:** Do not write modules that are thin wrappers around trivial operations. If a module has 3 lines of interface for 3 lines of implementation, it is shallow and increases cognitive load.
*   **Vertical Slices (Tracer Bullets):** Organise code by feature area rather than technical layer. Avoid scattering a single feature's logic across controllers, services, repositories, and DTOs unless strictly required. Keep cohesion high.

### Dependency Rules
*   **Decoupled Components:** Components must depend on abstractions (interfaces/types), not implementations.
*   **No Circular Dependencies:** Files must never import each other bidirectionally. Use events, callbacks, or shared modules to break loops.

---

## 2. Strict & Clean Typing

*   **No `any`:** The use of `any` is strictly prohibited. If a type is truly dynamic or unknown, use `unknown` and narrow it down using type guards or assertion functions.
*   **Strict Compiler Compliance:** Code must compile under the most strict configuration options (`strict: true`, `noImplicitAny: true`, etc.).
*   **Single Source of Truth:** Do not duplicate type definitions. Reuse existing types via generics, mapped types, or inheritance. Avoid write-only helper adapters that translate identical shapes.
*   **Discriminated Unions:** Prefer discriminated unions over boolean flags to represent state.
    ```typescript
    // BAD
    interface Request {
      isLoading: boolean;
      isError: boolean;
      data?: Data;
    }

    // GOOD
    type RequestState =
      | { status: 'idle' }
      | { status: 'loading' }
      | { status: 'success'; data: Data }
      | { status: 'error'; error: Error };
    ```

---

## 3. General Software Engineering Best Practices

### Simplicity & YAGNI
*   **You Aren't Gonna Need It (YAGNI):** Do not write code for future requirements. Only build what is needed to satisfy the current PRD.
*   **Avoid Over-abstraction:** A duplicate implementation is better than the wrong abstraction. Do not extract shared utilities until you have at least 3 distinct use cases.

### Test-Driven Development (TDD)
*   **Test Behavior, Not Code Shape:** Tests must exercise public interfaces. Do not test private methods or mock internal details.
*   **Red-Green-Refactor:** Follow the strict vertical cycle. Write a failing test first, make it pass with minimal code, then refactor for readability and performance.

### Self-Documenting Code
*   **Clear Naming:** Functions and variables must be named descriptively (verbs for functions, nouns for variables).
*   **Comments explain the 'Why', not the 'What':** Code should be readable enough to explain *what* it does. Use comments only to explain *why* non-obvious design decisions were made.
