import { afterEach, beforeAll, describe, expect, test } from 'bun:test';

const e2eTimeout = 60_000;
import { existsSync } from 'fs';
import { join } from 'path';
import { claimNextTask, completeTask, getTaskStatus } from '../src/orchestrator';
import {
  addDep,
  createIssue,
  createTestRepo,
  destroyRepo,
  assertPs1Parses,
  FABRIK_ROOT,
  initFabrikRepo,
  readText,
  requireBd,
  run,
  runOk,
} from './helpers/repo';

let repoPath = '';

beforeAll(() => {
  requireBd();
});

afterEach(() => {
  if (repoPath) {
    destroyRepo(repoPath);
    repoPath = '';
  }
});

describe('fabrik e2e workflow', () => {
  test.serial('init creates beads, fabrik config, and workflow in AGENTS.md', () => {
    repoPath = createTestRepo();
    initFabrikRepo(repoPath);

    expect(existsSync(join(repoPath, '.beads'))).toBe(true);
    expect(existsSync(join(repoPath, '.fabrik', 'config.yaml'))).toBe(true);
    expect(existsSync(join(repoPath, '.fabrik', 'PROMPT_plan.md'))).toBe(true);
    expect(existsSync(join(repoPath, '.fabrik', 'PROMPT_build.md'))).toBe(true);

    const agents = readText(join(repoPath, 'AGENTS.md'));
    expect(agents).toContain('Fabrik workflow');
    expect(agents).toContain('bun test');

    const planPrompt = readText(join(repoPath, '.fabrik', 'PROMPT_plan.md'));
    expect(planPrompt).toContain('bd create');
    expect(planPrompt).not.toContain('.fabrik/.tasks/');

    run('bd where', repoPath);
  }, e2eTimeout);

  test.serial('orchestrator claims ready issue and completes it', () => {
    repoPath = createTestRepo();
    initFabrikRepo(repoPath);

    createIssue(repoPath, 'First task', 'Do the first thing');
    createIssue(repoPath, 'Second task', 'Do the second thing');

    const before = getTaskStatus(repoPath);
    expect(before.openCount).toBe(2);
    expect(before.nextTaskId).toBeDefined();

    const claimed = claimNextTask(repoPath);
    expect(claimed.status).toBe('locked');
    expect(claimed.taskId).toBeDefined();
    expect(claimed.taskName).toBeDefined();

    const inProgress = getTaskStatus(repoPath);
    expect(inProgress.inProgressCount).toBe(1);
    expect(inProgress.openCount).toBe(1);

    const done = completeTask(repoPath, claimed.taskId!);
    expect(done.status).toBe('completed');

    const after = getTaskStatus(repoPath);
    expect(after.completedCount).toBe(1);
    expect(after.openCount).toBe(1);
    expect(after.inProgressCount).toBe(0);

    const shown = JSON.parse(run(`bd show ${claimed.taskId} --json`, repoPath)) as Array<{
      status: string;
    }>;
    expect(shown[0]?.status).toBe('closed');
  }, e2eTimeout);

  test.serial('beads dependencies gate bd ready queue', () => {
    repoPath = createTestRepo();
    initFabrikRepo(repoPath);

    const blocker = createIssue(repoPath, 'Blocker task');
    const blocked = createIssue(repoPath, 'Blocked task');
    addDep(repoPath, blocked.id, blocker.id);

    const ready = JSON.parse(run('bd ready --json', repoPath)) as Array<{ id: string }>;
    expect(ready.length).toBe(1);
    expect(ready[0]?.id).toBe(blocker.id);

    const claimed = claimNextTask(repoPath);
    expect(claimed.taskId).toBe(blocker.id);
    completeTask(repoPath, blocker.id);

    const readyAfter = JSON.parse(run('bd ready --json', repoPath)) as Array<{ id: string }>;
    expect(readyAfter.length).toBe(1);
    expect(readyAfter[0]?.id).toBe(blocked.id);
  }, e2eTimeout);

  test.serial('sample app backpressure scripts pass', () => {
    repoPath = createTestRepo();
    initFabrikRepo(repoPath);

    runOk('bun test src/', repoPath);
    runOk('bun run lint', repoPath);
    runOk('bun run build', repoPath);
  }, e2eTimeout);

  test.serial('fabrik orchestrator scripts parse on windows', () => {
    if (process.platform !== 'win32') return;

    const fabrikDir = join(FABRIK_ROOT, '.fabrik');
    const scripts = ['loop.ps1', 'fabrik.ps1', 'setup.ps1'];
    for (const script of scripts) {
      const path = join(fabrikDir, script);
      expect(existsSync(path)).toBe(true);
      assertPs1Parses(path);
    }
  }, e2eTimeout);
});
