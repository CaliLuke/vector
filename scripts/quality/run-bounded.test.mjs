import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";

const runner = new URL("./run-bounded.mjs", import.meta.url).pathname;
// Retain a bounded native allocation until the supervisor terminates us. A
// fixture that exits after allocation can disappear between process samples.
// Incompressible bytes keep this a resident-memory test under OS compression.
const memoryConsumer =
  "const {randomFillSync}=require('node:crypto'); const buffers=[]; setInterval(()=>{if(buffers.length<32)buffers.push(randomFillSync(Buffer.alloc(4*1024*1024)))},40)";
function run(script, extra = []) {
  const result = spawnSync(process.execPath, [runner, ...extra, "--", process.execPath, "-e", script], {
    encoding: "utf8",
    timeout: 7000,
  });
  assert.ifError(result.error);
  return result;
}

function runWithUnavailableSamples(t, count, { mode = "timeout", extra = [], childDuration = 4500 } = {}) {
  const directory = mkdtempSync(path.join(tmpdir(), "vector-process-sample-"));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const countPath = path.join(directory, "samples");
  writeFileSync(countPath, "0");
  writeFileSync(
    path.join(directory, "ps"),
    `#!${process.execPath}
const fs = require("node:fs");
const countPath = ${JSON.stringify(countPath)};
const count = Number(fs.readFileSync(countPath, "utf8")) + 1;
fs.writeFileSync(countPath, String(count));
if (count <= ${count}) {
  if (${JSON.stringify(mode)} === "timeout") setInterval(() => {}, 1000);
  else process.stdout.write("999999 1 999999 0\\n");
} else process.stdout.write(require("node:child_process").execFileSync("/bin/ps", process.argv.slice(2)));
`,
    { mode: 0o700 }
  );
  const result = spawnSync(
    process.execPath,
    [runner, ...extra, "--", process.execPath, "-e", `setTimeout(() => {}, ${childDuration})`],
    {
      encoding: "utf8",
      timeout: 10_000,
      env: { ...process.env, PATH: `${directory}${path.delimiter}${process.env.PATH}` },
    }
  );
  assert.ifError(result.error);
  return result;
}

void test("one delayed memory sample recovers without rejecting the child", (t) => {
  const result = runWithUnavailableSamples(t, 1);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stderr, /memory sample timed out; retrying once/);
});

void test("repeated memory sampling timeouts fail closed", (t) => {
  const result = runWithUnavailableSamples(t, 2, { childDuration: 60_000 });
  assert.equal(result.status, 125, result.stderr);
  assert.match(result.stderr, /memory monitoring failed/);
});

void test("memory limits still apply after recovering from a sampling timeout", (t) => {
  const result = runWithUnavailableSamples(t, 1, { extra: ["--memory-mib", "1"] });
  assert.equal(result.status, 137, result.stderr);
  assert.match(result.stderr, /RSS exceeded/);
});
void test("memory monitoring rediscovers the child after a snapshot omits its process tree", (t) => {
  const result = runWithUnavailableSamples(t, 1, { mode: "missing-root", extra: ["--memory-mib", "1"] });
  assert.equal(result.status, 137, result.stderr);
  assert.match(result.stderr, /RSS exceeded/);
});

void test("repeated snapshots without the child tree fail closed", (t) => {
  const result = runWithUnavailableSamples(t, 2, { mode: "missing-root" });
  assert.equal(result.status, 125, result.stderr);
  assert.match(result.stderr, /memory monitoring failed/);
});

async function assertWorkerStopped(pid) {
  let alive = true;
  for (let attempt = 0; alive && attempt < 20; attempt++) {
    try {
      process.kill(pid, 0);
    } catch {
      alive = false;
    }
    // oxlint-disable-next-line no-await-in-loop
    if (alive) await new Promise((resolve) => setTimeout(resolve, 25));
  }
  assert.equal(alive, false, "orphan worker must not survive the runner");
}

