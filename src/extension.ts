import type { ExtensionAPI } from './types';
import { loadConfig, DEFAULT_CONFIG } from './config';
import { stringifyYaml } from './yaml';
import { getTaskStatus, claimNextTask, completeTask } from './orchestrator';
import { beadsAvailable } from './bd';
import { existsSync, mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';
import { execSync } from 'child_process';
import { logger } from '@ur-wesley/ts-prelude/log';
import { runCatching } from '@ur-wesley/ts-prelude/result';

const log = logger.withTag('fabrik:extension');

export const VERSION = '2.0.0';

export default function fabrikExtension(pi: ExtensionAPI): void {
  const cwd = process.cwd();

  if (pi.on) {
    pi.on('session_start', () => {
      const config = loadConfig(cwd);
      log.info(`Fabrik Pi Extension v${VERSION} initialized (Skills: ${config.skills?.join(', ') ?? 'all'})`);
    });
  }

  pi.registerCommand('fabrik-init', {
    description: 'Initialize Fabrik config and Beads in this repository',
    handler: (_args, ctx) => {
      const fabrikDir = join(cwd, '.fabrik');

      runCatching(() => {
        if (!existsSync(fabrikDir)) mkdirSync(fabrikDir, { recursive: true });
        mkdirSync(join(fabrikDir, 'docs'), { recursive: true });

        const configYamlPath = join(fabrikDir, 'config.yaml');
        if (!existsSync(configYamlPath)) {
          const yamlStr = stringifyYaml(DEFAULT_CONFIG as unknown as Record<string, unknown>);
          writeFileSync(configYamlPath, yamlStr, 'utf8');
        }

        if (!beadsAvailable(cwd)) {
          execSync('bd init', { cwd, stdio: 'inherit' });
        }
      });

      const msg = `Initialized .fabrik/config.yaml and Beads for Pi Agent (Fabrik v${VERSION}).`;
      log.success(msg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(msg, 'success');
      }
    },
  });

  pi.registerCommand('fabrik-status', {
    description: 'Display Beads task counts, RTK status, and model routing',
    handler: (_args, ctx) => {
      const config = loadConfig(cwd);
      const status = getTaskStatus(cwd);

      const statusMsg =
        `Fabrik Status v${VERSION} (Pi Agent) | ` +
        `Open: ${status.openCount} | In Progress: ${status.inProgressCount} | Closed: ${status.completedCount} | ` +
        `Next: ${status.nextTaskId ?? 'none'} | ` +
        `RTK: ${config.tools?.rtk ? 'Enabled' : 'Disabled'} | Default Model: ${config.models?.default ?? 'Default'}`;

      log.info(statusMsg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(statusMsg, 'info');
      }
    },
  });

  pi.registerCommand('fabrik-plan', {
    description: 'Instruct Pi to partition PRD requirements into Beads issues via bd create',
    handler: (args, ctx) => {
      const focus = args ? ` Focus: ${args}` : '';
      const msg = `Running Fabrik Plan mode.${focus} Land issues with bd create and bd dep add.`;
      log.info(msg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(msg, 'info');
      }
    },
  });

  pi.registerCommand('caveman', {
    description: 'Toggle Caveman mode for ultra-concise token-efficient agent communication',
    handler: (args, ctx) => {
      const level = args.trim() || 'full';
      const msg = `Caveman protocol activated (intensity: ${level}).`;
      log.info(msg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(msg, 'info');
      }
    },
  });

  pi.registerCommand('ponytail', {
    description: 'Toggle Ponytail protocol for minimal code bloat and native-first solutions',
    handler: (args, ctx) => {
      const mode = args.trim() || 'full';
      const msg = `Ponytail protocol set to [${mode}].`;
      log.info(msg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(msg, 'info');
      }
    },
  });

  pi.registerCommand('grill-with-docs', {
    description: "Grill mode to interview codebase documentation and verify requirements before coding",
    handler: (args, ctx) => {
      const topic = args ? ` Topic: ${args}` : '';
      const msg = `Starting doc-grilling session.${topic}`;
      log.info(msg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(msg, 'info');
      }
    },
  });

  pi.registerCommand('rtk-compress', {
    description: 'Trigger RTK output filter for active shell tool executions',
    handler: (_args, ctx) => {
      const msg = 'RTK token compression active on bash tool executions.';
      log.info(msg);
      if (ctx.ui?.notify) {
        ctx.ui.notify(msg, 'info');
      }
    },
  });

  pi.registerTool({
    name: 'fabrik_next_task',
    description: 'Claim the next ready Beads issue via bd ready --claim',
    execute: async () => {
      return claimNextTask(cwd);
    },
  });

  pi.registerTool({
    name: 'fabrik_complete_task',
    description: 'Close a Beads issue and optionally auto-commit',
    execute: async (args: Record<string, unknown>) => {
      const taskId = String(args.taskId ?? args.taskName ?? '');
      if (!taskId) {
        return { status: 'error', message: 'taskId is required.' };
      }
      return completeTask(cwd, taskId);
    },
  });
}
