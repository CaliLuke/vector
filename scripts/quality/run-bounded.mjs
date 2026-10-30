#!/usr/bin/env node
// Run a POSIX process tree with an external deadline and sampled RSS ceiling.
// The deadline stays responsive even when a test blocks its own event loop.
import { spawn, execFile } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);
const separator = process.argv.indexOf("--");
const options = new Map();
for (let index = 2; index < separator; index += 2) {
  options.set(process.argv[index], Number(process.argv[index + 1]));
}
const seconds = options.get("--seconds") ?? 120;
const memoryMiB = options.get("--memory-mib") ?? 2048;
const heapMiB = options.get("--heap-mib") ?? 1024;
const command = process.argv.slice(separator + 1);
if (
  separator < 2 ||
  command.length === 0 ||
  [...options.keys()].some((key) => !["--seconds", "--memory-mib", "--heap-mib"].includes(key)) ||
  [seconds, memoryMiB, heapMiB].some((value) => !Number.isFinite(value) || value <= 0)
) {
  process.stderr.write("Usage: run-bounded.mjs [--seconds N] [--memory-mib N] [--heap-mib N] -- command args...\n");
  process.exit(2);
}
if (process.platform === "win32") {
  process.stderr.write("The process-group guard requires macOS or Linux.\n");
  process.exit(2);
}

const child = spawn(command[0], command.slice(1), {
  detached: true,
  stdio: "inherit",
  env: {
    ...process.env,
    NODE_OPTIONS: `${process.env.NODE_OPTIONS ?? ""} --max-old-space-size=${heapMiB}`.trim(),
  },
});
let failureCode = 0;
let monitoring = false;
let sampleFailures = 0;
let peakKiB = 0;
let forceTimer;
let finished = false;
let trackedPids = new Set(child.pid ? [child.pid] : []);
let trackedGroups = new Set(child.pid ? [child.pid] : []);
let ungroupedPids = [];
let trackedRows = child.pid ? [{ pid: child.pid, group: child.pid }] : [];
function signalProcess(target, signal) {
  try {
    process.kill(target, signal);
    return true;
  } catch (error) {
    if (error.code === "ESRCH") return true;
    // macOS can return EPERM for a group whose members are already exiting.
    // Signal its known member PIDs individually instead of abandoning cleanup.
    if (target < 0 && error.code === "EPERM") return false;
    throw error;
  }
}
function signalGroup(signal) {
  for (const group of trackedGroups) {
    if (!signalProcess(-group, signal)) {
      for (const member of trackedRows.filter((row) => row.group === group)) {
        signalProcess(member.pid, signal);
      }
    }
  }
  for (const pid of ungroupedPids) signalProcess(pid, signal);
}
function descendants(rows) {
  const byParent = new Map();
  for (const row of rows) {
    const siblings = byParent.get(row.parent) ?? [];
    siblings.push(row);
    byParent.set(row.parent, siblings);
  }
  const selected = rows.filter((row) => trackedPids.has(row.pid) || trackedGroups.has(row.group));
  const seen = new Set(selected.map((row) => row.pid));
  // Turbo starts task-specific groups. Follow ancestry as well as group identity.
  for (const row of selected) {
    for (const descendant of byParent.get(row.pid) ?? []) {
      if (!seen.has(descendant.pid)) {
        seen.add(descendant.pid);
        selected.push(descendant);
      }
    }
  }
  trackedRows = selected;
  trackedPids = seen;
  trackedGroups = new Set(
    selected.filter((row) => row.pid === row.group || trackedGroups.has(row.group)).map((row) => row.group)
  );
  // A snapshot can omit a live root. Keep its spawn identity so later samples
  // can rediscover the tree and cleanup still owns the original process group.
  if (child.pid) {
    trackedPids.add(child.pid);
    trackedGroups.add(child.pid);
  }
  ungroupedPids = selected.filter((row) => !trackedGroups.has(row.group)).map((row) => row.pid);
  return selected;
}
function stop(code, reason) {
  if (failureCode) return;
  failureCode = code;
  process.stderr.write(`[bounded] ${reason}; terminating process group ${child.pid}\n`);
  signalGroup("SIGTERM");
  forceTimer = setTimeout(() => signalGroup("SIGKILL"), 500);
}
async function groupMemory() {
  const { stdout } = await execFileAsync("ps", ["-axo", "pid=,ppid=,pgid=,rss="], {
    timeout: 2000,
    maxBuffer: 2 * 1024 * 1024,
  });
  const rows = stdout
    .trim()
    .split("\n")
    .map((row) => {
      const [pid, parent, group, rss] = row.trim().split(/\s+/).map(Number);
      return { pid, parent, group, rss };
    });
  // A snapshot captured before exit can arrive after the child's close event.
  // Keep that ownership evidence and reap workers discovered by the final sample.
  const owned = descendants(rows);
  if (finished) {
    signalGroup("SIGKILL");
    return 0;
  }
  if (owned.length === 0) {
    throw Object.assign(new Error("child tree missing from process snapshot"), { code: "NO_PROCESS_TREE" });
  }
  return owned.reduce((total, row) => total + row.rss, 0);
}

async function checkMemory() {
  if (monitoring || finished) return;
  monitoring = true;
  try {
    const usage = await groupMemory();
    sampleFailures = 0;
    peakKiB = Math.max(peakKiB, usage);
    if (failureCode) signalGroup("SIGKILL");
    if (usage > memoryMiB * 1024) stop(137, `RSS exceeded ${memoryMiB} MiB`);
  } catch (error) {
    if (finished) return;
    // Process enumeration can briefly stall during compiler process churn.
    // Permit one missed sample, but never continue without working monitoring.
    // execFile destroys its pipes before sending the timeout signal, so a
    // writer can exit with SIGPIPE instead of SIGTERM. Its killed flag and
    // null exit code identify this timeout; buffer-limit errors have a code.
    const timedOut = error.killed && error.code === null;
    if ((timedOut || error.code === "NO_PROCESS_TREE") && sampleFailures++ === 0) {
      const reason = timedOut ? "memory sample timed out" : "child tree missing from memory sample";
      process.stderr.write(`[bounded] ${reason}; retrying once\n`);
      return;
    }
    stop(
      125,
      `memory monitoring failed (code=${error.code ?? "unknown"}, killed=${Boolean(error.killed)}, signal=${error.signal ?? "none"})`
    );
  } finally {
    monitoring = false;
  }
}
const monitor = setInterval(() => {
  void checkMemory();
}, 100);
const deadline = setTimeout(() => stop(124, `deadline exceeded ${seconds}s`), seconds * 1000);
void checkMemory();
process.once("SIGINT", () => stop(130, "interrupted"));
process.once("SIGTERM", () => stop(143, "terminated"));
process.once("exit", () => signalGroup("SIGKILL"));
child.once("error", (error) => {
  process.stderr.write(`[bounded] ${error.message}\n`);
  failureCode = 127;
});
child.once("exit", () => signalGroup("SIGKILL"));
child.once("close", (code, signal) => {
  finished = true;
  clearInterval(monitor);
  clearTimeout(deadline);
  clearTimeout(forceTimer);
  // A runner can exit while its worker is stuck. Always reap the whole group.
  signalGroup("SIGKILL");
  process.stderr.write(`[bounded] peak sampled RSS ${(peakKiB / 1024).toFixed(0)} MiB\n`);
  process.exitCode = failureCode || code || (signal ? 1 : 0);
});