function cleanupWorker(pid) {
  if (!pid) return;
  try {
    process.kill(pid, "SIGKILL");
  } catch (error) {
    if (error.code !== "ESRCH") throw error;
  }
}
async function verifyWorkerCleanup(result, pidPath) {
  let pid;
  try {
    pid = Number(readFileSync(pidPath, "utf8"));
    assert.equal(result.status, 0, result.stderr);
    await assertWorkerStopped(pid);
  } finally {
    cleanupWorker(pid);
  }
}
void test("preserves the child exit status", () => {
  assert.equal(run("process.exit(7)").status, 7);
});
void test("external deadline interrupts a blocked child event loop", () => {
  const result = run("const end=Date.now()+5000; while(Date.now()<end) {} process.stdout.write('finished');", [
    "--seconds",
    "0.3",
  ]);
  assert.equal(result.status, 124);
  assert.match(result.stderr, /deadline exceeded/);
  // Process startup can be delayed before the guard itself starts its clock.
  // The contract is that the deadline interrupts the child's work, not that the
  // host schedules the entire test harness within an unrelated wall-clock limit.
  assert.doesNotMatch(result.stdout, /finished/);
});
void test("memory ceiling includes native buffer allocations", () => {
  const result = run(memoryConsumer, ["--memory-mib", "60", "--heap-mib", "64"]);
  assert.equal(result.status, 137, result.stderr);
  assert.match(result.stderr, /RSS exceeded/);
});
for (const detached of [false, true]) {
  void test(`cleans up a ${detached ? "detached" : "shared-group"} worker when its parent exits`, async () => {
    const directory = mkdtempSync(path.join(tmpdir(), "vector-process-guard-"));
    try {
      const pidPath = path.join(directory, "worker.pid");
      const result = run(`
      const {spawn}=require('node:child_process');
      const worker=spawn(process.execPath,['-e','setTimeout(()=>{},60000)'],{stdio:'ignore',detached:${detached}});
      require('node:fs').writeFileSync(${JSON.stringify(pidPath)},String(worker.pid));
      worker.unref();
      // Allow a complete two-second sampling window before parent exit.
      setTimeout(()=>{},2200);
    `);
      await verifyWorkerCleanup(result, pidPath);
    } finally {
      rmSync(directory, { recursive: true, force: true });
    }
  });
}

void test("a process snapshot delivered after parent exit still reaps its detached worker", async () => {
  const directory = mkdtempSync(path.join(tmpdir(), "vector-process-late-sample-"));
  const pidPath = path.join(directory, "worker.pid");
  const capturedPath = path.join(directory, "captured");
  try {
    writeFileSync(
      path.join(directory, "ps"),
      `#!/bin/sh
# Early samples must not wait for the application to start inside ps's deadline.
if [ ! -f '${pidPath}' ] || [ -f '${capturedPath}' ]; then
  exec /bin/ps "$@"
fi
/bin/ps "$@" > '${directory}/sample' || exit "$?"
touch '${capturedPath}'
sleep 0.2
cat '${directory}/sample'
`,
      { mode: 0o700 }
    );
    const script = `
const fs = require('node:fs');
const {spawn} = require('node:child_process');
const worker = spawn(process.execPath, ['-e', 'setTimeout(()=>{},60000)'], {detached:true, stdio:'ignore'});
fs.writeFileSync(${JSON.stringify(pidPath)}, String(worker.pid));
worker.unref();
const timer = setInterval(() => { if (fs.existsSync(${JSON.stringify(capturedPath)})) clearInterval(timer); }, 10);
`;
    const result = spawnSync(process.execPath, [runner, "--", process.execPath, "-e", script], {
      encoding: "utf8",
      timeout: 7000,
      env: { ...process.env, PATH: `${directory}${path.delimiter}${process.env.PATH}` },
    });
    await verifyWorkerCleanup(result, pidPath);
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});

void test("memory ceiling follows descendants that create a separate process group", () => {
  const result = run(
    `
    const {spawn}=require("node:child_process");
    spawn(process.execPath,["-e", ${JSON.stringify(memoryConsumer)}],{detached:true,stdio:"ignore"});
    setInterval(()=>{},1000);
  `,
    ["--memory-mib", "100", "--heap-mib", "64"]
  );
  assert.equal(result.status, 137, result.stderr);
  assert.match(result.stderr, /RSS exceeded/);
});
