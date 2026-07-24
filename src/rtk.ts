import { execSync } from 'child_process';
import { runCatching, matchResult } from '@ur-wesley/ts-prelude/result';
import { logger } from '@ur-wesley/ts-prelude/log';

const log = logger.withTag('fabrik:rtk');

/**
 * Filter CLI output through RTK (Rust Token Killer) if available,
 * reducing log noise and saving 60-90% LLM token usage.
 */
export function wrapWithRtk(command: string, cwd: string): string {
  const rtkResult = runCatching(() => execSync(`rtk exec ${command}`, { cwd, encoding: 'utf8' }));

  return matchResult(rtkResult, {
    ok: (output) => output,
    err: () => {
      // Fallback to raw execution if RTK is unavailable
      const directResult = runCatching(() => execSync(command, { cwd, encoding: 'utf8' }));
      return matchResult(directResult, {
        ok: (output) => output,
        err: (e) => {
          log.debug(`Command execution failed: ${String(e)}`);
          return e instanceof Error ? e.message : String(e);
        },
      });
    },
  });
}
