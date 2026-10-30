import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";

const runner = new URL("./run-bounded.mjs", import.meta.url).pathname;
// Retain a bounded native allocation until the supervisor terminates us. A
// fixture that exits after allocation can disappear between process samples.
// Incompressible bytes keep this a resident-memory test under OS compression.
const memoryConsumer =
  "const {randomFillSync}=require('node:crypto'); const buffers=[]; setInterval(()=>{if(buffers.length<32)buffers.push(randomFillSync(Buffer.alloc(4*1024*1024)))},40)";
const shellQuote = (value) => `'${value.replaceAll("'", "'\\''")}'`;
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
    `#!/bin/sh
read -r count < ${shellQuote(countPath)}
count=$((count + 1))
printf '%s\\n' "$count" > ${shellQuote(countPath)}
if [ "$count" -le ${count} ]; then
  if [ ${shellQuote(mode)} = timeout ]; then exec sleep 60; fi
  printf '999999 1 999999 0\\n'
else
  exec /bin/ps "$@"
fi
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
function cleanupWorkerFile(pidPath) {
  if (existsSync(pidPath)) cleanupWorker(Number(readFileSync(pidPath, "utf8")));
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
    const pidPath = path.join(directory, "worker.pid");
    const capturedPath = path.join(directory, "captured");
    try {
      writeFileSync(
        path.join(directory, "ps"),
        `#!/bin/sh
/bin/ps "$@" > ${shellQuote(path.join(directory, "sample"))} || exit "$?"
cat ${shellQuote(path.join(directory, "sample"))}
if [ -f ${shellQuote(pidPath)} ]; then
  read -r worker < ${shellQuote(pidPath)}
  if awk -v worker="$worker" '$1 == worker { found=1 } END { exit !found }' ${shellQuote(path.join(directory, "sample"))}; then
    touch ${shellQuote(capturedPath)}
  fi
fi
`,
        { mode: 0o700 }
      );
      const script = `
      const {spawn}=require('node:child_process');
      const worker=spawn(process.execPath,['-e','setTimeout(()=>{},60000)'],{stdio:'ignore',detached:${detached}});
      require('node:fs').writeFileSync(${JSON.stringify(pidPath)},String(worker.pid));
      worker.unref();
      const timer=setInterval(()=>{if(require('node:fs').existsSync(${JSON.stringify(capturedPath)}))clearInterval(timer)},10);
    `;
      const result = spawnSync(process.execPath, [runner, "--", process.execPath, "-e", script], {
        encoding: "utf8",
        timeout: 7000,
        env: { ...process.env, PATH: `${directory}${path.delimiter}${process.env.PATH}` },
      });
      assert.ifError(result.error);
      await verifyWorkerCleanup(result, pidPath);
    } finally {
      cleanupWorkerFile(pidPath);
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
    cleanupWorkerFile(pidPath);
    rmSync(directory, { recursive: true, force: true });
  }
});

void test("memory ceiling follows descendants that create a separate process group", async () => {
  const directory = mkdtempSync(path.join(tmpdir(), "vector-detached-memory-"));
  const pidPath = path.join(directory, "worker.pid");
  const snapshotPath = path.join(directory, "worker.snapshot");
  let worker;
  try {
    // Keep real process ancestry and group IDs, but control this worker's RSS.
    // The native-allocation test above covers actual resident memory; this test
    // isolates cross-group accounting from OS paging and allocation timing.
    writeFileSync(
      path.join(directory, "ps"),
      `#!/bin/sh
worker=0
if [ -f ${shellQuote(pidPath)} ]; then read -r worker < ${shellQuote(pidPath)}; fi
/bin/ps "$@" | awk -v worker="$worker" -v snapshot=${shellQuote(snapshotPath)} '
  $1 == worker { print $0 > snapshot; close(snapshot); $4 = 128 * 1024 }
  { print }
'
`,
      { mode: 0o700 }
    );
    const script = `
const {spawn} = require("node:child_process");
const worker = spawn(process.execPath, ["-e", "setInterval(()=>{},1000)"], {detached:true,stdio:"ignore"});
require("node:fs").writeFileSync(${JSON.stringify(pidPath)}, String(worker.pid));
setInterval(()=>{},1000);
`;
    const result = spawnSync(
      process.execPath,
      [runner, "--seconds", "5", "--memory-mib", "100", "--heap-mib", "64", "--", process.execPath, "-e", script],
      {
        encoding: "utf8",
        timeout: 10_000,
        env: { ...process.env, PATH: `${directory}${path.delimiter}${process.env.PATH}` },
      }
    );
    worker = Number(readFileSync(pidPath, "utf8"));
    assert.ifError(result.error);
    assert.equal(result.status, 137, result.stderr);
    assert.match(result.stderr, /RSS exceeded/);
    const [pid, parent, group] = readFileSync(snapshotPath, "utf8").trim().split(/\s+/).map(Number);
    assert.equal(pid, worker);
    assert.equal(group, worker);
    assert.notEqual(parent, group);
    await assertWorkerStopped(worker);
  } finally {
    cleanupWorker(worker);
    cleanupWorkerFile(pidPath);
    rmSync(directory, { recursive: true, force: true });
  }
});
