import { createHash } from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { spawn, execFileSync } from "node:child_process";

// Preserve order and bounded memory when hashing files or running expensive stages.
const serial = (values, operation) =>
  values.reduce((previous, value) => previous.then(() => operation(value)), Promise.resolve());

export function gitFiles(root) {
  return [
    ...new Set(
      execFileSync("git", ["ls-files", "-z", "--cached", "--others", "--exclude-standard"], { cwd: root })
        .toString()
        .split("\0")
        .filter(Boolean)
    ),
  ].toSorted((a, b) => (a < b ? -1 : a > b ? 1 : 0));
}

export async function hashPaths(root, names, { required = false } = {}) {
  const hash = createHash("sha256");
  async function add(name) {
    const filename = path.resolve(root, name);
    if (filename !== root && !filename.startsWith(root + path.sep))
      throw new Error("Fingerprint path escapes checkout");
    hash.update(name + "\0");
    let stat;
    try {
      stat = fs.lstatSync(filename);
    } catch (error) {
      if (error.code !== "ENOENT") throw error;
      if (required) throw new Error(`Required gate artifact is missing: ${name}`, { cause: error });
      hash.update("missing\0");
      return;
    }
    hash.update(String(stat.mode) + "\0");
    if (stat.isSymbolicLink()) {
      hash.update(fs.readlinkSync(filename));
      return;
    }
    if (stat.isDirectory()) {
      await serial(
        fs.readdirSync(filename).toSorted((a, b) => (a < b ? -1 : a > b ? 1 : 0)),
        (child) => add(path.join(name, child))
      );
    } else {
      for await (const chunk of fs.createReadStream(filename)) hash.update(chunk);
    }
    hash.update("\0");
  }
  await serial(
    [...new Set(names)].toSorted((a, b) => (a < b ? -1 : a > b ? 1 : 0)),
    add
  );
  return hash.digest("hex");
}

export function relevantEnvironment(env) {
  // Hash values, never write them to a receipt or log. Build-time variables are
  // included even when the tool does not explicitly list them in turbo.json.
  return Object.fromEntries(
    Object.entries(env)
      .filter(
        ([key]) =>
          /^(BUN_|JSC_|GO|CGO|DOCKER|TESTCONTAINERS|LIVE_TEST|PLANE_|VECTOR_|VITE_|NODE_|SENTRY_|API_|WEB_|LIVE_|CORS_|APP_|LC_)/.test(
            key
          ) || ["PATH", "TZ", "LANG", "CI", "DEV", "LOG_LEVEL"].includes(key)
      )
      // The resolved comparison base is tracked separately for stages that
      // select changed files. It does not change complete test/build results.
      .filter(([key]) => !["VECTOR_CHECK_SESSION", "VECTOR_CHECK_BASE"].includes(key))
      .toSorted(([a], [b]) => a.localeCompare(b))
  );
}

export function stageInput(scope, name) {
  const goOnly =
    (name.startsWith("apps/go-api/") && !/\.(md|json|ya?ml)$/.test(name)) ||
    /^(configs\/quality\/go-|scripts\/quality\/go[.-]|\.quality\/go-)/.test(name);
  // The engineering debt queue has no frontend consumer. Formatting still
  // checks it through its all-files stage; retain other docs conservatively.
  if (scope === "frontend") return !goOnly && name !== "docs/tech-debt-catalog.md";
  if (scope === "go")
    return (
      goOnly ||
      /^(apps\/go-api\/|scripts\/quality\/|configs\/quality\/go-|\.quality\/go-|scripts\/dev-env\.sh$|docker-compose)/.test(
        name
      ) ||
      ["check.sh", ".pre-commit-config.yaml"].includes(name)
    );
  return true;
}

