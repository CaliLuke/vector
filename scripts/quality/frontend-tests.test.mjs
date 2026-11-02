import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { partitionTestFiles } from "./frontend-test-shards.mjs";

function runFrontendFixture(t, failedFile, { addDuringRun = false, coverage = false, conversionFailure = false } = {}) {
  const root = mkdtempSync(path.join(os.tmpdir(), "vector frontend shards "));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  mkdirSync(path.join(root, "scripts/quality"), { recursive: true });
  mkdirSync(path.join(root, "tests/frontend"), { recursive: true });
  mkdirSync(path.join(root, "apps/web/tests"), { recursive: true });
  for (const script of ["frontend-tests.sh", "frontend-test-shards.mjs", "run-bounded.mjs"]) {
    copyFileSync(new URL(script, import.meta.url), path.join(root, "scripts/quality", script));
  }
  if (coverage) {
    mkdirSync(path.join(root, "tests/frontend/helpers"), { recursive: true });
    writeFileSync(
      path.join(root, "tests/frontend/helpers/coverage-preload.mjs"),
      `
      import assert from 'node:assert/strict';
      assert.equal(process.env.NODE_DISABLE_COMPILE_CACHE, '1');
      assert.equal(process.env.NODE_COMPILE_CACHE, undefined);
      assert.ok(process.env.NODE_V8_COVERAGE.endsWith('/coverage/frontend/raw'));
    `
    );
    writeFileSync(
      path.join(root, "scripts/quality/frontend-coverage.mjs"),
      `
      import {writeFileSync} from 'node:fs';
      writeFileSync('collected.txt', 'complete');
      if (${conversionFailure}) throw new Error('cannot attribute source coverage');
    `
    );
  }
  for (const name of ["a", "b", "c", "d"]) {
    writeFileSync(
      path.join(root, name === "d" ? `apps/web/tests/${name}.test.mjs` : `tests/frontend/${name}.test.mjs`),
      `import test from "node:test";
import assert from "node:assert/strict";
import { appendFileSync, writeFileSync } from "node:fs";
test("${name}", () => {
  appendFileSync("executed.txt", "${name}\\n");
  if (${addDuringRun} && "${name}" === "b") writeFileSync("tests/frontend/aa.test.mjs", "throw new Error('new test must wait until the next run');");
  assert.notEqual(${JSON.stringify(failedFile)}, "${name}");
});
`
    );
  }
  // Exercise a fresh CLI invocation, not a recursive node:test worker.
  const env = { ...process.env };
  delete env.NODE_TEST_CONTEXT;
  const result = spawnSync(
    "bash",
    [path.join(root, "scripts/quality/frontend-tests.sh"), ...(coverage ? ["--coverage"] : [])],
    {
      cwd: os.tmpdir(),
      env,
      encoding: "utf8",
      timeout: 15_000,
    }
  );
  assert.ifError(result.error);
  assert.equal(result.signal, null);
  assert.ok(existsSync(path.join(root, "executed.txt")), result.stdout + result.stderr);
  assert.deepEqual(readFileSync(path.join(root, "executed.txt"), "utf8").trim().split("\n").toSorted(), [
    "a",
    "b",
    "c",
    "d",
  ]);
  return { ...result, collected: existsSync(path.join(root, "collected.txt")) };
}

await test("frontend shards execute every discovered file exactly once", (t) => {
  const result = runFrontendFixture(t, null);
  assert.equal(result.status, 0, result.stdout + result.stderr);
});

await test("frontend shards balance large files and preserve paths with spaces", (t) => {
  const root = mkdtempSync(path.join(os.tmpdir(), "vector shard weights "));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const files = [100, 10, 90, 10].map((size, index) => {
    const file = path.join(root, `test ${index}.mjs`);
    writeFileSync(file, " ".repeat(size));
    return file;
  });
  const shards = partitionTestFiles(files);
  assert.deepEqual(shards.flat().toSorted(), files.toSorted());
  assert.ok(shards.every((shard) => shard.length === 2));
  assert.ok(shards.every((shard) => !(shard.includes(files[0]) && shard.includes(files[2]))));
  assert.deepEqual(partitionTestFiles(files.toReversed()), shards, "discovery order must not affect the split");
});

await test("a failed first frontend shard cannot be hidden by a passing second shard", (t) => {
  const result = runFrontendFixture(t, "b");
  assert.equal(result.status, 1, result.stdout + result.stderr);
});

await test("a failed second frontend shard fails the complete suite", (t) => {
  const result = runFrontendFixture(t, "a");
  assert.equal(result.status, 1, result.stdout + result.stderr);
});

await test("a failed workspace-owned test fails the complete suite", (t) => {
  const result = runFrontendFixture(t, "d");
  assert.equal(result.status, 1, result.stdout + result.stderr);
});

await test("adding a test during the first shard cannot reshuffle the second shard", (t) => {
  const result = runFrontendFixture(t, null, { addDuringRun: true });
  assert.equal(result.status, 0, result.stdout + result.stderr);
});

await test("coverage uses uncached workers and converts only after both shards pass", (t) => {
  const result = runFrontendFixture(t, null, { coverage: true });
  assert.equal(result.status, 0, result.stdout + result.stderr);
  assert.equal(result.collected, true);
});
await test("a failed coverage shard cannot produce a complete source report", (t) => {
  const result = runFrontendFixture(t, "b", { coverage: true });
  assert.equal(result.status, 1, result.stdout + result.stderr);
  assert.equal(result.collected, false);
});
await test("source attribution failure fails the coverage command", (t) => {
  const result = runFrontendFixture(t, null, { coverage: true, conversionFailure: true });
  assert.equal(result.status, 1, result.stdout + result.stderr);
  assert.equal(result.collected, true);
});
