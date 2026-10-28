import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";

const runner = new URL("./run-bounded.mjs", import.meta.url).pathname;
function run(script, extra = []) {
  return spawnSync(process.execPath, [runner, ...extra, "--", process.execPath, "-e", script], {
    encoding: "utf8",
    timeout: 7000,
  });
}
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
  const result = run(
    "const buffers=[]; const timer=setInterval(()=>{buffers.push(Buffer.alloc(4*1024*1024,1)); if(buffers.length===24)clearInterval(timer)},40)",
    ["--memory-mib", "60", "--heap-mib", "64"]
  );
  assert.equal(result.status, 137);
  assert.match(result.stderr, /RSS exceeded/);
});
for (const detached of [false, true]) {
  void test(`cleans up a ${detached ? "detached" : "shared-group"} worker when its parent exits`, async () => {
    const directory = mkdtempSync(path.join(tmpdir(), "vector-process-guard-"));
    let pid;
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
      pid = Number(readFileSync(pidPath, "utf8"));
      assert.equal(result.status, 0, result.stderr);
      // Give the kernel a bounded opportunity to reap the signalled child.
      await assertWorkerStopped(pid);
    } finally {
      cleanupWorker(pid);
      rmSync(directory, { recursive: true, force: true });
    }
  });
}

void test("a process snapshot delivered after parent exit still reaps its detached worker", async () => {
  const directory = mkdtempSync(path.join(tmpdir(), "vector-process-late-sample-"));
  const pidPath = path.join(directory, "worker.pid");
  const capturedPath = path.join(directory, "captured");
  let pid;
  try {
    writeFileSync(
      path.join(directory, "ps"),
      `#!/usr/bin/env node
const fs = require('node:fs');
const {spawnSync} = require('node:child_process');
const timer = setInterval(() => {
  if (!fs.existsSync(${JSON.stringify(pidPath)})) return;
  clearInterval(timer);
  const sample = spawnSync('/bin/ps', process.argv.slice(2), {encoding: 'utf8'});
  fs.writeFileSync(${JSON.stringify(capturedPath)}, 'captured');
  setTimeout(() => { process.stdout.write(sample.stdout); process.exitCode = sample.status ?? 1; }, 200);
}, 10);
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
    pid = Number(readFileSync(pidPath, "utf8"));
    assert.equal(result.status, 0, result.stderr);
    await assertWorkerStopped(pid);
  } finally {
    cleanupWorker(pid);
    rmSync(directory, { recursive: true, force: true });
  }
});

void test("memory ceiling follows descendants that create a separate process group", () => {
  const result = run(
    `
    const {spawn}=require("node:child_process");
    spawn(process.execPath,["-e", "const buffers=[]; const timer=setInterval(()=>{buffers.push(Buffer.alloc(4*1024*1024,1)); if(buffers.length===24)clearInterval(timer)},40)"],{detached:true,stdio:"ignore"});
    setTimeout(()=>{},5000);
  `,
    ["--memory-mib", "100", "--heap-mib", "64"]
  );
  assert.equal(result.status, 137);
  assert.match(result.stderr, /RSS exceeded/);
});
