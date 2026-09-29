import { describe, test, expect } from 'bun:test';
import { resolveModel, getTaskStatus } from './orchestrator';
import { DEFAULT_CONFIG, type FabrikConfig } from './config';

describe('Orchestrator Functions', () => {
  test('resolveModel evaluates mode model and default model hierarchy', () => {
    const config: FabrikConfig = {
      ...DEFAULT_CONFIG,
      models: {
        default: 'anthropic/claude-3-5-sonnet',
        plan: 'openai/o3-mini',
      },
    };

    expect(resolveModel(config, 'plan', 'google/gemini-pro')).toBe('google/gemini-pro');
    expect(resolveModel(config, 'plan')).toBe('openai/o3-mini');
    expect(resolveModel(config, 'build')).toBe('anthropic/claude-3-5-sonnet');
  });

  test('getTaskStatus executes without runtime errors', () => {
    const status = getTaskStatus(process.cwd());
    expect(typeof status.openCount).toBe('number');
    expect(typeof status.inProgressCount).toBe('number');
    expect(typeof status.completedCount).toBe('number');
  });
});
