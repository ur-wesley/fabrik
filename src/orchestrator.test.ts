import { describe, test, expect } from 'bun:test';
import { resolveModel, parseTaskFrontmatter, getTaskStatus } from './orchestrator';

describe('Orchestrator Functions', () => {
  test('resolveModel evaluates task model, mode model, and default model hierarchy', () => {
    const config = {
      agent: 'pi',
      models: {
        default: 'anthropic/claude-3-5-sonnet',
        plan: 'openai/o3-mini',
      },
    };

    expect(resolveModel(config, 'plan', 'google/gemini-pro')).toBe('google/gemini-pro');
    expect(resolveModel(config, 'plan')).toBe('openai/o3-mini');
    expect(resolveModel(config, 'build')).toBe('anthropic/claude-3-5-sonnet');
  });

  test('parseTaskFrontmatter extracts model specifier correctly', () => {
    const markdown = `---
model: openai/o3-mini
---
# Task Title
Task description...`;

    const res = parseTaskFrontmatter(markdown);
    expect(res.model).toBe('openai/o3-mini');
  });

  test('getTaskStatus executes without runtime errors', () => {
    const status = getTaskStatus(process.cwd());
    expect(typeof status.openCount).toBe('number');
    expect(typeof status.inProgressCount).toBe('number');
    expect(typeof status.completedCount).toBe('number');
  });
});