export async function inputFingerprint(root, stage, identity) {
  const files = gitFiles(root).filter((name) => stageInput(stage.scope, name));
  // Ignored local configuration and installed dependency manifests also affect
  // results; git's source inventory alone cannot establish equivalence.
  const extras = [".env", ".env.test.local", "bun.lock", "bunfig.toml"];
  for (const parent of ["apps", "packages"]) {
    if (!fs.existsSync(path.join(root, parent))) continue;
    for (const entry of fs.readdirSync(path.join(root, parent))) {
      const directory = path.join(root, parent, entry);
      if (!fs.statSync(directory).isDirectory()) continue;
      for (const name of fs
        .readdirSync(directory)
        .filter((candidate) => candidate === ".env" || candidate.startsWith(".env.")))
        extras.push(path.join(parent, entry, name));
    }
  }
  if (stage.scope === "go")
    extras.push(...["golangci-lint", "goimports", "deadcode", "dupl"].map((name) => `.quality/bin/${name}`));
  const source = await hashPaths(root, [...files, ...extras]);
  const scopedIdentity = { ...identity };
  // The source digest already covers the checked tree. A commit alone must not
  // invalidate unchanged stages, including frontend checks after a Go-only fix.
  delete scopedIdentity.head;
  if (!stage.dependsOnChangedFiles) delete scopedIdentity.comparisonBase;
  if (stage.scope === "go") delete scopedIdentity.packageManager;
  if (stage.scope === "frontend") delete scopedIdentity.go;
  return createHash("sha256")
    .update(JSON.stringify({ source, identity: scopedIdentity, command: stage.command, version: 1 }))
    .digest("hex");
}

export function acquireLock(directory) {
  fs.mkdirSync(directory, { recursive: true });
  const lock = path.join(directory, "lock");
  try {
    fs.mkdirSync(lock);
  } catch (error) {
    if (error.code !== "EEXIST") throw error;
    let owner;
    try {
      owner = JSON.parse(fs.readFileSync(path.join(lock, "owner.json"), "utf8"));
    } catch (ownerError) {
      throw new Error(`Gate lock has no valid owner: ${lock}`, { cause: ownerError });
    }
    try {
      process.kill(owner.pid, 0);
    } catch (failure) {
      if (failure.code !== "ESRCH") throw failure;
      fs.rmSync(lock, { recursive: true });
      return acquireLock(directory);
    }
    throw new Error(`A quality gate is already running (PID ${owner.pid}); see ${directory}/status.json`, {
      cause: error,
    });
  }
  fs.writeFileSync(path.join(lock, "owner.json"), JSON.stringify({ pid: process.pid }));
  return () => fs.rmSync(lock, { recursive: true, force: true });
}

function writeJSON(filename, value) {
  const temporary = `${filename}.${process.pid}.tmp`;
  fs.writeFileSync(temporary, JSON.stringify(value, null, 2) + "\n", { mode: 0o600 });
  fs.renameSync(temporary, filename);
}

export async function executeStage(command, root, log, report, signal) {
  const fd = fs.openSync(log, "w", 0o600);
  try {
    await new Promise((resolve, reject) => {
      const child = spawn(command[0], command.slice(1), {
        cwd: root,
        env: process.env,
        stdio: ["ignore", "pipe", "pipe"],
      });
      let failure;
      const abort = () => {
        failure = new Error("Quality gate interrupted");
        child.kill("SIGTERM");
      };
      signal?.addEventListener("abort", abort, { once: true });
      if (signal?.aborted) abort();
      for (const output of [child.stdout, child.stderr])
        output.on("data", (chunk) => {
          if (failure) return;
          try {
            fs.writeSync(fd, chunk);
            report(chunk.toString());
          } catch (error) {
            failure = error;
            child.kill("SIGTERM");
          }
        });
      child.on("error", (error) => {
        failure = error;
      });
      child.on("close", (code, exitSignal) => {
        signal?.removeEventListener("abort", abort);
        if (failure) reject(failure);
        else if (code === 0 && !signal?.aborted) resolve();
        else reject(new Error(`Stage exited ${exitSignal ?? code}`));
      });
    });
  } finally {
    fs.closeSync(fd);
  }
}

