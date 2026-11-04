import assert from "node:assert/strict";
import { spawn, spawnSync } from "node:child_process";
import { once } from "node:events";
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
function run(script, extra = [], childArgs = []) {
  const result = spawnSync(process.execPath, [runner, ...extra, "--", process.execPath, "-e", script, ...childArgs], {
    encoding: "utf8",
    timeout: 7000,
  });
  assert.ok(
    !result.error,
    `guard fixture failed (${result.error?.code ?? "unknown"}, signal=${result.signal ?? "none"}, status=${result.status ?? "none"}); stderr:\n${result.stderr}`
  );
  return result;
}

function runWithUnavailableSamples(t, count, { mode = "timeout", extra = [], childDuration } = {}) {
  const directory = mkdtempSync(path.join(tmpdir(), "vector-process-sample-"));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const countPath = path.join(directory, "samples");
  const recoveredPath = path.join(directory, "recovered");
  writeFileSync(countPath, "0");
  writeFileSync(
    path.join(directory, "ps"),
    `#!/bin/sh
read -r count < ${shellQuote(countPath)}
count=$((count + 1))
printf '%s\\n' "$count" > ${shellQuote(countPath)}
if [ "$count" -le ${count} ]; then
  if [ ${shellQuote(mode)} = timeout ]; then exec sleep 60; fi
  if [ ${shellQuote(mode)} = timeout-pipe ]; then
    # execFile closes the output pipe before sending its timeout signal. A
    # process that writes while shutting down can therefore exit with SIGPIPE.
    trap '' TERM
    sleep 2.2
    exec /bin/echo '999999 1 999999 0'
  fi
  if [ ${shellQuote(mode)} = broken-pipe ]; then kill -PIPE $$; fi
  if [ ${shellQuote(mode)} = oversized ]; then exec /usr/bin/head -c 3145728 /dev/zero; fi
  printf '999999 1 999999 0\\n'
else
  # The first healthy snapshot must reach the guard before the child exits.
  if [ "$count" -ge ${count + 2} ]; then touch ${shellQuote(recoveredPath)}; fi
  exec /bin/ps "$@"
fi
`,
    { mode: 0o700 }
  );
  const childScript = childDuration
    ? `setTimeout(() => {}, ${childDuration})`
    : `const fs = require('node:fs'); const timer = setInterval(() => {
        if (fs.existsSync(${JSON.stringify(recoveredPath)})) clearInterval(timer);
      }, 10);`;
  const result = spawnSync(process.execPath, [runner, ...extra, "--", process.execPath, "-e", childScript], {
    encoding: "utf8",
    timeout: 10_000,
    env: { ...process.env, PATH: `${directory}${path.delimiter}${process.env.PATH}` },
  });
  assert.ok(
    !result.error,
    `sampling fixture failed (${result.error?.code ?? "unknown"}, signal=${result.signal ?? "none"}, status=${result.status ?? "none"}); stderr:\n${result.stderr}`
  );
  return result;
}

void test("one delayed memory sample recovers without rejecting the child", (t) => {
  const result = runWithUnavailableSamples(t, 1);
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stderr, /memory sample timed out; retrying once/);
});

void test("repeated memory sampling timeouts fail closed with bounded diagnostics", (t) => {
  const result = runWithUnavailableSamples(t, 2, { childDuration: 60_000 });
  assert.equal(result.status, 125, result.stderr);
  assert.match(result.stderr, /memory monitoring failed/);
  assert.match(result.stderr, /sampleMs=\d+, stdoutBytes=\d+, stderrBytes=\d+/);
});

void test("a timed-out memory inspector with pending output gets one retry", (t) => {
  const result = runWithUnavailableSamples(t, 1, { mode: "timeout-pipe" });
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stderr, /memory sample timed out; retrying once/);
});

void test("repeated inspector timeouts with pending output still fail closed", (t) => {
  const result = runWithUnavailableSamples(t, 2, { mode: "timeout-pipe", childDuration: 60_000 });
  assert.equal(result.status, 125, result.stderr);
  assert.match(result.stderr, /memory sample timed out; retrying once/);
  assert.match(result.stderr, /memory monitoring failed/);
});

void test("a sampler SIGPIPE without a timeout fails immediately", (t) => {
  const result = runWithUnavailableSamples(t, 1, { mode: "broken-pipe" });
  assert.equal(result.status, 125, result.stderr);
  assert.doesNotMatch(result.stderr, /retrying once/);
});

