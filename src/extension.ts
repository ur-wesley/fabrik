import type { ExtensionAPI } from './types';
import { loadConfig, DEFAULT_CONFIG } from './config';
import { stringifyYaml } from './yaml';
import { getTaskStatus, claimNextTask, completeTask } from './orchestrator';
import { existsSync, mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';
import { logger } from '@ur-wesley/ts-prelude/log';
import { runCatching } from '@ur-wesley/ts-prelude/result';

const log = logger.withTag('fabrik:extension');

export const VERSION = '1.0.0';

export default function fabrikExtension(pi: ExtensionAPI): void {
  const cwd = process.cwd();

  // Session Start Listener
  if (pi.on) {
    pi.on('session_start', () => {
      const config = loadConfig(cwd);
      log.info(`Fabrik Pi Extension v${VERSION} initialized (Default model: ${config.models?.default ?? 'default'})`);
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
