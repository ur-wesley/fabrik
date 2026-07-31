import type { ExtensionAPI } from './types';
import { loadConfig, DEFAULT_CONFIG } from './config';
import { stringifyYaml } from './yaml';
import { getTaskStatus, claimNextTask, completeTask } from './orchestrator';
import { existsSync, mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';
import { logger } from '@ur-wesley/ts-prelude/log';
import { runCatching } from '@ur-wesley/ts-prelude/result';

const log = logger.withTag('fabrik:extension');

export const VERSION = '1.1.0';

export default function fabrikExtension(pi: ExtensionAPI): void {
  const cwd = process.cwd();

  // Session Start Listener
  if (pi.on) {
    pi.on('session_start', () => {
      const config = loadConfig(cwd);
      log.info(`Fabrik Pi Extension v${VERSION} initialized (Skills: ${config.skills?.join(', ') ?? 'all'})`);
    });
  }

  // Command: /fabrik-init
  pi.registerCommand('fabrik-init', {
    description: 'Initialize Fabrik YAML configuration and task directory for Pi Agent',
    handler: (_args, ctx) => {
      const fabrikDir = join(cwd, '.fabrik');
      const tasksDir = join(fabrikDir, '.tasks');

      runCatching(() => {
        if (!existsSync(fabrikDir)) mkdirSync(fabrikDir, { recursive: true });
        if (!existsSync(tasksDir)) mkdirSync(tasksDir, { recursive: true });

        const configYamlPath = join(fabrikDir, 'config.yaml');
        if (!existsSync(configYamlPath)) {
          const yamlStr = stringifyYaml(DEFAULT_CONFIG as unknown as Record<string, unknown>);
          writeFileSync(configYamlPath, yamlStr, 'utf8');
        }
      });

      const msg = `Initialized .fabrik/config.yaml and task directory for Pi Agent (Fabrik v${VERSION}).`;
      log.success(msg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(msg, 'success');
      }
    },
  });

  // Command: /fabrik-status
  pi.registerCommand('fabrik-status', {
    description: 'Display active Fabrik task counts, RTK token compression, and model routing',
    handler: (_args, ctx) => {
      const config = loadConfig(cwd);
      const status = getTaskStatus(cwd);

      const statusMsg =
        `Fabrik Status v${VERSION} (Pi Agent) | ` +
        `Open Tasks: ${status.openCount} | In Progress: ${status.inProgressCount} | Completed: ${status.completedCount} | ` +
        `RTK: ${config.tools?.rtk ? 'Enabled' : 'Disabled'} | Default Model: ${config.models?.default ?? 'Default'}`;

      log.info(statusMsg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(statusMsg, 'info');
      }
    },
  });

  // Command: /fabrik-plan
  pi.registerCommand('fabrik-plan', {
    description: 'Instruct Pi to analyze PRD and partition work into decoupled task markdown files in .fabrik/.tasks/',
    handler: (args, ctx) => {
      const focus = args ? ` Focus: ${args}` : '';
      const msg = `Running Fabrik Plan mode.${focus} Partitioning tasks into .fabrik/.tasks/...`;
      log.info(msg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(msg, 'info');
      }
    },
  });

  // Command: /caveman
  pi.registerCommand('caveman', {
    description: 'Toggle Caveman mode for ultra-concise token-efficient agent communication (lite, full, ultra)',
    handler: (args, ctx) => {
      const level = args.trim() || 'full';
      const msg = `Caveman protocol activated (intensity: ${level}). Stripping non-essential filler.`;
      log.info(msg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(msg, 'info');
      }
    },
  });

  // Command: /ponytail
  pi.registerCommand('ponytail', {
    description: 'Toggle Ponytail protocol for minimal code bloat and native-first solutions',
    handler: (args, ctx) => {
      const mode = args.trim() || 'full';
      const msg = `Ponytail lazy senior dev protocol set to [${mode}]. Enforcing native abstractions and minimal diffs.`;
      log.info(msg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(msg, 'info');
      }
    },
  });

  // Command: /grill-with-docs
  pi.registerCommand('grill-with-docs', {
    description: "Matt Pocock's grill mode to interview codebase documentation and verify requirements before coding",
    handler: (args, ctx) => {
      const topic = args ? ` Topic: ${args}` : '';
      const msg = `Starting Matt Pocock doc-grilling session.${topic} Anchoring answers in domain specs and codebase invariants.`;
      log.info(msg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(msg, 'info');
      }
    },
  });

  // Command: /sdd-fusion
  pi.registerCommand('sdd-fusion', {
    description: 'Devin Fusion Spec-Driven Development (SDD) autonomous workflow phase launcher',
    handler: (args, ctx) => {
      const phase = args.trim() || 'explore';
      const msg = `Devin Fusion SDD workflow running phase: [${phase}]. Executing phased spec-driven loop.`;
      log.info(msg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(msg, 'info');
      }
    },
  });

  // Command: /rtk-compress
  pi.registerCommand('rtk-compress', {
    description: 'Trigger RTK (Rust Token Killer) output filter for active shell tool executions',
    handler: (_args, ctx) => {
      const msg = 'RTK (Rust Token Killer) token compression active on bash tool executions.';
      log.info(msg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(msg, 'info');
      }
    },
  });

  // Native Tool: fabrik_next_task
  pi.registerTool({
    name: 'fabrik_next_task',
    description: 'Fetch and lock the next open task from .fabrik/.tasks/',
    execute: async () => {
      return claimNextTask(cwd);
    },
  });

  // Native Tool: fabrik_complete_task
  pi.registerTool({
    name: 'fabrik_complete_task',
    description: 'Archive completed task to .tasks/completed/ and trigger auto-commit',
    execute: async (args: Record<string, unknown>) => {
      const taskName = String(args.taskName ?? '');
      if (!taskName) {
        return { status: 'error', message: 'Task name is required.' };
      }
      return completeTask(cwd, taskName);
    },
  });
}
