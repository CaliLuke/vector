import { spawn } from "node:child_process";
import path from "node:path";

// Keep fixture hooks, globals and module caches isolated between test files.
// The outer supervisor owns the aggregate deadline, RSS budget and cancellation.
const files = process.argv.slice(2);
let next = 0;
let failed = false;
async function worker() {
  if (!failed && next < files.length) {
    const file = files[next++];
    const code = await new Promise((resolve, reject) => {
      const child = spawn(
        process.execPath,
        [
          "test",
          "--smol",
          "--timeout",
          "15000",
          ...(process.env.VECTOR_FRONTEND_COVERAGE
            ? ["--preload", path.resolve("tests/frontend/helpers/coverage-preload.mjs")]
            : []),
          path.resolve(file),
        ],
        { stdio: "inherit" }
      );
      child.once("error", reject);
      child.once("exit", (status, signal) => resolve(signal ? 1 : status));
    });
    if (code !== 0) failed = true;
    return worker();
  }
}
await Promise.all([worker(), worker()]);
process.exitCode = failed ? 1 : 0;
