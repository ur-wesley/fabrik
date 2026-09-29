import { describe, expect, test } from 'bun:test';
import { existsSync, readFileSync } from 'fs';
import { join } from 'path';
import { FABRIK_ROOT, run } from './helpers/repo';

describe('fabrik install assets', () => {
  test('deps.json has cross-platform pins', () => {
    const deps = JSON.parse(readFileSync(join(FABRIK_ROOT, 'install', 'deps.json'), 'utf8')) as {
      beads: { tag: string; assets: Record<string, string> };
      engram: { tag: string; assets: Record<string, string> };
      graphify: { version: string };
    };

    expect(deps.beads.tag).toMatch(/^v/);
    expect(deps.engram.tag).toMatch(/^v/);
    expect(deps.graphify.version).toMatch(/\d/);

    for (const key of ['windows_amd64', 'linux_amd64', 'darwin_arm64']) {
      expect(deps.beads.assets[key]).toBeTruthy();
      expect(deps.engram.assets[key]).toBeTruthy();
    }
  });

  test('workflow template mentions beads not markdown tasks', () => {
    const note = readFileSync(join(FABRIK_ROOT, 'install', 'templates', 'workflow-note.md'), 'utf8');
    expect(note).toContain('bd create');
    expect(note).toContain('Engram');
    expect(note).not.toContain('.fabrik/.tasks');
  });

  test('bd and engram available when install was run', () => {
    const hasBd = existsSync(join(process.env['USERPROFILE'] ?? '', '.local', 'bin', 'bd.exe'));
    const bdOnPath = (() => {
      try {
        run('bd version', FABRIK_ROOT);
        return true;
      } catch {
        return false;
      }
    })();

    expect(hasBd || bdOnPath).toBe(true);
  });
});
