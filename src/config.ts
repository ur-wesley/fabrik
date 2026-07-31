import { join } from 'path';
import { readFileSync, existsSync } from 'fs';
import { safeParseYaml } from './yaml';
import { runCatching, matchResult } from '@ur-wesley/ts-prelude/result';
import { logger } from '@ur-wesley/ts-prelude/log';
import * as v from 'valibot';

const log = logger.withTag('fabrik:config');

const ModelConfigSchema = v.object({
  default: v.optional(v.string(), 'anthropic/claude-3-5-sonnet'),
  grill: v.optional(v.string()),
  plan: v.optional(v.string()),
  build: v.optional(v.string()),
});

const SessionConfigSchema = v.object({
  auto_lock: v.optional(v.boolean(), true),
  auto_commit: v.optional(v.boolean(), true),
});

const ToolsConfigSchema = v.object({
  rtk: v.optional(v.boolean(), true),
  engram: v.optional(v.boolean(), true),
  caveman: v.optional(v.boolean(), true),
  ponytail: v.optional(v.boolean(), true),
});

const FabrikConfigSchema = v.object({
  agent: v.optional(v.string(), 'pi'),
  models: v.optional(ModelConfigSchema, {}),
  session: v.optional(SessionConfigSchema, {}),
  tools: v.optional(ToolsConfigSchema, {}),
  skills: v.optional(v.array(v.string()), [
    'caveman',
    'mattpocock-planner',
    'ponytail',
    'rtk-usage',
    'devin-fusion',
  ]),
});

export type FabrikConfig = v.InferOutput<typeof FabrikConfigSchema>;

export const DEFAULT_CONFIG: FabrikConfig = {
  agent: 'pi',
  models: {
    default: 'anthropic/claude-3-5-sonnet',
  },
  session: {
    auto_lock: true,
    auto_commit: true,
  },
  tools: {
    rtk: true,
    engram: true,
    caveman: true,
    ponytail: true,
  },
  skills: ['caveman', 'mattpocock-planner', 'ponytail', 'rtk-usage', 'devin-fusion'],
};

export function loadConfig(cwd: string): FabrikConfig {
  const yamlPath = join(cwd, '.fabrik', 'config.yaml');
  const ymlPath = join(cwd, '.fabrik', 'config.yml');

  let rawConfig: unknown = null;

  if (existsSync(yamlPath)) {
    rawConfig = safeParseYaml(readFileSync(yamlPath, 'utf8'));
  } else if (existsSync(ymlPath)) {
    rawConfig = safeParseYaml(readFileSync(ymlPath, 'utf8'));
  } else {
    return DEFAULT_CONFIG;
  }

  const parsed = runCatching(() => v.parse(FabrikConfigSchema, rawConfig));

  return matchResult(parsed, {
    ok: (config) => config,
    err: (error) => {
      log.warn(`Failed to validate config.yaml: ${String(error)}. Using defaults.`);
      return DEFAULT_CONFIG;
    },
  });
}
