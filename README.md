# Fabrik: The AI-Agent Engineering Framework

**Fabrik** is a structured, test-driven development workflow designed to run AI coding agents (specifically **OpenCode**) in reliable, autonomous loops. 

It completely replaces conversational "vibe coding" with predictable software engineering cycles: **Grill → PRD → Decoupled Issues → Test-First Implementation → Auto-Commit Loop**.

---

## 🚀 Quick Start

### 📦 Add Fabrik to an Existing Project (One-Liner)
Run this one-liner in your project's root directory to download and set up the `ai/` folder:
```bash
npx degit ur-wesley/fabrik/ai ai
```
*Alternatively, using native Git:*
```bash
git clone --depth 1 --sparse https://github.com/ur-wesley/fabrik.git && cd fabrik && git sparse-checkout set ai && mv ai .. && cd .. && rm -rf fabrik
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
    .\ai\setup.ps1
    ```
*   **Unix / Git Bash:**
    ```bash
    ./ai/setup.sh
    ```

### 3. Initialize OpenCode Skills Configuration
Trigger OpenCode in this directory and run the official setup skill:
```bash
/setup-matt-pocock-skills
```
*(This will ask you for your preferred issue tracker and documentation preferences).*

---

## 🛠️ The Workflow

### Step 1: Grill & Align
Open your chat session with OpenCode and run:
```bash
/grill-with-docs
```
The agent will interview you about your new feature or bug, checking requirements against your codebase and code guidelines. This refines your domain language and saves it in `ai/CONTEXT.md`.

### Step 2: Generate the PRD
After aligning during the grilling session, run the PRD generator inside OpenCode:
```bash
/to-prd
```
This writes the requirements specification document directly to `ai/docs/PRD.md`.

### Step 3: Split PRD into Decoupled Tasks
Run the planning agent to perform a gap analysis (specs vs. existing code) and partition the work into independent, non-overlapping task files in `ai/.tasks/`:
*   **Windows:** `.\ai\loop.ps1 -Mode plan`
*   **Linux/macOS:** `./ai/loop.sh plan`

### Step 4: Run the Build Loop (AFK Mode)
Start the automated builder. It will load `opencode --agent build`, select the lowest-numbered open task file in `ai/.tasks/`, implement it, verify the build/tests, move the task to `completed/`, commit and push, and restart with a clean context window:
*   **Windows:** `.\ai\loop.ps1 -Mode build`
*   **Linux/macOS:** `./ai/loop.sh build`

---

## 📂 Core Reference Documentation

*   **[PLAYBOOK.md](PLAYBOOK.md)**: Human-facing workflow instructions and commands reference.
*   **[ai/styleguide/STYLEGUIDE.md](ai/styleguide/STYLEGUIDE.md)**: Rules for strict, clean typing, deep modularity, and software design principles.
*   **[ai/AGENTS.md](ai/AGENTS.md)**: System instruction file governing the agent's **Caveman** behaviour and testing backpressure.
