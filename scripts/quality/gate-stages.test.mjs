import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { execFileSync } from "node:child_process";
import test from "node:test";
import { acquireLock, executeStage, inputFingerprint, runStages } from "./gate-stages.mjs";
import { affectedGoPackages, orderedGoPackages, stagePlan } from "./full-check.mjs";

function fixture(t) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "vector-gate-test-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const write = (name, value = "fixture\n") => {
    fs.mkdirSync(path.dirname(path.join(root, name)), { recursive: true });
    fs.writeFileSync(path.join(root, name), value);
  };
  write(".gitignore", ".quality/reports/\n");
  write("apps/go-api/internal/example/value.go", "package example\n");
  write("apps/web/src/value.ts", "export const value = 1;\n");
  write("scripts/quality/go.sh");
  write("pnpm-lock.yaml");
  execFileSync("git", ["init", "-q"], { cwd: root });
  execFileSync("git", ["add", "."], { cwd: root });
  execFileSync(
    "git",
    ["-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "fixture", "--no-verify"],
    { cwd: root }
  );
  const directory = path.join(root, ".quality/reports/stages/test");
  const calls = [];
  const stages = [
    { id: "affected-go-tests", scope: "go", command: ["preflight"] },
    { id: "frontend-build", scope: "frontend", command: ["build"] },
  ];
  const execute = async ([name], _root, log) => {
    calls.push(name);
    fs.writeFileSync(log, `passed ${name}\n`);
  };
  const options = { root, directory, stages, execute, identity: { tools: "v1" }, reuse: true, report: () => {} };
  return { root, directory, write, calls, stages, options };
}

test("a failing affected test prevents all expensive stages and never creates a success receipt", async (t) => {
  const f = fixture(t);
  await assert.rejects(
    runStages({
      ...f.options,
      execute: async ([name]) => {
        f.calls.push(name);
        throw new Error("regression");
      },
    }),
    /regression/
  );
  assert.deepEqual(f.calls, ["preflight"]);
  assert.equal(fs.existsSync(path.join(f.directory, "affected-go-tests.json")), false);
  assert.equal(JSON.parse(fs.readFileSync(path.join(f.directory, "status.json"))).state, "failed");
  await runStages(f.options);
  assert.deepEqual(f.calls, ["preflight", "preflight", "build"]);
});

test("a Go fix reruns its preflight while reusing the unchanged frontend result", async (t) => {
  const f = fixture(t);
  await runStages(f.options);
  f.write("apps/go-api/internal/example/value.go", "package example\n// fixed\n");
  const result = await runStages(f.options);
  assert.deepEqual(f.calls, ["preflight", "build", "preflight"]);
  assert.deepEqual(
    result.map(({ state }) => state),
    ["passed", "reused"]
  );
  f.write("apps/web/src/value.ts", "export const value = 2;\n");
  await runStages(f.options);
  assert.equal(f.calls.at(-1), "build");
});

test("untracked sources, deleted sources, and lockfile changes invalidate affected results", async (t) => {
  const f = fixture(t);
  const stage = f.stages[1];
  const first = await inputFingerprint(f.root, stage, f.options.identity);
  f.write("apps/web/src/new.ts");
  const added = await inputFingerprint(f.root, stage, f.options.identity);
  assert.notEqual(first, added);
  fs.unlinkSync(path.join(f.root, "apps/web/src/value.ts"));
  const deleted = await inputFingerprint(f.root, stage, f.options.identity);
  assert.notEqual(added, deleted);
  f.write("pnpm-lock.yaml", "changed dependencies\n");
  assert.notEqual(deleted, await inputFingerprint(f.root, stage, f.options.identity));
});

test("toolchain, environment and checker configuration changes invalidate successful results", async (t) => {
  const f = fixture(t);
  await runStages(f.options);
  await runStages({ ...f.options, identity: { tools: "v2", environment: { NODE_OPTIONS: "--conditions=test" } } });
  assert.deepEqual(f.calls, ["preflight", "build", "preflight", "build"]);
  f.write("scripts/quality/go.sh", "changed gate\n");
  const result = await runStages({
    ...f.options,
    identity: { tools: "v2", environment: { NODE_OPTIONS: "--conditions=test" } },
  });
  assert.deepEqual(
    result.map(({ state }) => state),
    ["passed", "reused"]
  );
});

test("missing or altered artifacts cannot certify a cached stage", async (t) => {
  const f = fixture(t);
  const artifact = ".quality/reports/build/output.js";
  const stages = [{ ...f.stages[1], artifacts: () => [artifact] }];
  const options = {
    ...f.options,
    stages,
    execute: async (...args) => {
      await f.options.execute(...args);
      f.write(artifact, "build output\n");
    },
  };
  await runStages(options);
  fs.unlinkSync(path.join(f.root, artifact));
  await runStages(options);
  f.write(artifact, "corrupt output\n");
  await runStages(options);
  assert.deepEqual(f.calls, ["build", "build", "build"]);
});

