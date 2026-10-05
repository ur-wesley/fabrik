// Keep versions in sync: root package.json is the source of truth.
// Mirrors into cli/package.json (Go ldflags version), cli/internal/version/package.json
// (embedded fallback) and all npm platform packages
// (including root optionalDependencies ranges).
// Usage: node scripts/sync-versions.mjs [--check] [--set X.Y.Z]
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, "..");
const rootPkgPath = path.join(ROOT, "package.json");
const cliPkgPath = path.join(ROOT, "cli", "package.json");
const cliEmbeddedPkgPath = path.join(ROOT, "cli", "internal", "version", "package.json");
const npmDir = path.join(ROOT, "npm");

const PLATFORMS = [
  "fabrik-win32-x64",
  "fabrik-linux-x64",
  "fabrik-linux-arm64",
  "fabrik-darwin-x64",
  "fabrik-darwin-arm64",
];

function readJson(p) {
  return JSON.parse(fs.readFileSync(p, "utf8"));
}

function writeJson(p, obj) {
  fs.writeFileSync(p, JSON.stringify(obj, null, 2) + "\n");
}

function main() {
  const args = process.argv.slice(2);
  const check = args.includes("--check");
  const setIdx = args.indexOf("--set");
  const setVersion = setIdx !== -1 ? args[setIdx + 1] : null;
  if (setIdx !== -1 && !setVersion) {
    console.error("sync-versions: --set requires a version, e.g. --set 1.2.3");
    process.exit(1);
  }

  const rootPkg = readJson(rootPkgPath);
  const sourceVersion = setVersion || rootPkg.version;
  if (!/^\d+\.\d+\.\d+(-.+)?$/.test(sourceVersion)) {
    console.error(`sync-versions: invalid version ${sourceVersion}`);
    process.exit(1);
  }

  const problems = [];
  const updates = [];

  function ensureVersion(label, pkg) {
    if (pkg.version !== sourceVersion) {
      problems.push(`${label}: ${pkg.version} != ${sourceVersion}`);
      pkg.version = sourceVersion;
      updates.push(label);
    }
  }

  if (rootPkg.version !== sourceVersion) {
    problems.push(`root: ${rootPkg.version} != ${sourceVersion}`);
    rootPkg.version = sourceVersion;
    updates.push("root");
  }

  rootPkg.optionalDependencies = rootPkg.optionalDependencies || {};
  for (const plat of PLATFORMS) {
    const key = `@ur-wesley/${plat}`;
    if (rootPkg.optionalDependencies[key] !== sourceVersion) {
      problems.push(`root optionalDependencies[${key}]: ${rootPkg.optionalDependencies[key]} != ${sourceVersion}`);
      rootPkg.optionalDependencies[key] = sourceVersion;
      updates.push(`root optionalDependencies[${key}]`);
    }
  }
  for (const field of ["postinstall", "install", "preinstall", "preprepare"]) {
    if (rootPkg.scripts && rootPkg.scripts[field]) {
      problems.push(`root scripts.${field} must not exist (no postinstall allowed)`);
    }
  }

  const cliPkg = readJson(cliPkgPath);
  ensureVersion("cli/package.json", cliPkg);

  const cliEmbeddedPkg = readJson(cliEmbeddedPkgPath);
  ensureVersion("cli/internal/version/package.json", cliEmbeddedPkg);

  const platPkgs = PLATFORMS.map((plat) => {
    const p = path.join(npmDir, plat, "package.json");
    return { label: `npm/${plat}/package.json`, path: p, pkg: readJson(p) };
  });
  for (const { label, pkg } of platPkgs) ensureVersion(label, pkg);

  const expectedRepo = "https://github.com/ur-wesley/fabrik";
  function ensureRepo(label, pkg) {
    const url = pkg.repository && pkg.repository.url;
    if (url !== expectedRepo) {
      problems.push(`${label}: repository.url is ${JSON.stringify(url)} != ${JSON.stringify(expectedRepo)} (required for npm provenance)`);
      pkg.repository = { type: "git", url: expectedRepo };
      updates.push(label + " repository");
    }
  }
  ensureRepo("root package.json", rootPkg);
  for (const { label, pkg } of platPkgs) ensureRepo(label, pkg);

  if (check) {
    if (problems.length) {
      console.error("version mismatch:\n  - " + problems.join("\n  - "));
      console.error("\nRun: node scripts/sync-versions.mjs  (or npm run versions:sync)");
      process.exit(1);
    }
    console.log(`versions OK (${sourceVersion})`);
    return;
  }

  writeJson(rootPkgPath, rootPkg);
  writeJson(cliPkgPath, cliPkg);
  writeJson(cliEmbeddedPkgPath, cliEmbeddedPkg);
  for (const { path: p, pkg } of platPkgs) writeJson(p, pkg);
  console.log(updates.length ? `synced to ${sourceVersion}: ${updates.join(", ")}` : `already in sync (${sourceVersion})`);
}

main();