async function reusableReceipt(root, receiptFile, key, reuse, now, maxAgeMs) {
  if (!reuse) return null;
  let receipt;
  try {
    receipt = JSON.parse(fs.readFileSync(receiptFile, "utf8"));
  } catch {
    return null;
  }
  if (
    receipt?.version !== 1 ||
    receipt.key !== key ||
    !(receipt.completed <= now()) ||
    now() - receipt.completed >= maxAgeMs
  )
    return null;
  try {
    return (await hashPaths(root, receipt.artifacts, { required: true })) === receipt.artifactHash ? receipt : null;
  } catch {
    return null;
  }
}

async function recordStage({
  root,
  stage,
  directory,
  identity,
  key,
  receiptFile,
  execute,
  report,
  status,
  signal,
  now,
}) {
  fs.rmSync(receiptFile, { force: true });
  const log = path.join(directory, stage.id + ".log");
  const stageStarted = now();
  status("running", stage.id, { log });
  report(`\n▶ ${stage.id} — log: ${log}\n`);
  const timer = setInterval(() => {
    const stageElapsedSeconds = Math.round((now() - stageStarted) / 1000);
    status("running", stage.id, { log, stageElapsedSeconds });
    report(`[quality] ${stage.id}: ${stageElapsedSeconds}s elapsed\n`);
  }, 30000);
  try {
    await execute(stage.command, root, log, report, signal);
    if (signal?.aborted || (await inputFingerprint(root, stage, identity)) !== key)
      throw new Error("Inputs changed during the stage; rerun against a stable checkout");
    const artifacts = [path.relative(root, log), ...(stage.artifacts?.() ?? [])];
    const artifactHash = await hashPaths(root, artifacts, { required: true });
    const elapsedSeconds = Math.round((now() - stageStarted) / 1000);
    writeJSON(receiptFile, { version: 1, key, completed: now(), elapsedSeconds, artifacts, artifactHash, log });
    report(`✓ ${stage.id}: passed in ${elapsedSeconds}s\n`);
    return { stage: stage.id, state: "passed", elapsedSeconds };
  } finally {
    clearInterval(timer);
  }
}

export async function runStages({
  root,
  stages,
  directory,
  identity,
  reuse = false,
  maxAgeMs = 8 * 60 * 60 * 1000,
  execute = executeStage,
  report = (text) => process.stdout.write(text),
  signal,
  now = Date.now,
}) {
  const release = acquireLock(directory);
  const started = now();
  const results = [];
  const completed = [];
  let activeStage = null;
  const statusFile = path.join(directory, "status.json");
  const status = (state, stage, extra = {}) =>
    writeJSON(statusFile, {
      state,
      stage,
      started,
      elapsedSeconds: Math.round((now() - started) / 1000),
      results,
      ...extra,
    });
  try {
    await serial(stages, async (stage) => {
      activeStage = stage.id;
      if (signal?.aborted) throw new Error("Quality gate interrupted");
      const key = await inputFingerprint(root, stage, identity);
      const receiptFile = path.join(directory, stage.id + ".json");
      const receipt = await reusableReceipt(root, receiptFile, key, reuse, now, maxAgeMs);
      if (receipt) {
        report(`\n✓ ${stage.id}: reused matching success (${receipt.elapsedSeconds}s; ${receipt.log})\n`);
        results.push({ stage: stage.id, state: "reused", elapsedSeconds: receipt.elapsedSeconds });
        status("running", stage.id);
      } else {
        results.push(
          await recordStage({
            root,
            stage,
            directory,
            identity,
            key,
            receiptFile,
            execute,
            report,
            status,
            signal,
            now,
          })
        );
      }
      completed.push({ stage, key });
    });
    await serial(completed, async ({ stage, key }) => {
      activeStage = stage.id;
      if ((await inputFingerprint(root, stage, identity)) !== key)
        throw new Error(`Inputs changed after ${stage.id} passed; rerun the gate`);
    });
    status("passed", null);
    report(`\nAll ${stages.length} required stages passed or have matching successful evidence.\n`);
    return results;
  } catch (error) {
    status("failed", activeStage, { error: error.message, log: path.join(directory, activeStage + ".log") });
    throw error;
  } finally {
    release();
  }
}