test("fresh runs and expired results execute tests again", async (t) => {
  const f = fixture(t);
  await runStages({ ...f.options, now: () => 1000 });
  await runStages({ ...f.options, now: () => 2000, maxAgeMs: 500 });
  await runStages({ ...f.options, now: () => 2001, reuse: false });
  assert.equal(f.calls.length, 6);
});

test("a failed fresh recheck removes an earlier success even if inputs are unchanged", async (t) => {
  const f = fixture(t);
  await runStages(f.options);
  await assert.rejects(
    runStages({
      ...f.options,
      reuse: false,
      execute: async () => {
        throw new Error("failed recheck");
      },
    }),
    /failed recheck/
  );
  const result = await runStages(f.options);
  assert.equal(result[0].state, "passed");
});

test("source mutation during a stage invalidates its result and stops the pipeline", async (t) => {
  const f = fixture(t);
  await assert.rejects(
    runStages({
      ...f.options,
      execute: async (...args) => {
        await f.options.execute(...args);
        f.write("apps/go-api/internal/example/new.go");
      },
    }),
    /Inputs changed/
  );
  assert.deepEqual(f.calls, ["preflight"]);
  assert.equal(fs.existsSync(path.join(f.directory, "affected-go-tests.json")), false);
});

test("corrupt receipts rerun checks and concurrent invocations cannot overwrite evidence", async (t) => {
  const f = fixture(t);
  await runStages(f.options);
  fs.writeFileSync(path.join(f.directory, "affected-go-tests.json"), "not json");
  const result = await runStages(f.options);
  assert.equal(result[0].state, "passed");
  const release = acquireLock(f.directory);
  try {
    await assert.rejects(runStages(f.options), /already running/);
  } finally {
    release();
  }
});

test("actual subprocess failures propagate and retain their output", async (t) => {
  const f = fixture(t);
  const log = path.join(f.root, "failure.log");
  await assert.rejects(
    executeStage([process.execPath, "-e", "console.log('failure evidence'); process.exit(17)"], f.root, log, () => {}),
    /17/
  );
  assert.match(fs.readFileSync(log, "utf8"), /failure evidence/);
});

test("affected Go selection includes reverse production and test imports and fails broad for deleted packages", () => {
  const module = "/fixture/apps/go-api";
  const packages = [
    { ImportPath: "example/domain", Dir: module + "/domain" },
    { ImportPath: "example/service", Dir: module + "/service", Imports: ["example/domain"] },
    { ImportPath: "example/acceptance", Dir: module + "/acceptance", TestImports: ["example/service"] },
    { ImportPath: "example/unrelated", Dir: module + "/unrelated" },
  ];
  assert.deepEqual(affectedGoPackages(packages, ["apps/go-api/domain/value.go"], module), [
    "example/acceptance",
    "example/domain",
    "example/service",
  ]);
  assert.equal(affectedGoPackages(packages, ["apps/go-api/deleted/value.go"], module).length, 4);
  assert.equal(affectedGoPackages(packages, ["apps/go-api/go.sum"], module).length, 4);
  assert.deepEqual(affectedGoPackages(packages, ["apps/web/src/value.ts"], module), []);
  const ordered = orderedGoPackages(packages, ["apps/go-api/service/value.go"], module);
  assert.deepEqual(ordered, ["example/acceptance", "example/service", "example/domain", "example/unrelated"]);
  for (const changed of [[], ["apps/go-api/go.sum"], ["apps/go-api/deleted/value.go"]]) {
    assert.deepEqual(orderedGoPackages(packages, changed, module), packages.map((pkg) => pkg.ImportPath).toSorted());
  }
});

test("the full plan puts preflights before expensive work and retains complete coverage stages", () => {
  const plan = stagePlan("all").map(({ id }) => id);
  assert.ok(plan.indexOf("go-integration-coverage") < plan.indexOf("go-static"));
  assert.equal(plan.filter((id) => id === "go-integration-coverage").length, 1);
  assert.ok(!plan.includes("affected-go-tests"));
  assert.ok(plan.indexOf("affected-frontend-tests") < plan.indexOf("frontend-build"));
  assert.ok(plan.includes("go-integration-coverage"));
  assert.ok(plan.includes("frontend-checks-coverage"));
  assert.ok(stagePlan("go").every(({ scope }) => scope !== "frontend"));
  assert.ok(stagePlan("typescript").every(({ scope }) => scope !== "go"));
});

