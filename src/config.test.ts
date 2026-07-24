import { expect, test, describe } from 'bun:test';
import { loadConfig, DEFAULT_CONFIG } from './config';

describe('Config with @ur-wesley/ts-prelude', () => {
  test('returns default config when config file is missing', () => {
    const config = loadConfig('/non/existent/path');
    expect(config.agent).toBe('pi');
    expect(config.tools?.rtk).toBe(true);
    expect(config.session?.auto_lock).toBe(true);
  });
});
