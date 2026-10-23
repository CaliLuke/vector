import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync } from "node:fs";
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
void test("preserves the child exit status", () => {
  assert.equal(run("process.exit(7)").status, 7);
});
void test("external deadline interrupts a blocked child event loop", () => {
  const started = Date.now();
  const result = run("const end=Date.now()+5000; while(Date.now()<end) {}", ["--seconds", "0.3"]);
  assert.equal(result.status, 124);
  assert.match(result.stderr, /deadline exceeded/);
  assert.ok(Date.now() - started < 3000);
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
    try {
      const pidPath = path.join(directory, "worker.pid");
      const result = run(`
      const {spawn}=require('node:child_process');
      const worker=spawn(process.execPath,['-e','setTimeout(()=>{},5000)'],{stdio:'ignore',detached:${detached}});
      require('node:fs').writeFileSync(${JSON.stringify(pidPath)},String(worker.pid));
      worker.unref();
      setTimeout(()=>{},300);
    `);
      assert.equal(result.status, 0);
      const pid = Number(readFileSync(pidPath, "utf8"));
      // Give the kernel a bounded opportunity to reap the signalled child.
      let alive = true;
      for (let attempt = 0; alive && attempt < 20; attempt++) {
        try {
          process.kill(pid, 0);
        } catch {
          alive = false;
        }
        // Poll this one PID serially so success means the worker has exited.
        // oxlint-disable-next-line no-await-in-loop
        if (alive) await new Promise((resolve) => setTimeout(resolve, 25));
      }
      assert.equal(alive, false, "orphan worker must not survive the runner");
    } finally {
      rmSync(directory, { recursive: true, force: true });
    }
  });
}

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
