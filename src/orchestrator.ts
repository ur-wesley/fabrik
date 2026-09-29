import { execSync } from 'child_process';
import { loadConfig, type FabrikConfig } from './config';
import {
  beadsAvailable,
  claimReadyIssue,
  closeIssue,
  getReadyIssues,
  listIssues,
  type BeadsIssue,
} from './bd';
import { runCatching, matchResult } from '@ur-wesley/ts-prelude/result';
import { logger } from '@ur-wesley/ts-prelude/log';

const log = logger.withTag('fabrik:orchestrator');

export function resolveModel(
  config: FabrikConfig,
  mode: 'grill' | 'plan' | 'build',
  taskModel?: string,
): string | undefined {
  if (taskModel && taskModel.trim().length > 0) {
    return taskModel.trim();
  }
  const modeModel = config.models?.[mode];
  if (modeModel && modeModel.trim().length > 0) {
    return modeModel.trim();
  }
  const defaultModel = config.models?.default;
  if (defaultModel && defaultModel.trim().length > 0) {
    return defaultModel.trim();
  }
  return undefined;
}

export function commitTask(cwd: string, taskName: string): boolean {
  const res = runCatching(() => {
    execSync(`git add . && git commit -m "feat(tasks): complete ${taskName}"`, {
      cwd,
      stdio: 'ignore',
    });
  });

  return matchResult(res, {
    ok: () => {
      log.info(`Auto-committed task: ${taskName}`);
      return true;
    },
    err: () => false,
  });
}

export interface TaskStatus {
  openCount: number;
  inProgressCount: number;
  completedCount: number;
  nextTaskId?: string;
  nextTaskName?: string;
  nextTaskModel?: string;
}

function nextReady(cwd: string): BeadsIssue | undefined {
  const ready = getReadyIssues(cwd);
  return ready[0];
}

export function getTaskStatus(cwd: string): TaskStatus {
  if (!beadsAvailable(cwd)) {
    return { openCount: 0, inProgressCount: 0, completedCount: 0 };
  }

  const open = listIssues(cwd, 'open');
  const inProgress = listIssues(cwd, 'in_progress');
  const completed = listIssues(cwd, 'closed');
  const config = loadConfig(cwd);
  const next = nextReady(cwd);

  const result: TaskStatus = {
    openCount: open.length,
    inProgressCount: inProgress.length,
    completedCount: completed.length,
  };
  if (next) {
    result.nextTaskId = next.id;
    result.nextTaskName = next.title;
    const model = resolveModel(config, 'build');
    if (model) result.nextTaskModel = model;
  }

  return result;
}

export function claimNextTask(cwd: string): {
  status: 'locked' | 'empty';
  taskId?: string;
  taskName?: string;
  model?: string;
  content?: string;
} {
  if (!beadsAvailable(cwd)) {
    return { status: 'empty' };
  }

  const issue = claimReadyIssue(cwd);
  if (!issue) {
    return { status: 'empty' };
  }

  const config = loadConfig(cwd);
  const model = resolveModel(config, 'build');
  log.info(`Claimed Beads issue ${issue.id}: ${issue.title}`);

  const res: {
    status: 'locked' | 'empty';
    taskId?: string;
    taskName?: string;
    model?: string;
    content?: string;
  } = {
    status: 'locked',
    taskId: issue.id,
    taskName: issue.title,
    content: issue.description ?? issue.title,
  };
  if (model) res.model = model;

  return res;
}

export function completeTask(
  cwd: string,
  taskId: string,
): { status: 'completed' | 'error'; taskId: string; taskName?: string; message?: string } {
  const config = loadConfig(cwd);

  if (!closeIssue(cwd, taskId)) {
    return { status: 'error', taskId, message: `Failed to close ${taskId}` };
  }

  if (config.session?.auto_commit) {
    commitTask(cwd, taskId);
  }

  return { status: 'completed', taskId };
}
