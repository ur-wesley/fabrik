import { describe, test, expect } from 'bun:test';
import { parseYaml, stringifyYaml } from './yaml';

describe('YAML Parser & Stringifier', () => {
  test('parses scalars and lists correctly', () => {
    const input = `
agent: pi
rtk: true
skills:
  - ponytail
  - grill-with-docs
`;
    const parsed = parseYaml(input);
    expect(parsed.agent).toBe('pi');
    expect(parsed.rtk).toBe(true);
    expect(parsed.skills).toEqual(['ponytail', 'grill-with-docs']);
  });

  test('parses nested objects correctly', () => {
    const input = `
models:
  default: anthropic/claude-3-5-sonnet
  plan: openai/o3-mini
session:
  auto_lock: true
`;
    const parsed = parseYaml(input);
    expect(parsed.models).toEqual({
      default: 'anthropic/claude-3-5-sonnet',
      plan: 'openai/o3-mini',
    });
    expect(parsed.session).toEqual({
      auto_lock: true,
    });
  });

  test('stringifies object to clean YAML string', () => {
    const obj = {
      agent: 'pi',
      models: { default: 'claude-3-5-sonnet' },
      skills: ['ponytail'],
    };
    const output = stringifyYaml(obj);
    expect(output).toContain('agent: pi');
    expect(output).toContain('models:');
    expect(output).toContain('  default: claude-3-5-sonnet');
    expect(output).toContain('skills:');
    expect(output).toContain('  - ponytail');
  });
});
