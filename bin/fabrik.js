#!/usr/bin/env node
// Fabrik binary launcher — no postinstall, no download.
// Resolves the platform-specific optionalDependency and spawns it.
// Works under npm, bun, pnpm (global or local installs).
"use strict";

const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const PKG_SCOPE = "@ur-wesley";

function npmTarget(platform, arch) {
  // Node naming: win32/x64, darwin/x64+arm64, linux/x64+arm64
  if (platform === "win32" && arch === "x64") return "win32-x64";
  if (platform === "darwin" && arch === "x64") return "darwin-x64";
  if (platform === "darwin" && arch === "arm64") return "darwin-arm64";
  if (platform === "linux" && arch === "x64") return "linux-x64";
  if (platform === "linux" && arch === "arm64") return "linux-arm64";
  return null;
}

function binaryName(platform) {
  return platform === "win32" ? "fabrik.exe" : "fabrik";
}

function tryPaths(target, binary) {
  const candidates = [];
  // 1. Explicit override for testing / custom installs.
  if (process.env.FABRIK_BINARY) candidates.push(process.env.FABRIK_BINARY);
  // 2. Repo checkout layout: <repo>/bin/fabrik.js -> <repo>/npm/fabrik-<target>/bin/<binary>
  candidates.push(path.resolve(__dirname, "..", "npm", `fabrik-${target}`, "bin", binary));
  // 2b. Local Go build: <repo>/bin/fabrik.js -> <repo>/cli/fabrik[.exe] (`npm run cli:build` / `go build`)
  candidates.push(path.resolve(__dirname, "..", "cli", binary));
  candidates.push(path.resolve(__dirname, "..", "cli", "fabrik"));
  candidates.push(path.resolve(__dirname, "..", "cli", "fabrik.exe"));
  // 3. Installed layout: .../node_modules/@ur-wesley/fabrik/bin -> .../node_modules/@ur-wesley/fabrik-<target>/bin
  candidates.push(path.resolve(__dirname, "..", "..", `fabrik-${target}`, "bin", binary));
  // 4. require.resolve (handles pnpm / hoisted layouts).
  try {
    candidates.push(require.resolve(`${PKG_SCOPE}/fabrik-${target}/bin/${binary}`));
  } catch {
    // optional dep for another platform — not installed, ignore.
  }
  return candidates;
}

function main() {
  const target = npmTarget(process.platform, process.arch);
  if (!target) {
    console.error(
      `fabrik: unsupported platform ${process.platform}/${process.arch}. Supported: win32-x64, linux-x64, linux-arm64, darwin-x64, darwin-arm64.`
    );
    process.exit(1);
  }
  const binary = binaryName(process.platform);
  const candidates = tryPaths(target, binary);
  const found = candidates.find((p) => {
    try {
      return p && fs.existsSync(p) && fs.statSync(p).isFile();
    } catch {
      return false;
    }
  });
  if (!found) {
    console.error(`fabrik: binary not found for ${target} (${binary}).`);
    console.error(`Looked in:\n  - ${candidates.filter(Boolean).join("\n  - ")}`);
    console.error(
      `This usually means the optionalDependency ${PKG_SCOPE}/fabrik-${target} was not installed. ` +
        `Reinstall with npm (do not use --no-optional) or bun, e.g. \`npm i -g ${PKG_SCOPE}/fabrik\`.`
    );
    process.exit(1);
  }
  const args = process.argv.slice(2);
  const res = spawnSync(found, args, { stdio: "inherit", windowsHide: true });
  if (res.error) {
    console.error(`fabrik: failed to launch ${found}: ${res.error.message}`);
    process.exit(1);
  }
  if (res.signal) {
    // Match shell behavior: 128 + signal number is not portable here; exit 1 after forwarding.
    process.exit(1);
  }
  process.exit(res.status ?? 0);
}

main();
