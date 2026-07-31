import { readdirSync, readFileSync, existsSync, mkdirSync, renameSync } from 'fs';
import { join } from 'path';
import { execSync } from 'child_process';
import { loadConfig, type FabrikConfig } from './config';
import { parseYaml } from './yaml';
import { runCatching, matchResult } from '@ur-wesley/ts-prelude/result';
import { fromNullable, map } from '@ur-wesley/ts-prelude/option';
import { logger } from '@ur-wesley/ts-prelude/log';

const log = logger.withTag('fabrik:orchestrator');

export function parseTaskFrontmatter(content: string): { model?: string } {
  if (!content.startsWith('---')) return {};
  const endIdx = content.indexOf('---', 3);
  if (endIdx === -1) return {};
  const frontmatterStr = content.slice(3, endIdx).trim();
  const parsed = parseYaml(frontmatterStr);
  const modelStr = typeof parsed.model === 'string' ? parsed.model : undefined;
  return modelStr ? { model: modelStr } : {};
}

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
  nextTaskName?: string;
  nextTaskModel?: string;
}

export function getTaskStatus(cwd: string): TaskStatus {
  const tasksDir = join(cwd, '.fabrik', '.tasks');
  const inProgressDir = join(tasksDir, '.in-progress');
  const completedDir = join(tasksDir, 'completed');

  const openTasks = existsSync(tasksDir)
    ? readdirSync(tasksDir).filter((f) => f.endsWith('.md')).sort()
    : [];
  const inProgressCount = existsSync(inProgressDir)
    ? readdirSync(inProgressDir).filter((f) => f.endsWith('.md')).length
    : 0;
  const completedCount = existsSync(completedDir)
    ? readdirSync(completedDir).filter((f) => f.endsWith('.md')).length
    : 0;

  let nextTaskModel: string | undefined;
  if (openTasks.length > 0) {
    const config = loadConfig(cwd);
    const readRes = runCatching(() => readFileSync(join(tasksDir, openTasks[0]!), 'utf8'));
    map(fromNullable(readRes.isOk() ? readRes.value : null), (content) => {
      const frontmatter = parseTaskFrontmatter(content);
      nextTaskModel = resolveModel(config, 'build', frontmatter.model);
    });
  }

  const result: TaskStatus = {
    openCount: openTasks.length,
    inProgressCount,
    completedCount,
  };
  if (openTasks[0]) result.nextTaskName = openTasks[0];
  if (nextTaskModel) result.nextTaskModel = nextTaskModel;

  return result;
}

export function claimNextTask(cwd: string): {
  status: 'locked' | 'empty';
  taskName?: string;
  model?: string;
  content?: string;
} {
  const tasksDir = join(cwd, '.fabrik', '.tasks');
  const inProgressDir = join(tasksDir, '.in-progress');
  if (!existsSync(tasksDir)) mkdirSync(tasksDir, { recursive: true });
  if (!existsSync(inProgressDir)) mkdirSync(inProgressDir, { recursive: true });

  const openTasks = readdirSync(tasksDir)
    .filter((f) => f.endsWith('.md'))
    .sort();

  if (openTasks.length === 0) {
    return { status: 'empty' };
  }

  const taskName = openTasks[0]!;
  const srcPath = join(tasksDir, taskName);
  const destPath = join(inProgressDir, taskName);

  const contentRes = runCatching(() => readFileSync(srcPath, 'utf8'));
  const content = contentRes.isOk() ? contentRes.value : '';
  const frontmatter = parseTaskFrontmatter(content);
  const config = loadConfig(cwd);
  const model = resolveModel(config, 'build', frontmatter.model);

  const lockRes = runCatching(() => renameSync(srcPath, destPath));
  matchResult(lockRes, {
    ok: () => log.info(`Acquired session task lock for ${taskName}`),
    err: () => log.warn(`Failed to acquire lock for ${taskName}, using direct file reference`),
  });

  const res: {
    status: 'locked' | 'empty';
    taskName?: string;
    model?: string;
    content?: string;
  } = {
    status: 'locked',
    taskName,
    content,
  };
  if (model) res.model = model;

  return res;
}

export function completeTask(
  cwd: string,
  taskName: string,
): { status: 'completed' | 'error'; taskName: string; message?: string } {
  const config = loadConfig(cwd);
  const tasksDir = join(cwd, '.fabrik', '.tasks');
  const inProgressDir = join(tasksDir, '.in-progress');
  const completedDir = join(tasksDir, 'completed');
  if (!existsSync(completedDir)) mkdirSync(completedDir, { recursive: true });

  const inProgressPath = join(inProgressDir, taskName);
  const srcPath = existsSync(inProgressPath) ? inProgressPath : join(tasksDir, taskName);
  const destPath = join(completedDir, taskName);

  if (existsSync(srcPath)) {
    runCatching(() => renameSync(srcPath, destPath));
  }

  if (config.session?.auto_commit) {
    commitTask(cwd, taskName);
  }

  return { status: 'completed', taskName };
}
