import { cpSync, existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync, mkdirSync } from 'fs';
import { tmpdir } from 'os';
import { join, resolve } from 'path';
import { execSync, spawnSync } from 'child_process';

export const FABRIK_ROOT = resolve(import.meta.dir, '../..');
export const FIXTURE_ROOT = resolve(import.meta.dir, '../fixtures/sample-app');

export function requireBd(): void {
  try {
    execSync('bd version', { stdio: 'ignore' });
  } catch {
    throw new Error('bd not on PATH — run install/setup.ps1 first');
  }
}

export function run(cmd: string, cwd: string): string {
  return execSync(cmd, { cwd, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim();
}

export function runOk(cmd: string, cwd: string): void {
  execSync(cmd, { cwd, stdio: 'inherit' });
}

export function createTestRepo(): string {
  const dir = mkdtempSync(join(tmpdir(), 'fabrik-e2e-'));
  cpSync(FIXTURE_ROOT, dir, { recursive: true });

  runOk('git init', dir);
  runOk('git config user.email "fabrik-e2e@test.local"', dir);
  runOk('git config user.name "Fabrik E2E"', dir);
  runOk('git add -A', dir);
  runOk('git commit -m "chore: initial fixture"', dir);

  return dir;
}

export function initFabrikRepo(repoPath: string): void {
  if (!existsSync(join(repoPath, '.beads'))) {
    run(
      'bd init --non-interactive --skip-agents --skip-hooks --agents-profile minimal -q',
      repoPath,
    );
  }

  const fabrikDir = join(repoPath, '.fabrik');
  mkdirSync(join(fabrikDir, 'docs'), { recursive: true });
  mkdirSync(join(fabrikDir, 'specs'), { recursive: true });
  mkdirSync(join(fabrikDir, 'styleguide'), { recursive: true });

  const configPath = join(fabrikDir, 'config.yaml');
  if (!existsSync(configPath)) {
    const template = join(FABRIK_ROOT, '.fabrik', 'config.yaml');
    if (existsSync(template)) {
      cpSync(template, configPath);
    } else {
      writeFileSync(
        configPath,
        `agent: pi
models:
  default: inherit
session:
  auto_lock: true
  auto_commit: true
tools:
  rtk: true
  engram: true
`,
        'utf8',
      );
    }
  }

  bootstrapFabrikDir(repoPath);

  const workflowTemplate = readFileSync(
    join(FABRIK_ROOT, 'install', 'templates', 'workflow-note.md'),
    'utf8',
  );
  const agentsPath = join(repoPath, 'AGENTS.md');
  const agentsBody = existsSync(agentsPath) ? readFileSync(agentsPath, 'utf8') : '';
  if (!agentsBody.includes('Fabrik workflow')) {
    const block = `\n## Fabrik workflow\n\n${workflowTemplate}`;
    writeFileSync(agentsPath, agentsBody + block, 'utf8');
  }
}

export function bootstrapFabrikDir(repoPath: string): void {
  const templateDir = join(FABRIK_ROOT, '.fabrik');
  const targetDir = join(repoPath, '.fabrik');
  const files = [
    'PROMPT_plan.md',
    'PROMPT_build.md',
    'loop.ps1',
    'loop.sh',
    'fabrik.ps1',
    'fabrik.sh',
    'AGENTS.md',
    'setup.ps1',
    'setup.sh',
    'CONTEXT.md',
  ];
  for (const file of files) {
    const src = join(templateDir, file);
    const dst = join(targetDir, file);
    if (existsSync(src) && !existsSync(dst)) {
      cpSync(src, dst);
    }
  }
  const styleguide = join(targetDir, 'styleguide', 'STYLEGUIDE.md');
  if (!existsSync(styleguide) && existsSync(join(templateDir, 'styleguide', 'STYLEGUIDE.md'))) {
    cpSync(join(templateDir, 'styleguide', 'STYLEGUIDE.md'), styleguide);
  }
}

export function createIssue(
  repoPath: string,
  title: string,
  description = '',
): { id: string; title: string } {
  const safeTitle = title.replace(/"/g, '\\"');
  const descArg = description ? ` --description="${description.replace(/"/g, '\\"')}"` : '';
  const out = run(`bd create "${safeTitle}" -t task -p 2 --json${descArg}`, repoPath);
  return JSON.parse(out) as { id: string; title: string };
}

export function addDep(repoPath: string, blockedId: string, blockerId: string): void {
  run(`bd dep add ${blockedId} ${blockerId}`, repoPath);
}

export function readText(path: string): string {
  return readFileSync(path, 'utf8');
}

export function assertPs1Parses(scriptPath: string): void {
  const normalized = scriptPath.replace(/\\/g, '/');
  const escaped = normalized.replace(/'/g, "''");
  const script = `$err=$null;$tok=$null;[void][System.Management.Automation.Language.Parser]::ParseFile('${escaped}', [ref]$tok, [ref]$err); if ($err) { exit 1 }`;
  const result = spawnSync('powershell.exe', ['-NoProfile', '-Command', script], { stdio: 'pipe' });
  if (result.status !== 0) {
    const detail = result.stderr?.toString() || result.stdout?.toString() || 'unknown';
    throw new Error(`PS1 parse failed: ${scriptPath}\n${detail}`);
  }
}

export function destroyRepo(repoPath: string): void {
  try {
    run('bd dolt stop', repoPath);
  } catch {
    /* dolt may not be running */
  }
  for (let attempt = 0; attempt < 5; attempt++) {
    try {
      rmSync(repoPath, { recursive: true, force: true, maxRetries: 3, retryDelay: 200 });
      return;
    } catch {
      Bun.sleepSync(300);
    }
  }
}
