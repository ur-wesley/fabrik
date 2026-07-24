import type { Plugin } from "@opencode-ai/plugin";
import type { Event, EventSessionIdle, Message, Part } from "@opencode-ai/sdk";
import { spawn } from "node:child_process";
import { readdir, readFile, rename, writeFile } from "node:fs/promises";
import { join } from "node:path";

interface Task {
  filename: string;
  number: number;
  title: string;
  body: string;
  blockedBy: number[];
}

type LogLevel = "debug" | "info" | "warn" | "error";

const parseBlockedBy = (body: string): number[] => {
  const m = body.match(/##\s*Blocked by\s*\n([\s\S]*?)(?=\n##|\s*$)/i);
  if (!m) return [];
  const text = m[1]!.trim().toLowerCase();
  if (text.startsWith("none") || text === "") return [];
  const nums: number[] = [];
  for (const match of text.matchAll(/#?(\d+)/g)) nums.push(Number(match[1]));
  return nums;
};

const parseTask = (filename: string, body: string): Task | null => {
  const m = filename.match(/^(\d+)-(.+)\.md$/);
  if (!m) return null;
  return {
    filename,
    number: Number(m[1]),
    title: m[2]!,
    body,
    blockedBy: parseBlockedBy(body),
  };
};

const buildWaves = (tasks: Task[]): Task[][] => {
  const waves: Task[][] = [];
  const remaining = new Set(tasks);
  while (remaining.size > 0) {
    const wave: Task[] = [];
    for (const t of remaining) {
      const hasBlocker = t.blockedBy.some((n) =>
        [...remaining].some((r) => r.number === n),
      );
      if (!hasBlocker) wave.push(t);
    }
    if (wave.length === 0) {
      waves.push([...remaining]);
      break;
    }
    waves.push(wave);
    for (const t of wave) remaining.delete(t);
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

export const FabrikPlugin: Plugin = async ({ client, $, directory }) => {
  const tasksDir = join(directory, ".fabrik", ".tasks");
  const completedDir = join(tasksDir, "completed");

  const log = async (level: LogLevel, message: string): Promise<void> => {
    await client.app.log({ body: { service: "fabrik", level, message } });
  };

  const loadTasks = async (): Promise<Task[]> => {
    let files: string[];
    try {
      files = await readdir(tasksDir);
    } catch {
      return [];
    }
    const tasks: Task[] = [];
    for (const f of files) {
      if (!f.endsWith(".md")) continue;
      const body = await readFile(join(tasksDir, f), "utf8");
      const t = parseTask(f, body);
      if (t) tasks.push(t);
    }
    return tasks.sort((a, b) => a.number - b.number);
  };

  const archiveTask = async (task: Task): Promise<void> => {
    await rename(join(tasksDir, task.filename), join(completedDir, task.filename));
  };

  const markStuck = async (task: Task, reason: string): Promise<void> => {
    const path = join(tasksDir, task.filename);
    const body = await readFile(path, "utf8");
    await writeFile(
      path,
      `${body}\n\n## Discoveries / Obstacles\n\n- runner exit non-zero\n- ${reason}\n`,
    );
  };

  const runTask = (
    task: Task,
  ): Promise<{ ok: boolean; code: number | null }> => {
    return new Promise((resolve) => {
      const child = spawn("opencode", ["run", task.body], {
        cwd: directory,
        stdio: "inherit",
        windowsHide: true,
      });
      child.on("error", () => resolve({ ok: false, code: -1 }));
      child.on("close", (code: number | null) => resolve({ ok: code === 0, code }));
    });
  };

  const commitWave = async (wave: Task[]): Promise<void> => {
    const titles = wave.map((t) => t.title).join(", ");
    const summary = wave.map((t) => `- ${t.filename}`).join("\n");
    const msg = `feat: ${titles}\n\n${summary}`;
    await $`git add -A`.cwd(directory);
    await $`git commit -m ${msg}`.cwd(directory);
  };

  const runLoop = async (): Promise<void> => {
    await log("info", "fabrik: loop starting");
    while (true) {
      const tasks = await loadTasks();
      if (tasks.length === 0) {
        await log("info", "fabrik: no open tasks remaining — done");
        return;
      }
      const waves = buildWaves(tasks);
      const wave = waves[0];
      if (!wave || wave.length === 0) break;

      const next = wave[0]!;
      await log("info", `fabrik: running ${next.filename}`);

      const result = await runTask(next);
      if (!result.ok) {
        await markStuck(next, `exit code ${result.code}`);
        await log("warn", `fabrik: ${next.filename} failed — paused`);
        return;
      }

      await archiveTask(next);

      const refreshed = await loadTasks();
      const waveStillOpen = wave.filter((w) =>
        refreshed.some((t) => t.filename === w.filename),
      );
      if (waveStillOpen.length === 0) {
        try {
          await commitWave(wave);
          await log("info", `fabrik: wave committed (${wave.length} tasks)`);
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
