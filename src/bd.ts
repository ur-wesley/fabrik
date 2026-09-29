import { execSync } from 'child_process';
import { runCatching, matchResult } from '@ur-wesley/ts-prelude/result';

export interface BeadsIssue {
  id: string;
  title: string;
  description?: string;
  status: string;
  priority?: number;
  issue_type?: string;
}

function runBdJson<T>(cwd: string, args: string[]): T | null {
  const res = runCatching(() =>
    execSync(`bd ${args.join(' ')}`, {
      cwd,
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'pipe'],
    }),
  );
  return matchResult(res, {
    ok: (stdout) => {
      const trimmed = stdout.trim();
      if (!trimmed) return null;
      return JSON.parse(trimmed) as T;
    },
    err: () => null,
  });
}

export function listIssues(
  cwd: string,
  status: 'open' | 'in_progress' | 'closed',
  limit = 0,
): BeadsIssue[] {
  const args = ['list', `--status=${status}`, '--json'];
  if (limit > 0) args.push(`--limit=${limit}`);
  const result = runBdJson<BeadsIssue[]>(cwd, args);
  return result ?? [];
}

export function getReadyIssues(cwd: string): BeadsIssue[] {
  const result = runBdJson<BeadsIssue[]>(cwd, ['ready', '--json']);
  return result ?? [];
}

export function claimReadyIssue(cwd: string): BeadsIssue | null {
  const result = runBdJson<BeadsIssue | BeadsIssue[]>(cwd, ['ready', '--claim', '--json']);
  if (!result) return null;
  if (Array.isArray(result)) return result[0] ?? null;
  return result;
}

export function closeIssue(cwd: string, id: string, reason = 'Completed'): boolean {
  const res = runCatching(() =>
    execSync(`bd close ${id} --reason="${reason.replace(/"/g, '\\"')}"`, {
      cwd,
      stdio: 'ignore',
    }),
  );
  return res.isOk();
}

export function getBlockers(cwd: string, id: string): BeadsIssue[] {
  const result = runBdJson<BeadsIssue[]>(cwd, ['dep', 'list', id, '--json']);
  return result ?? [];
}

export function beadsAvailable(cwd: string): boolean {
  const res = runCatching(() =>
    execSync('bd where', { cwd, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }),
  );
  return res.isOk();
}
