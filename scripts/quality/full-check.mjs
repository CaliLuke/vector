import fs from "node:fs";
import path from "node:path";
import { createHash } from "node:crypto";
import { execFileSync, spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import { relevantEnvironment, runStages, stageInput } from "./gate-stages.mjs";

const root = fileURLToPath(new URL("../../", import.meta.url)).replace(/\/$/, "");
const output = (executable, args, cwd = root) =>
  execFileSync(executable, args, { cwd, encoding: "utf8", maxBuffer: 64 * 1024 * 1024 }).trim();

export function changedFiles(directory) {
  const base = process.env.VECTOR_CHECK_BASE || "HEAD";
  const tracked = execFileSync("git", ["diff", "--name-only", "-z", base, "--"], { cwd: directory })
    .toString()
    .split("\0");
  const untracked = execFileSync("git", ["ls-files", "--others", "--exclude-standard", "-z"], { cwd: directory })
    .toString()
    .split("\0");
  return [...new Set([...tracked, ...untracked].filter(Boolean))].toSorted((a, b) => (a < b ? -1 : a > b ? 1 : 0));
}

export function affectedGoPackages(packages, changed, moduleRoot) {
  const byName = new Map(packages.map((pkg) => [pkg.ImportPath, pkg]));
  const selected = new Set();
  const changedGo = changed.filter((name) => name.startsWith("apps/go-api/"));
  if (
    changed.some((name) =>
      /^(scripts\/quality\/go|scripts\/quality\/(full-check|gate-stages)|check\.sh|scripts\/dev-env\.sh|configs\/quality\/go-|\.quality\/go-|docker-compose)/.test(
        name
      )
    )
  )
    return [...byName.keys()].toSorted((a, b) => (a < b ? -1 : a > b ? 1 : 0));
  for (const name of changedGo) {
    if (!name.endsWith(".go")) return [...byName.keys()].toSorted((a, b) => (a < b ? -1 : a > b ? 1 : 0));
    const directory = path.resolve(moduleRoot, "../..", path.dirname(name));
    const pkg = packages.find((candidate) => candidate.Dir === directory);
    // Deleted packages and shared non-Go fixtures cannot be mapped safely.
    if (!pkg) return [...byName.keys()].toSorted((a, b) => (a < b ? -1 : a > b ? 1 : 0));
    selected.add(pkg.ImportPath);
  }
  let expanded = true;
  while (expanded) {
    expanded = false;
    for (const pkg of packages) {
      if (
        !selected.has(pkg.ImportPath) &&
        [...(pkg.Imports ?? []), ...(pkg.TestImports ?? []), ...(pkg.XTestImports ?? [])].some((dependency) =>
          selected.has(dependency)
        )
      ) {
        selected.add(pkg.ImportPath);
        expanded = true;
      }
    }
  }
  return [...selected].toSorted((a, b) => (a < b ? -1 : a > b ? 1 : 0));
}

async function command(args, options = {}) {
  await new Promise((resolve, reject) => {
    const child = spawn(args[0], args.slice(1), { cwd: root, stdio: "inherit", ...options });
    const stop = () => child.kill("SIGTERM");
    process.once("SIGTERM", stop);
    process.once("SIGINT", stop);
    child.once("error", reject);
    child.once("close", (code, signal) => {
      process.removeListener("SIGTERM", stop);
      process.removeListener("SIGINT", stop);
      if (code === 0 && !signal) resolve();
      else reject(new Error(`${args[0]} exited ${signal ?? code}`));
    });
  });
}

// Prioritization changes order only: every package remains in the coverage run,
// exactly once, including when there are no changes or all packages are affected.
export function orderedGoPackages(packages, changed, moduleRoot) {
  return [
    ...new Set([
      ...affectedGoPackages(packages, changed, moduleRoot),
      ...packages.map((pkg) => pkg.ImportPath).toSorted((a, b) => (a < b ? -1 : a > b ? 1 : 0)),
    ]),
  ];
}

function goTestPackages() {
  const module = path.join(root, "apps/go-api");
  const packages = output("go", ["list", "-tags=integration", "-json", "./..."], module)
    .split(/\n(?=\{)/)
    .filter(Boolean)
    .map((line) => JSON.parse(line));
  const ordered = orderedGoPackages(packages, changedFiles(root), module);
  if (!ordered.length) throw new Error("No Go packages found for the complete coverage run");
  process.stdout.write(ordered.join(" ") + "\n");
}

async function affectedFrontend() {
  const changed = changedFiles(root).filter(
    (name) => stageInput("frontend", name) && /\.(?:[cm]?[jt]sx?|json|ya?ml|sh)$/.test(name)
  );
  if (!changed.length) {
    process.stdout.write("No changed frontend inputs; complete frontend checks still run below.\n");
    return;
  }
  const testsOnly = changed.every((name) => /^(tests\/frontend|apps\/web\/tests)\/.*\.test\.mjs$/.test(name));
  if (testsOnly)
    await command([
      process.execPath,
      "scripts/quality/run-bounded.mjs",
      "--seconds",
      "600",
      "--memory-mib",
      "1024",
      "--heap-mib",
      "384",
      "--",
      process.execPath,
      "--test",
      "--test-timeout=15000",
      "--test-concurrency=2",
      ...changed,
    ]);
  else await command(["bash", "scripts/quality/frontend-tests.sh"]);
}

function buildArtifacts() {
  const result = [];
  for (const parent of ["apps", "packages"]) {
    for (const name of fs.readdirSync(path.join(root, parent))) {
      for (const suffix of ["dist", "build", ".react-router", "src/types/keys.generated.ts"]) {
        const relative = path.join(parent, name, suffix);
        if (fs.existsSync(path.join(root, relative))) result.push(relative);
      }
    }
  }
  if (!result.length) throw new Error("Frontend build produced no artifacts");
  return result;
}

function frontendCoverageArtifacts() {
  const scope = JSON.parse(fs.readFileSync(path.join(root, ".quality/coverage-scope.json"), "utf8"));
  return Object.entries(scope.coveredSuites).map(([workspace, suite]) =>
    suite.testCommand === "pnpm test:frontend:coverage"
      ? `coverage/frontend/workspaces/${workspace.replaceAll("/", "-")}.json`
      : `${workspace}/coverage/coverage-summary.json`
  );
}

const boundedGo = (args) => [
  process.execPath,
  "scripts/quality/run-bounded.mjs",
  "--seconds",
  "3600",
  "--memory-mib",
  "6144",
  "--heap-mib",
  "512",
  "--",
  "env",
  "GOMEMLIMIT=2GiB",
  "GOMAXPROCS=2",
  ...args,
];

export function stagePlan(scope) {
  const stages = [
    {
      id: "runner-regressions",
      scope: "all",
      command: [process.execPath, "--test", "scripts/quality/gate-stages.test.mjs"],
    },
  ];
  if (scope !== "typescript")
    stages.push({
      id: "go-integration-coverage",
      scope: "go",
      command: boundedGo(["bash", "scripts/quality/go.sh", "--tests"]),
      artifacts: () => [".quality/reports/go-test.jsonl", ".quality/reports/go-integration.cover"],
    });
  if (scope !== "go")
    stages.push(
      { id: "frontend-format", scope: "all", command: ["pnpm", "exec", "oxfmt", "--check", "."] },
      {
        id: "affected-frontend-tests",
        scope: "frontend",
        command: [process.execPath, "scripts/quality/full-check.mjs", "--affected-frontend"],
      }
    );
  if (scope !== "typescript")
    stages.push({ id: "go-static", scope: "go", command: boundedGo(["bash", "scripts/quality/go.sh", "--static"]) });
  if (scope !== "go")
    stages.push(
      { id: "frontend-build", scope: "frontend", command: ["pnpm", "build"], artifacts: buildArtifacts },
      {
        id: "storybook-build",
        scope: "frontend",
        command: [
          process.execPath,
          "scripts/quality/run-bounded.mjs",
          "--seconds",
          "1200",
          "--memory-mib",
          "4096",
          "--heap-mib",
          "2048",
          "--",
          "pnpm",
          "--workspace-concurrency=1",
          "--filter",
          "@plane/ui",
          "--filter",
          "@plane/propel",
          "build-storybook",
        ],
        artifacts: () => ["packages/ui/storybook-static", "packages/propel/storybook-static"],
      },
      {
        id: "frontend-checks-coverage",
        scope: "frontend",
        command: ["bash", "scripts/quality/typescript.sh", "--full"],
        artifacts: frontendCoverageArtifacts,
      }
    );
  return stages;
}

async function main(args) {
  if (args[0] === "--go-test-packages") return goTestPackages();
  if (args[0] === "--affected-frontend") return affectedFrontend();
  const pointer = path.join(root, ".quality/reports/full-gate-current.json");
  if (args[0] === "--status") {
    if (!fs.existsSync(pointer)) throw new Error("No staged quality run has been recorded yet");
    const { directory } = JSON.parse(fs.readFileSync(pointer, "utf8"));
    process.stdout.write(fs.readFileSync(path.join(directory, "status.json"), "utf8"));
    return;
  }
  const scope = args.find((arg) => ["all", "go", "typescript"].includes(arg)) ?? "all";
  if (args.some((arg) => !["all", "go", "typescript", "--fresh"].includes(arg)))
    throw new Error("Usage: full-check.mjs [all|go|typescript] [--fresh|--status]");
  const session = process.env.VECTOR_CHECK_SESSION || process.env.CODEX_THREAD_ID;
  const reuse = !!session && !process.env.CI && !args.includes("--fresh");
  const id = createHash("sha256")
    .update(`${root}\0${session || `manual-${process.pid}`}`)
    .digest("hex")
    .slice(0, 20);
  const directory = path.join(root, ".quality/reports/stages", id);
  fs.mkdirSync(directory, { recursive: true });
  fs.writeFileSync(pointer, JSON.stringify({ directory }) + "\n", { mode: 0o600 });
  const identity = {
    head: output("git", ["rev-parse", "HEAD"]),
    node: process.version,
    platform: process.platform,
    architecture: process.arch,
    environment: relevantEnvironment(process.env),
  };
  if (scope !== "typescript") identity.go = output("go", ["version"]);
  if (scope !== "go") identity.pnpm = output("pnpm", ["--version"]);
  const controller = new AbortController();
  const stop = () => controller.abort();
  process.once("SIGINT", stop);
  process.once("SIGTERM", stop);
  try {
    await runStages({ root, stages: stagePlan(scope), directory, identity, reuse, signal: controller.signal });
  } finally {
    process.removeListener("SIGINT", stop);
    process.removeListener("SIGTERM", stop);
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main(process.argv.slice(2)).catch((error) => {
    console.error(`[quality] ${error.message}`);
    process.exitCode = 1;
  });
}
