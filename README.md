# Fabrik — Native Pi Agent Extension Package 🚀

**Fabrik** is a native **Pi Agent Extension Package** ([pi.dev](https://pi.dev/)) designed for structured, test-driven, multi-task orchestration.

It connects beste-in-class AI developer tools (**RTK** output token compression, **Engram** cross-session memory, **Ponytail** YAGNI rules, **mattpocock/skills**) into a single, low-token declarative YAML configuration (`.fabrik/config.yaml`).

---

## 📦 Installation & Setup

### 1. Install via Pi Package Manager
```bash
pi install git:https://github.com/ur-wesley/fabrik.git
```

### 2. Or Test Directly in Current Session
```bash
pi -e ./index.ts
```

*When starting `pi` inside a project containing `.pi/extensions/fabrik.ts`, Fabrik is automatically discovered and loaded.*

---

## ⚡ Features & Slash Commands

| Slash Command / Tool | Description |
| :--- | :--- |
| `/fabrik-init` | Bootstraps `.fabrik/config.yaml` and `.fabrik/.tasks/` in your repository |
| `/fabrik-status` | Displays task counts (open, in-progress, completed), RTK status, and active model |
| `/fabrik-plan` | Prompts Pi to partition PRD requirements into decoupled `.fabrik/.tasks/*.md` task files |
| `fabrik_next_task` | Native LLM tool: Atomically claims and locks the next task into `.tasks/.in-progress/` |
| `fabrik_complete_task` | Native LLM tool: Archives finished tasks to `.tasks/completed/` and creates auto-commits |

---

## 📑 Low-Token Declarative Configuration (`.fabrik/config.yaml`)

```yaml
agent: pi

models:
  default: anthropic/claude-3-5-sonnet
  grill: anthropic/claude-3-5-sonnet
  plan: openai/o3-mini
  build: anthropic/claude-3-5-sonnet

session:
  auto_lock: true       # Prevents multi-session collisions on task files
  auto_commit: true     # Automatically creates conventional git commits on task completion

tools:
  rtk: true            # Intercepts CLI outputs (test, git) to save 60-90% LLM tokens
  engram: true         # Persistent cross-session memory layer

skills:
  - ponytail          # YAGNI & minimal code guidelines
  - grill-with-docs   # Interactive alignment
  - to-prd            # Specification generator
  - to-issues         # Task partitioning
  - tdd               # Test-driven development enforcement
  - caveman           # Token-saving blunt response style
```
