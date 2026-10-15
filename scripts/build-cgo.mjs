// CGO build of the Go CLI (embedded Dolt issue store needs gcc + CGO_ENABLED=1).
// Usage: node scripts/build-cgo.mjs [--out <name>] [--check]
//   --out <name>  output file name inside cli/ (default: fabrik.exe on Windows, fabrik elsewhere)
//   --check       only verify gcc is present (prints version), no build.
// Fails fast with a clear `gcc missing` error when no compiler is on PATH.
// Graph commands are pure Go (modernc.org/sqlite) and work without gcc;
// issue commands use the embedded Dolt engine (dolthub/driver) and need this build.
import { spawnSync } from "node:child_process";
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
      "gcc missing: the issue store needs a CGO build (embedded Dolt requires CGO_ENABLED=1).\n" +
        "  Install a C toolchain with ICU and re-run:\n" +
        "    Windows: MSYS2 with toolchain + ICU (same as CI releases):\n" +
        "               pacman -S mingw-w64-x86_64-toolchain mingw-w64-x86_64-icu\n" +
        "             then put C:\\msys64\\mingw64\\bin on PATH\n" +
        "             (a plain WinLibs gcc lacks ICU headers and cannot link the Dolt engine)\n" +
        "    Linux:   sudo apt-get install -y build-essential libicu-dev\n" +
        "    macOS:   brew install icu4c",
    );
  }
  if (checkOnly) {
    console.log(`build-cgo: ${version}`);
    return;
  }

  console.log(`build-cgo: ${version} — CGO_ENABLED=1 go build -trimpath -o ${outName} ./cmd/fabrik`);
  const r = spawnSync("go", ["build", "-trimpath", "-o", outName, "./cmd/fabrik"], {
    cwd: CLI_DIR,
    stdio: "inherit",
    env: { ...process.env, CGO_ENABLED: "1" },
  });
  if (r.error || r.status !== 0) {
    fail(`go build failed (exit ${r.status ?? r.error})`);
  }
  console.log(`build-cgo: wrote cli/${outName}`);
}

main();
