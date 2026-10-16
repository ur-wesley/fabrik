// Stripped (CGO-capable) build of the Go CLI: -trimpath plus -ldflags="-s -w"
// (drops DWARF/symtab, ~20-25% smaller). Prints the output binary size.
// Usage: node scripts/build-cgo.mjs [--out <name>] [--check]
//   --out <name>  output file name inside cli/ (default: fabrik.exe on Windows, fabrik elsewhere)
//   --check       only verify gcc is present (prints version), no build.
// CGO is optional now: the issue store is pure Go (modernc.org/sqlite,
// HasCGO=false) and builds fine with CGO_ENABLED=0 and no gcc. This script
// keeps the CGO_ENABLED=1 default so existing `mise run cli:build-cgo`
// callers keep working; it fails fast with a clear `gcc missing` error when
// no compiler is on PATH (pure-Go fallback: CGO_ENABLED=0 go build
// -trimpath -ldflags="-s -w" -o <out> ./cmd/fabrik).
import { spawnSync } from "node:child_process";
import { statSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, "..");
const CLI_DIR = path.join(ROOT, "cli");

function fail(msg) {
  console.error(`build-cgo: ${msg}`);
  process.exit(1);
}

function gccVersion() {
  const r = spawnSync("gcc", ["--version"], { encoding: "utf8" });
  if (r.error || r.status !== 0) return null;
  const first = (r.stdout || "").split("\n")[0].trim();
  return first || "gcc (version unknown)";
}

function main() {
  const args = process.argv.slice(2);
  const checkOnly = args.includes("--check");
  const outIdx = args.indexOf("--out");
  const outName =
    outIdx !== -1 ? args[outIdx + 1] : process.platform === "win32" ? "fabrik.exe" : "fabrik";
  if (outIdx !== -1 && !outName) fail("--out requires a file name, e.g. --out fabrik.exe");

  const version = gccVersion();
  if (!version) {
    fail(
      "gcc missing: this script builds with CGO_ENABLED=1, which needs a C toolchain with ICU.\n" +
        "  The CLI itself is pure Go now (modernc.org/sqlite, no CGO required), so either\n" +
        "  install a toolchain and re-run, or build without CGO:\n" +
        "    CGO_ENABLED=0 go build -trimpath -ldflags=\"-s -w\" -o <out> ./cmd/fabrik\n" +
        "  Toolchain install:\n" +
        "    Windows: MSYS2 with toolchain + ICU (same as CI releases):\n" +
        "               pacman -S mingw-w64-x86_64-toolchain mingw-w64-x86_64-icu\n" +
        "             then put C:\\msys64\\mingw64\\bin on PATH\n" +
        "    Linux:   sudo apt-get install -y build-essential libicu-dev\n" +
        "    macOS:   brew install icu4c",
    );
  }
  if (checkOnly) {
    console.log(`build-cgo: ${version}`);
    return;
  }

  console.log(
    `build-cgo: ${version} — CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o ${outName} ./cmd/fabrik`,
  );
  const r = spawnSync(
    "go",
    ["build", "-trimpath", "-ldflags", "-s -w", "-o", outName, "./cmd/fabrik"],
    {
      cwd: CLI_DIR,
      stdio: "inherit",
      env: { ...process.env, CGO_ENABLED: "1" },
    },
  );
  if (r.error || r.status !== 0) {
    fail(`go build failed (exit ${r.status ?? r.error})`);
  }
  try {
    const bytes = statSync(path.join(CLI_DIR, outName)).size;
    console.log(
      `build-cgo: wrote cli/${outName} (${bytes} bytes, ${(bytes / 1048576).toFixed(1)} MiB, stripped -s -w)`,
    );
  } catch {
    console.log(`build-cgo: wrote cli/${outName}`);
  }
}

main();
