import { beforeAll, describe, expect, test } from 'bun:test';
import { readFile, stat } from 'node:fs/promises';
import { dirname, join } from 'path';
import { fileURLToPath } from 'url';

const projectRoot = join(dirname(fileURLToPath(import.meta.url)), '..');

interface PackageJson {
  scripts: Record<string, string>;
}

async function pathExists(target: string): Promise<boolean> {
  try {
    await stat(target);
    return true;
  } catch {
    return false;
  }
}

async function readJson(target: string): Promise<unknown> {
  return JSON.parse(await readFile(target, 'utf8'));
}

describe('Toolchain', () => {
  let scripts: Record<string, string>;

  beforeAll(async () => {
    const pkg = (await readJson(join(projectRoot, 'package.json'))) as PackageJson;
    scripts = pkg.scripts;
  });

  test('.oxlintrc.json exists and parses as JSON', async () => {
    const configPath = join(projectRoot, '.oxlintrc.json');
    expect(await pathExists(configPath)).toBe(true);
    expect(await readJson(configPath)).toBeDefined();
  });

  test('.oxfmtrc.json exists and parses as JSON', async () => {
    const configPath = join(projectRoot, '.oxfmtrc.json');
    expect(await pathExists(configPath)).toBe(true);
    expect(await readJson(configPath)).toBeDefined();
  });

  test('biome.json does NOT exist at repo root', async () => {
    expect(await pathExists(join(projectRoot, 'biome.json'))).toBe(false);
  });

  describe('package.json scripts', () => {
    test('lint references oxlint or oxfmt', () => {
      const script = scripts['lint'];
      expect(script).toBeDefined();
      if (script !== undefined) {
        expect(script).toMatch(/oxlint|oxfmt/);
      }
    });

    test('lint:fix references oxlint or oxfmt', () => {
      const script = scripts['lint:fix'];
      expect(script).toBeDefined();
      if (script !== undefined) {
        expect(script).toMatch(/oxlint|oxfmt/);
      }
    });

    test('format references oxlint or oxfmt', () => {
      const script = scripts['format'];
      expect(script).toBeDefined();
      if (script !== undefined) {
        expect(script).toMatch(/oxlint|oxfmt/);
      }
    });
  });
});
