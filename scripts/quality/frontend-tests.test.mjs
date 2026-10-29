import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";

function runFrontendFixture(t, failedFile) {
  const root = mkdtempSync(path.join(os.tmpdir(), "vector frontend shards "));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  mkdirSync(path.join(root, "scripts/quality"), { recursive: true });
  mkdirSync(path.join(root, "tests/frontend"), { recursive: true });
  for (const script of ["frontend-tests.sh", "run-bounded.mjs"]) {
    copyFileSync(new URL(script, import.meta.url), path.join(root, "scripts/quality", script));
  }
  for (const name of ["a", "b", "c"]) {
    writeFileSync(
      path.join(root, `tests/frontend/${name}.test.mjs`),
      `import test from "node:test";
import assert from "node:assert/strict";
import { appendFileSync } from "node:fs";
test("${name}", () => {
  appendFileSync("executed.txt", "${name}\\n");
  assert.notEqual(${JSON.stringify(failedFile)}, "${name}");
});
`
    );
  }
  // Exercise a fresh CLI invocation, not a recursive node:test worker.
  const env = { ...process.env };
  delete env.NODE_TEST_CONTEXT;
  const result = spawnSync("bash", [path.join(root, "scripts/quality/frontend-tests.sh")], {
    cwd: os.tmpdir(),
    env,
    encoding: "utf8",
    timeout: 15_000,
  });
  assert.ifError(result.error);
  assert.equal(result.signal, null);
  assert.ok(existsSync(path.join(root, "executed.txt")), result.stdout + result.stderr);
  assert.deepEqual(readFileSync(path.join(root, "executed.txt"), "utf8").trim().split("\n").toSorted(), [
    "a",
    "b",
    "c",
  ]);
  return result;
}

await test("frontend shards execute every discovered file exactly once", (t) => {
  const result = runFrontendFixture(t, null);
  assert.equal(result.status, 0, result.stdout + result.stderr);
});

await test("a failed first frontend shard cannot be hidden by a passing second shard", (t) => {
  const result = runFrontendFixture(t, "a");
  assert.equal(result.status, 1, result.stdout + result.stderr);
});

await test("a failed second frontend shard fails the complete suite", (t) => {
  const result = runFrontendFixture(t, "b");
  assert.equal(result.status, 1, result.stdout + result.stderr);
});
