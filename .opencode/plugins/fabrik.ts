import type { Plugin } from "@opencode-ai/plugin";
import type { Event, EventSessionIdle, Message, Part } from "@opencode-ai/sdk";
import { spawn } from "node:child_process";
import { writeFile } from "node:fs/promises";
import { join } from "node:path";

interface BeadsIssue {
  id: string;
  title: string;
  description?: string;
  status: string;
}

type LogLevel = "debug" | "info" | "warn" | "error";

const runBdJson = async <T>(directory: string, args: string[]): Promise<T | null> => {
  return new Promise((resolve) => {
    const child = spawn("bd", args, {
      cwd: directory,
      stdio: ["ignore", "pipe", "pipe"],
      windowsHide: true,
    });
    let stdout = "";
    child.stdout.on("data", (chunk: Buffer) => {
      stdout += chunk.toString();
    });
    child.on("error", () => resolve(null));
    child.on("close", (code) => {
      if (code !== 0 || !stdout.trim()) {
        resolve(null);
        return;
      }
      try {
        resolve(JSON.parse(stdout) as T);
      } catch {
        resolve(null);
      }
    });
  });
};

const listOpen = async (directory: string): Promise<BeadsIssue[]> => {
  const result = await runBdJson<BeadsIssue[]>(directory, ["list", "--status=open", "--json"]);
  return result ?? [];
};

const listBlockers = async (directory: string, id: string): Promise<string[]> => {
  const result = await runBdJson<Array<{ id: string }>>(directory, ["dep", "list", id, "--json"]);
  return (result ?? []).map((d) => d.id);
};

const buildWaves = async (directory: string, issues: BeadsIssue[]): Promise<BeadsIssue[][]> => {
  const waves: BeadsIssue[][] = [];
  const remaining = new Map(issues.map((i) => [i.id, i]));
  const blockers = new Map<string, string[]>();
  for (const issue of issues) {
    blockers.set(issue.id, await listBlockers(directory, issue.id));
  }

  while (remaining.size > 0) {
    const wave: BeadsIssue[] = [];
    for (const issue of remaining.values()) {
      const deps = blockers.get(issue.id) ?? [];
      const hasOpenBlocker = deps.some((depId) => remaining.has(depId));
      if (!hasOpenBlocker) wave.push(issue);
    }
    if (wave.length === 0) {
      waves.push([...remaining.values()]);
      break;
    }
    waves.push(wave);
    for (const issue of wave) remaining.delete(issue.id);
  }
  return waves;
};

const lastUserText = (msgs: ReadonlyArray<{ info: Message; parts: Part[] }>): string => {
  for (let i = msgs.length - 1; i >= 0; i--) {
    const m = msgs[i]!;
    if (m.info.role === "user") {
      return m.parts.map((p) => ("text" in p ? (p.text ?? "") : "")).join("");
    }
  }
  return "";
};

const runBuildPrompt = (
  directory: string,
  issue: BeadsIssue,
): Promise<{ ok: boolean; code: number | null }> => {
  return new Promise((resolve) => {
    const promptPath = join(directory, ".fabrik", "PROMPT_build.md");
    const child = spawn("opencode", ["run", "--agent", "builder", `@${promptPath}\n\nIssue: ${issue.id} — ${issue.title}\n\n${issue.description ?? ""}`], {
      cwd: directory,
      stdio: "inherit",
      windowsHide: true,
      shell: true,
    });
    child.on("error", () => resolve({ ok: false, code: -1 }));
    child.on("close", (code: number | null) => resolve({ ok: code === 0, code }));
  });
};

const closeIssue = async (directory: string, id: string): Promise<boolean> => {
  return new Promise((resolve) => {
    const child = spawn("bd", ["close", id, "--reason=Completed by fabrik loop"], {
      cwd: directory,
      stdio: "ignore",
      windowsHide: true,
    });
    child.on("error", () => resolve(false));
    child.on("close", (code) => resolve(code === 0));
  });
};

const annotateIssue = async (directory: string, issue: BeadsIssue, reason: string): Promise<void> => {
  const path = join(directory, ".fabrik", "state.md");
  await writeFile(
    path,
    `# Fabrik loop obstacle\n\nIssue: ${issue.id} — ${issue.title}\nReason: ${reason}\n`,
    "utf8",
  );
};

export const FabrikPlugin: Plugin = async ({ client, $, directory }) => {
  const log = async (level: LogLevel, message: string): Promise<void> => {
    await client.app.log({ body: { service: "fabrik", level, message } });
  };

  const commitWave = async (wave: BeadsIssue[]): Promise<void> => {
    const titles = wave.map((t) => t.title).join(", ");
    const summary = wave.map((t) => `- ${t.id}: ${t.title}`).join("\n");
    const msg = `feat: ${titles}\n\n${summary}`;
    await $`git add -A`.cwd(directory);
    await $`git commit -m ${msg}`.cwd(directory);
  };

  const runLoop = async (): Promise<void> => {
    await log("info", "fabrik: loop starting");
    while (true) {
      const issues = await listOpen(directory);
      if (issues.length === 0) {
        await log("info", "fabrik: no open Beads issues — done");
        return;
      }

      const waves = await buildWaves(directory, issues);
      const wave = waves[0];
      if (!wave || wave.length === 0) break;

      const next = wave[0]!;
      await log("info", `fabrik: running ${next.id} — ${next.title}`);

      const result = await runBuildPrompt(directory, next);
      if (!result.ok) {
        await annotateIssue(directory, next, `exit code ${result.code}`);
        await log("warn", `fabrik: ${next.id} failed — paused`);
        return;
      }

      const closed = await closeIssue(directory, next.id);
      if (!closed) {
        await annotateIssue(directory, next, "bd close failed");
        await log("warn", `fabrik: could not close ${next.id}`);
        return;
      }

      const refreshed = await listOpen(directory);
      const waveStillOpen = wave.filter((w) => refreshed.some((t) => t.id === w.id));
      if (waveStillOpen.length === 0) {
        try {
          await commitWave(wave);
          await log("info", `fabrik: wave committed (${wave.length} issues)`);
        } catch (e) {
          await log("error", `fabrik: commit failed: ${(e as Error).message}`);
        }
      }
    }
  };

  return {
    event: async ({ event }: { event: Event }) => {
      const idle = event as EventSessionIdle;
      if (idle.type !== "session.idle") return;

      const sessionID = idle.properties.sessionID;

      let msgs: ReadonlyArray<{ info: Message; parts: Part[] }>;
      try {
        const result = await client.session.messages({ path: { id: sessionID } });
        msgs = (result.data ?? []) as ReadonlyArray<{ info: Message; parts: Part[] }>;
      } catch {
        return;
      }

      if (!lastUserText(msgs).includes("/exit")) return;

      try {
        await runLoop();
      } catch (e) {
        await log("error", `fabrik: loop crashed: ${(e as Error).message}`);
      }
    },
  };
};
