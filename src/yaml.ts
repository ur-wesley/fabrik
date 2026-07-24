import { runCatching, matchResult } from '@ur-wesley/ts-prelude/result';
import { fromNullable, getOrElse, map, type Option } from '@ur-wesley/ts-prelude/option';
import { pipe } from '@ur-wesley/ts-prelude/pipe';

/**
 * Lightweight YAML parser and stringifier powered by @ur-wesley/ts-prelude
 */

export function parseYaml(input: string): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  const lines = input.split(/\r?\n/);

  let currentKey: Option<string> = fromNullable(null);

  for (let i = 0; i < lines.length; i++) {
    const rawLine = lines[i]!;
    const commentIdx = rawLine.indexOf('#');
    const line = (commentIdx !== -1 ? rawLine.slice(0, commentIdx) : rawLine).trimEnd();
    if (!line.trim()) continue;

    const indent = line.search(/\S/);
    const trimmed = line.trim();

    // Array item line
    if (trimmed.startsWith('- ')) {
      const val = parseScalar(trimmed.slice(2).trim());
      map(currentKey, (key) => {
        if (!Array.isArray(result[key])) {
          result[key] = [];
        }
        (result[key] as unknown[]).push(val);
      });
      continue;
    }

    const colonIdx = trimmed.indexOf(':');
    if (colonIdx !== -1) {
      const key = trimmed.slice(0, colonIdx).trim();
      const rawVal = trimmed.slice(colonIdx + 1).trim();

      if (indent === 0) {
        currentKey = fromNullable(key);

        if (rawVal === '') {
          result[key] = {};
        } else {
          result[key] = parseScalar(rawVal);
        }
      } else if (indent > 0) {
        map(currentKey, (ck) => {
          const parent = result[ck];
          if (parent && typeof parent === 'object' && !Array.isArray(parent)) {
            const obj = parent as Record<string, unknown>;
            if (rawVal === '') {
              obj[key] = {};
            } else {
              obj[key] = parseScalar(rawVal);
            }
          }
        });
      }
    }
  }

  return result;
}

export function safeParseYaml(input: string): Record<string, unknown> {
  const parseRes = runCatching(() => parseYaml(input));
  return matchResult(parseRes, {
    ok: (res) => res,
    err: () => ({}),
  });
}

function parseScalar(val: string): unknown {
  if (val === 'true') return true;
  if (val === 'false') return false;
  if (val === 'null' || val === '~') return null;
  if (!isNaN(Number(val)) && val !== '') return Number(val);
  if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
    return val.slice(1, -1);
  }
  return val;
}

export function stringifyYaml(obj: Record<string, unknown>, indentLevel = 0): string {
  const lines: string[] = [];
  const pad = ' '.repeat(indentLevel);

  for (const [key, value] of Object.entries(obj)) {
    if (value === undefined) continue;

    if (Array.isArray(value)) {
      lines.push(`${pad}${key}:`);
      for (const item of value) {
        lines.push(`${pad}  - ${formatScalar(item)}`);
      }
    } else if (value !== null && typeof value === 'object') {
      lines.push(`${pad}${key}:`);
      for (const [subKey, subValue] of Object.entries(value as Record<string, unknown>)) {
        if (subValue !== undefined) {
          lines.push(`${pad}  ${subKey}: ${formatScalar(subValue)}`);
        }
      }
    } else {
      lines.push(`${pad}${key}: ${formatScalar(value)}`);
    }
  }

  return lines.join('\n');
}

function formatScalar(val: unknown): string {
  if (typeof val === 'boolean' || typeof val === 'number') {
    return String(val);
  }
  if (val === null || val === undefined) {
    return 'null';
  }
  const str = String(val);
  if (str.includes(':') || str.includes('#') || str.includes(' ') || str === '') {
    return `"${str.replace(/"/g, '\\"')}"`;
  }
  return str;
}
