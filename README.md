# Fabrik: The AI-Agent Engineering Framework

**Fabrik** is a structured, test-driven development workflow designed to run AI coding agents (specifically **OpenCode**) in reliable, autonomous loops. 

It completely replaces conversational "vibe coding" with predictable software engineering cycles: **Grill → PRD → Decoupled Issues → Test-First Implementation → Auto-Commit Loop**.

---

## 🚀 Quick Start

### 📦 Add Fabrik to an Existing Project (One-Liner)
Run this one-liner in your project's root directory to download and set up the `.fabrik/` folder:
```bash
npx degit ur-wesley/fabrik/.fabrik .fabrik
```
*Alternatively, using native Git:*
```bash
git clone --depth 1 --sparse https://github.com/ur-wesley/fabrik.git && cd fabrik && git sparse-checkout set .fabrik && mv .fabrik .. && cd .. && rm -rf fabrik
```

### 1. Prerequisites
Ensure you have the following installed on your machine:
*   [Node.js](https://nodejs.org/) (includes `npx`)
*   [Git](https://git-scm.com/)
*   **OpenCode CLI** (installed in your environment)

### 2. Run Setup
Initialize the directory structure and install the battle-tested community skills (like `/grill-with-docs`, `/to-prd`, `/to-issues`, and `/caveman`) from the `mattpocock/skills` repository.

*   **Windows (PowerShell):**
    ```powershell
    .\.fabrik\setup.ps1
    ```
*   **Unix / Git Bash:**
    ```bash
    ./.fabrik/setup.sh
    ```

### 3. Initialize OpenCode Skills Configuration
Trigger OpenCode in this directory and run the official setup skill:
```bash
/setup-matt-pocock-skills
```
*(This will ask you for your preferred issue tracker and documentation preferences).*

### 4. Run the Master Orchestrator (Back-to-Back Flow)
To execute the entire pipeline autonomously from a single command:
*   **Windows (PowerShell):**
    ```powershell
    .\fabrik.ps1
    ```
*   **Unix / Git Bash:**
    ```bash
    ./fabrik.sh
    ```
*This starts the TUI for alignment, then automatically switches to planning and building loop modes.*

---

## 🛠️ The Manual/Step-by-Step Workflow

### Step 1: Grill & Align
Open your chat session with OpenCode and run:
```bash
/grill-with-docs
```
The agent will interview you about your new feature or bug, checking requirements against your codebase and code guidelines. This refines your domain language and saves it in `.fabrik/CONTEXT.md`.

### Step 2: Generate the PRD
After aligning during the grilling session, run the PRD generator inside OpenCode:
```bash
/to-prd
```
This writes the requirements specification document directly to `.fabrik/docs/PRD.md`.

### Step 3: Split PRD into Decoupled Tasks
Run the planning agent to perform a gap analysis (specs vs. existing code) and partition the work into independent, non-overlapping task files in `.fabrik/.tasks/`:
*   **Windows:** `.\.fabrik\loop.ps1 -Mode plan`
*   **Linux/macOS:** `./.fabrik/loop.sh plan`

### Step 4: Run the Build Loop (AFK Mode)
Start the automated builder. It will load `opencode --agent build`, select the lowest-numbered open task file in `.fabrik/.tasks/`, implement it, verify the build/tests, move the task to `completed/`, commit and push, and restart with a clean context window:
*   **Windows:** `.\.fabrik\loop.ps1 -Mode build`
*   **Linux/macOS:** `./.fabrik/loop.sh build`

---

## 📂 Core Reference Documentation

*   **[PLAYBOOK.md](PLAYBOOK.md)**: Human-facing workflow instructions and commands reference.
*   **[.fabrik/styleguide/STYLEGUIDE.md](.fabrik/styleguide/STYLEGUIDE.md)**: Rules for strict, clean typing, deep modularity, and software design principles.
*   **[.fabrik/AGENTS.md](.fabrik/AGENTS.md)**: System instruction file governing the agent's **Caveman** behaviour and testing backpressure.
