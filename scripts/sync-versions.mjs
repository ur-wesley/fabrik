// Keep versions in sync: root package.json is the source of truth.
// Mirrors into cli/package.json (Go ldflags version), cli/internal/version/package.json
// (embedded fallback), gui/Cargo.toml + gui/Cargo.lock (neon package), and all npm
// platform packages (including root optionalDependencies ranges).
// Usage: node scripts/sync-versions.mjs [--check] [--set X.Y.Z] [--print-version]
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, "..");
const rootPkgPath = path.join(ROOT, "package.json");
const cliPkgPath = path.join(ROOT, "cli", "package.json");
const cliEmbeddedPkgPath = path.join(ROOT, "cli", "internal", "version", "package.json");
const guiCargoTomlPath = path.join(ROOT, "gui", "Cargo.toml");
const guiCargoLockPath = path.join(ROOT, "gui", "Cargo.lock");
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

function readGuiCargoVersion(tomlPath) {
  const text = fs.readFileSync(tomlPath, "utf8");
  const m = text.match(/^\[package\]\r?\n(?:[^\[]*\r?\n)*?^version = "([^"]+)"/m);
  if (!m) {
    throw new Error(`sync-versions: no [package] version in ${tomlPath}`);
  }
  return { text, version: m[1] };
}

function readGuiLockNeonVersion(lockPath) {
  const text = fs.readFileSync(lockPath, "utf8");
  const m = text.match(/^name = "neon"\r?\nversion = "([^"]+)"/m);
  if (!m) {
    throw new Error(`sync-versions: no neon package in ${lockPath}`);
  }
  return { text, version: m[1] };
}

function main() {
  const args = process.argv.slice(2);
  const check = args.includes("--check");
  const printVersion = args.includes("--print-version");
  const setIdx = args.indexOf("--set");
  const setVersion = setIdx !== -1 ? args[setIdx + 1] : null;
  if (setIdx !== -1 && !setVersion) {
    console.error("sync-versions: --set requires a version, e.g. --set 1.2.3");
    process.exit(1);
  }

  const rootPkg = readJson(rootPkgPath);
  const sourceVersion = setVersion || rootPkg.version;
  if (printVersion) {
    console.log(sourceVersion);
    return;
  }
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
  const LIFECYCLE_FIELDS = ["postinstall", "install", "preinstall", "preprepare"];
  for (const field of LIFECYCLE_FIELDS) {
    if (rootPkg.scripts && rootPkg.scripts[field]) {
      problems.push(`root scripts.${field} must not exist (no postinstall allowed)`);
    }
  }
  if (!rootPkg.publishConfig || rootPkg.publishConfig.access !== "public") {
    problems.push(`root: publishConfig.access must be "public"`);
    rootPkg.publishConfig = { access: "public" };
    updates.push("root publishConfig");
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
  for (const { label, pkg } of platPkgs) {
    for (const field of LIFECYCLE_FIELDS) {
      if (pkg.scripts && pkg.scripts[field]) {
        problems.push(`${label}: scripts.${field} must not exist (no lifecycle on platform pkgs)`);
        delete pkg.scripts[field];
        updates.push(`${label} scripts.${field}`);
      }
    }
    if (pkg.private) {
      problems.push(`${label}: private must not be set (platform pkgs are published)`);
      delete pkg.private;
      updates.push(`${label} private`);
    }
    if (!pkg.publishConfig || pkg.publishConfig.access !== "public") {
      problems.push(`${label}: publishConfig.access must be "public"`);
      pkg.publishConfig = { access: "public" };
      updates.push(`${label} publishConfig`);
    }
  }

  const guiCargo = readGuiCargoVersion(guiCargoTomlPath);
  let guiCargoText = guiCargo.text;
  if (guiCargo.version !== sourceVersion) {
    problems.push(`gui/Cargo.toml: ${guiCargo.version} != ${sourceVersion}`);
    guiCargoText = guiCargo.text.replace(
      /^(\[package\]\r?\n(?:[^\[]*\r?\n)*?^version = )"[^"]+"/m,
      `$1"${sourceVersion}"`,
    );
    updates.push("gui/Cargo.toml");
  }

  const guiLock = readGuiLockNeonVersion(guiCargoLockPath);
  let guiLockText = guiLock.text;
  if (guiLock.version !== sourceVersion) {
    problems.push(`gui/Cargo.lock (neon): ${guiLock.version} != ${sourceVersion}`);
    guiLockText = guiLock.text.replace(
      /^name = "neon"\r?\nversion = "[^"]+"/m,
      `name = "neon"\nversion = "${sourceVersion}"`,
    );
    updates.push("gui/Cargo.lock");
  }

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
  if (guiCargo.version !== sourceVersion) {
    fs.writeFileSync(guiCargoTomlPath, guiCargoText);
  }
  if (guiLock.version !== sourceVersion) {
    fs.writeFileSync(guiCargoLockPath, guiLockText);
  }
  console.log(updates.length ? `synced to ${sourceVersion}: ${updates.join(", ")}` : `already in sync (${sourceVersion})`);
}

main();