test("later changes cannot leave an earlier stage falsely certified", async (t) => {
  const f = fixture(t);
  await assert.rejects(
    runStages({
      ...f.options,
      execute: async (...args) => {
        await f.options.execute(...args);
        if (args[0][0] === "build")
          f.write("apps/go-api/internal/example/value.go", "package example // late change\n");
      },
    }),
    /Inputs changed after affected-go-tests/
  );
  assert.equal(JSON.parse(fs.readFileSync(path.join(f.directory, "status.json"))).state, "failed");
});

test("an interrupted subprocess cannot become reusable success", async (t) => {
  const f = fixture(t);
  const controller = new AbortController();
  const promise = executeStage(
    [process.execPath, "-e", "setInterval(() => {}, 1000)"],
    f.root,
    path.join(f.root, "interrupted.log"),
    () => controller.abort(),
    controller.signal
  );
  const timer = setTimeout(() => controller.abort(), 50);
  try {
    await assert.rejects(promise, /interrupted/);
  } finally {
    clearTimeout(timer);
  }
});

test("check.sh routes full, scoped, fresh, and status requests through the staged runner", (t) => {
  const f = fixture(t);
  f.write("check.sh", fs.readFileSync(new URL("../../check.sh", import.meta.url), "utf8"));
  f.write("scripts/dev-env.sh", "# isolated fixture\n");
  f.write("bin/node", '#!/bin/sh\nprintf "%s\\n" "$@" > "$VECTOR_FIXTURE_ARGS"\n');
  fs.chmodSync(path.join(f.root, "bin/node"), 0o755);
  const result = path.join(f.root, "arguments.txt");
  const env = {
    ...process.env,
    PATH: path.join(f.root, "bin") + path.delimiter + process.env.PATH,
    VECTOR_FIXTURE_ARGS: result,
  };
  for (const [args, expected] of [
    [["--full"], ["scripts/quality/full-check.mjs", "all"]],
    [
      ["--full", "--go", "--fresh"],
      ["scripts/quality/full-check.mjs", "go", "--fresh"],
    ],
    [
      ["--full", "--typescript"],
      ["scripts/quality/full-check.mjs", "typescript"],
    ],
    [["--status"], ["scripts/quality/full-check.mjs", "--status"]],
  ]) {
    execFileSync("bash", [path.join(f.root, "check.sh"), ...args], { env });
    assert.deepEqual(fs.readFileSync(result, "utf8").trim().split("\n"), expected);
  }
});

test("the Go gate executes the ordered package list once with full coverage and fails closed on selection errors", (t) => {
  const f = fixture(t);
  f.write("scripts/quality/go.sh", fs.readFileSync(new URL("./go.sh", import.meta.url), "utf8"));
  f.write("scripts/quality/go-tools.sh", "exit 0\n");
  f.write("scripts/dev-env.sh", "# isolated fixture\n");
  f.write(
    "bin/node",
    `#!${process.execPath}
const command = process.argv[2];
if (command.endsWith('/go-coverage-inventory.mjs')) console.log('example/changed,example/other');
if (command.endsWith('/full-check.mjs')) {
  if (process.env.VECTOR_FIXTURE_SELECTION_FAIL) process.exit(17);
  console.log('example/changed example/other');
}
`
  );
  f.write(
    "bin/go",
    `#!${process.execPath}
const fs = require('node:fs');
fs.appendFileSync(process.env.VECTOR_FIXTURE_ARGS, JSON.stringify(process.argv.slice(2)) + '\\n');
const profile = process.argv.find(arg => arg.startsWith('-coverprofile='));
fs.writeFileSync(profile.slice('-coverprofile='.length), 'mode: set\\n');
console.log('{}');
`
  );
  for (const binary of ["node", "go"]) fs.chmodSync(path.join(f.root, "bin", binary), 0o755);
  const result = path.join(f.root, "go-arguments.jsonl");
  const env = {
    ...process.env,
    PATH: path.join(f.root, "bin") + path.delimiter + process.env.PATH,
    VECTOR_FIXTURE_ARGS: result,
  };
  execFileSync("bash", [path.join(f.root, "scripts/quality/go.sh"), "--tests"], { env });
  const calls = fs
    .readFileSync(result, "utf8")
    .trim()
    .split("\n")
    .map((line) => JSON.parse(line));
  assert.equal(calls.length, 1);
  assert.deepEqual(calls[0].slice(-2), ["example/changed", "example/other"]);
  for (const argument of [
    "test",
    "-json",
    "-failfast",
    "-tags=integration",
    "-count=1",
    "-coverpkg=example/changed,example/other",
  ])
    assert.ok(calls[0].includes(argument), argument);
  fs.unlinkSync(result);
  assert.throws(
    () =>
      execFileSync("bash", [path.join(f.root, "scripts/quality/go.sh"), "--tests"], {
        env: { ...env, VECTOR_FIXTURE_SELECTION_FAIL: "1" },
      }),
    { status: 17 }
  );
  assert.equal(fs.existsSync(result), false);
});