void test("oversized memory snapshots fail immediately instead of retrying", (t) => {
  const result = runWithUnavailableSamples(t, 1, { mode: "oversized" });
  assert.equal(result.status, 125, result.stderr);
  assert.doesNotMatch(result.stderr, /retrying once/);
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
void test("preserves the child exit status with its own argument delimiter", () => {
  assert.equal(run("process.exit(7)", [], ["--", "child-argument"]).status, 7);
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
  const result = run(memoryConsumer, ["--memory-mib", "60"], ["private-command-argument"]);
  assert.equal(result.status, 137, result.stderr);
  assert.match(result.stderr, /RSS exceeded/);
  const prefix = "[bounded] memory sample ";
  const line = result.stderr.split("\n").find((value) => value.startsWith(prefix));
  assert.ok(line, "memory failure must identify the processes in its failing sample");
  const sample = JSON.parse(line.slice(prefix.length));
  assert.ok(sample.totalMiB >= 60);
  assert.ok(sample.processes.length > 0 && sample.processes.length <= 10);
  assert.ok(sample.processes.some(({ command, rssMiB }) => /(?:bun|node)/i.test(command) && rssMiB > 0));
  assert.ok(sample.processes.every(({ pid, command }) => Number.isInteger(pid) && !command.includes("/")));
  assert.doesNotMatch(result.stderr, /private-command-argument/);
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
      [runner, "--seconds", "5", "--memory-mib", "100", "--", process.execPath, "-e", script],
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

function waitForWorker(pidPath) {
  return new Promise((resolve, reject) => {
    const interval = setInterval(() => {
      if (!existsSync(pidPath)) return;
      clearInterval(interval);
      clearTimeout(deadline);
      resolve(Number(readFileSync(pidPath, "utf8")));
    }, 10);
    const deadline = setTimeout(() => {
      clearInterval(interval);
      reject(new Error("worker did not start before the fixture deadline"));
    }, 5000);
  });
}

for (const [signal, code] of [
  ["SIGINT", 130],
  ["SIGTERM", 143],
]) {
  void test(`canceling the invocation group with ${signal} stops its detached worker`, async (t) => {
    const directory = mkdtempSync(path.join(tmpdir(), "vector-group-cancellation-"));
    const pidPath = path.join(directory, "worker.pid");
    const script = `require('node:fs').writeFileSync(${JSON.stringify(pidPath)}, String(process.pid)); setInterval(()=>{},1000);`;
    const child = spawn(process.execPath, [runner, "--seconds", "10", "--", process.execPath, "-e", script], {
      detached: true,
      stdio: ["ignore", "ignore", "pipe"],
    });
    t.after(() => {
      cleanupWorkerFile(pidPath);
      cleanupWorker(child.pid);
      rmSync(directory, { recursive: true, force: true });
    });
    let stderr = "";
    child.stderr.setEncoding("utf8").on("data", (chunk) => {
      stderr += chunk;
    });
    const closed = once(child, "close");
    const worker = await waitForWorker(pidPath);
    const group = child.pid;
    if (group === undefined || group <= 0) throw new Error("guard process did not start");
    process.kill(-group, signal);
    const [exitCode, exitSignal] = await closed;
    assert.equal(exitCode, code, stderr);
    assert.equal(exitSignal, null, stderr);
    await assertWorkerStopped(worker);
  });
}

void test("a SIGTERM-resistant memory inspector cannot suspend enforcement past its sampling deadlines", async () => {
  const directory = mkdtempSync(path.join(tmpdir(), "vector-stuck-inspector-"));
  const childPath = path.join(directory, "child.pid");
  const inspectorsPath = path.join(directory, "inspectors");
  const inspector = `
process.on("SIGTERM", () => {});
require("node:fs").appendFileSync(${JSON.stringify(inspectorsPath)}, process.pid + "\\n");
setInterval(() => {}, 1000);
`;
  writeFileSync(
    path.join(directory, "ps"),
    `#!/bin/sh\nexec ${shellQuote(process.execPath)} -e ${shellQuote(inspector)}\n`,
    { mode: 0o700 }
  );
  // Keep shutdown pending until the guard escalates, so another sampling tick
  // would deterministically launch an unauthorized third inspector.
  const application = `process.on("SIGTERM", () => {}); require("node:fs").writeFileSync(${JSON.stringify(childPath)}, String(process.pid)); setInterval(() => {}, 1000);`;
  const child = spawn(process.execPath, [runner, "--seconds", "30", "--", process.execPath, "-e", application], {
    stdio: ["ignore", "ignore", "pipe"],
    env: { ...process.env, PATH: `${directory}${path.delimiter}${process.env.PATH}` },
  });
  let stderr = "";
  child.stderr.on("data", (chunk) => {
    stderr += chunk;
  });
  const completed = new Promise((resolve, reject) => {
    child.once("error", reject);
    child.once("close", (code, signal) => resolve({ code, signal }));
  });
  let deadline;
  try {
    const result = await Promise.race([
      completed,
      new Promise((resolve) => {
        deadline = setTimeout(() => resolve(null), 7000);
      }),
    ]);
    assert.notEqual(result, null, "a stuck inspector must not disable monitoring until the command deadline");
    assert.equal(result.code, 125, stderr);
    assert.match(stderr, /memory sample timed out; retrying once/);
    assert.match(stderr, /memory monitoring failed/);
    await assertWorkerStopped(Number(readFileSync(childPath, "utf8")));
    const inspectors = readFileSync(inspectorsPath, "utf8").trim().split("\n").map(Number);
    assert.equal(inspectors.length, 2, "one bounded retry is permitted");
    for (const pid of inspectors) {
      assert.throws(() => process.kill(pid, 0), { code: "ESRCH" }, "timed-out inspectors must be reaped");
    }
  } finally {
    clearTimeout(deadline);
    child.kill("SIGKILL");
    cleanupWorkerFile(childPath);
    if (existsSync(inspectorsPath)) {
      for (const pid of readFileSync(inspectorsPath, "utf8").trim().split("\n").map(Number)) cleanupWorker(pid);
    }
    await completed;
    rmSync(directory, { recursive: true, force: true });
  }
});
