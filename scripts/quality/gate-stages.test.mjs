import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { execFileSync } from "node:child_process";
import test from "node:test";
import { acquireLock, executeStage, inputFingerprint, relevantEnvironment, runStages } from "./gate-stages.mjs";
import { affectedGoPackages, orderedGoPackages, stagePlan } from "./full-check.mjs";

await test("workspace commands keep Node-compatible executables on the pinned Bun runtime", () => {
  const root = fileURLToPath(new URL("../../", import.meta.url));
  const expected = fs.readFileSync(path.join(root, ".bun-version"), "utf8").trim();
  for (const parent of ["apps", "packages"]) {
    for (const name of fs.readdirSync(path.join(root, parent))) {
      const cwd = path.join(root, parent, name);
      if (!fs.existsSync(path.join(cwd, "package.json"))) continue;
      const actual = execFileSync("bun", ["run", "node", "-p", "process.versions.bun"], {
        cwd,
        encoding: "utf8",
      }).trim();
      assert.equal(actual, expected, `${parent}/${name} must use Bun for dependency executables`);
    }
  }
});

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
  write("bun.lock");
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

await test("manual and Git hook entry points use the same tool path", (t) => {
  const f = fixture(t);
  const root = fs.realpathSync(f.root);
  f.write("check.sh", fs.readFileSync(new URL("../../check.sh", import.meta.url)));
  f.write("fixture-bin/bun", '#!/bin/sh\nprintf "%s" "$PATH"\n');
  fs.chmodSync(path.join(f.root, "fixture-bin/bun"), 0o755);
  const toolPath = `${f.root}/fixture-bin:${process.env.PATH}`;
  const gitPath = execFileSync("git", ["--exec-path"], { encoding: "utf8" }).trim();
  const run = (PATH) =>
    execFileSync("bash", ["check.sh", "--status"], {
      cwd: f.root,
      env: { ...process.env, PATH },
      encoding: "utf8",
    });
  const manual = run(toolPath);
  assert.equal(manual, `${root}/node_modules/.bin:${toolPath}`);
  assert.equal(run(`node_modules/.bin:${gitPath}:${toolPath}`), manual);
  assert.equal(run(`${root}/node_modules/.bin:${gitPath}:${toolPath}`), manual);
});

await test("a failing affected test prevents all expensive stages and never creates a success receipt", async (t) => {
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

await test("a Go fix reruns its preflight while reusing the unchanged frontend result", async (t) => {
  const f = fixture(t);
  f.write("docs/tech-debt-catalog.md", "Backend debt\n");
  await runStages({ ...f.options, identity: { ...f.options.identity, head: "before-fix" } });
  f.write("apps/go-api/internal/example/value.go", "package example\n// fixed\n");
  const formatting = { id: "frontend-format", scope: "all", command: ["format"] };
  const beforeDocumentation = await inputFingerprint(f.root, formatting, f.options.identity);
  f.write("docs/tech-debt-catalog.md", "Backend debt retired\n");
  assert.notEqual(beforeDocumentation, await inputFingerprint(f.root, formatting, f.options.identity));
  const result = await runStages({ ...f.options, identity: { ...f.options.identity, head: "after-fix" } });
  assert.deepEqual(f.calls, ["preflight", "build", "preflight"]);
  assert.deepEqual(
    result.map(({ state }) => state),
    ["passed", "reused"]
  );
  f.write("apps/web/src/value.ts", "export const value = 2;\n");
  await runStages(f.options);
  assert.equal(f.calls.at(-1), "build");
  const beforeOtherDocumentation = await inputFingerprint(f.root, f.stages[1], f.options.identity);
  f.write("docs/mcp.md", "Other documentation remains an input until reviewed\n");
  assert.notEqual(beforeOtherDocumentation, await inputFingerprint(f.root, f.stages[1], f.options.identity));
});

await test("comparison base changes rerun selection without repeating full checks", async (t) => {
  const f = fixture(t);
  const selection = {
    id: "affected-frontend-tests",
    scope: "frontend",
    command: ["selection"],
    dependsOnChangedFiles: true,
  };
  assert.equal(stagePlan("typescript").find(({ id }) => id === selection.id).dependsOnChangedFiles, true);
  const identity = (comparisonBase) => ({
    ...f.options.identity,
    comparisonBase,
    environment: relevantEnvironment({ VECTOR_CHECK_BASE: comparisonBase, VITE_API_URL: "https://api.example.test" }),
  });
  const options = { ...f.options, stages: [...f.stages, selection] };
  await runStages({ ...options, identity: identity("first-base") });
  const result = await runStages({ ...options, identity: identity("second-base") });
  assert.deepEqual(
    result.map(({ state }) => state),
    ["reused", "reused", "passed"]
  );
  assert.deepEqual(f.calls, ["preflight", "build", "selection", "selection"]);
  assert.notEqual(
    await inputFingerprint(f.root, f.stages[1], identity("first-base")),
    await inputFingerprint(f.root, f.stages[1], {
      ...identity("first-base"),
      environment: { VITE_API_URL: "https://changed.example.test" },
    })
  );
});

await test("untracked sources, deleted sources, and lockfile changes invalidate affected results", async (t) => {
  const f = fixture(t);
  const stage = f.stages[1];
  const first = await inputFingerprint(f.root, stage, f.options.identity);
  f.write("apps/web/src/new.ts");
  const added = await inputFingerprint(f.root, stage, f.options.identity);
  assert.notEqual(first, added);
  fs.unlinkSync(path.join(f.root, "apps/web/src/value.ts"));
  const deleted = await inputFingerprint(f.root, stage, f.options.identity);
  assert.notEqual(added, deleted);
  f.write("bun.lock", "changed dependencies\n");
  assert.notEqual(deleted, await inputFingerprint(f.root, stage, f.options.identity));
});

await test("toolchain, environment and checker configuration changes invalidate successful results", async (t) => {
  const f = fixture(t);
  const environment = relevantEnvironment({ BUN_OPTIONS: "--conditions=test", JSC_useJIT: "false" });
  assert.deepEqual(environment, { BUN_OPTIONS: "--conditions=test", JSC_useJIT: "false" });
  await runStages(f.options);
  await runStages({ ...f.options, identity: { tools: "v2", environment } });
  assert.deepEqual(f.calls, ["preflight", "build", "preflight", "build"]);
  f.write("scripts/quality/go.sh", "changed gate\n");
  const result = await runStages({
    ...f.options,
    identity: { tools: "v2", environment },
  });
  assert.deepEqual(
    result.map(({ state }) => state),
    ["passed", "reused"]
  );
});

await test("missing or altered artifacts cannot certify a cached stage", async (t) => {
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

await test("fresh runs and expired results execute tests again", async (t) => {
  const f = fixture(t);
  await runStages({ ...f.options, now: () => 1000 });
  await runStages({ ...f.options, now: () => 2000, maxAgeMs: 500 });
  await runStages({ ...f.options, now: () => 2001, reuse: false });
  assert.equal(f.calls.length, 6);
});

await test("a failed fresh recheck removes an earlier success even if inputs are unchanged", async (t) => {
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

await test("source mutation during a stage invalidates its result and stops the pipeline", async (t) => {
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

await test("corrupt receipts rerun checks and concurrent invocations cannot overwrite evidence", async (t) => {
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

await test("actual subprocess failures propagate and retain their output", async (t) => {
  const f = fixture(t);
  const log = path.join(f.root, "failure.log");
  await assert.rejects(
    executeStage([process.execPath, "-e", "console.log('failure evidence'); process.exit(17)"], f.root, log, () => {}),
    /17/
  );
  assert.match(fs.readFileSync(log, "utf8"), /failure evidence/);
});

await test("affected Go selection includes reverse production and test imports and fails broad for deleted packages", () => {
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

await test("the full plan puts preflights before expensive work and retains complete coverage stages", () => {
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

await test("later changes cannot leave an earlier stage falsely certified", async (t) => {
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

await test("an interrupted subprocess cannot become reusable success", async (t) => {
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

await test("check.sh routes full, scoped, fresh, and status requests through the staged runner", (t) => {
  const f = fixture(t);
  f.write("check.sh", fs.readFileSync(new URL("../../check.sh", import.meta.url), "utf8"));
  f.write("scripts/dev-env.sh", "# isolated fixture\n");
  f.write("bin/bun", '#!/bin/sh\nprintf "%s\\n" "$@" > "$VECTOR_FIXTURE_ARGS"\n');
  fs.chmodSync(path.join(f.root, "bin/bun"), 0o755);
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

await test("the Go gate executes the ordered package list once with full coverage and fails closed on selection errors", (t) => {
  const f = fixture(t);
  f.write("scripts/quality/go.sh", fs.readFileSync(new URL("./go.sh", import.meta.url), "utf8"));
  f.write("scripts/quality/go-tools.sh", "exit 0\n");
  f.write("scripts/dev-env.sh", "# isolated fixture\n");
  f.write(
    "bin/bun",
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
  for (const binary of ["bun", "go"]) fs.chmodSync(path.join(f.root, "bin", binary), 0o755);
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

await test("manual checks and Husky use the same tool path without removing caller tool directories", (t) => {
  const f = fixture(t);
  f.write("check.sh", fs.readFileSync(new URL("../../check.sh", import.meta.url), "utf8"));
  f.write("scripts/dev-env.sh", "# isolated fixture\n");
  f.write("bin/bun", '#!/bin/sh\nprintf "%s" "$PATH" > "$VECTOR_FIXTURE_ARGS"\n');
  fs.chmodSync(path.join(f.root, "bin/bun"), 0o755);
  const result = path.join(f.root, "tool-path.txt");
  const callerPath = path.join(f.root, "bin") + path.delimiter + process.env.PATH;
  const paths = [callerPath, "node_modules/.bin" + path.delimiter + callerPath];
  const observed = paths.map((PATH) => {
    execFileSync("bash", [path.join(f.root, "check.sh"), "--status"], {
      cwd: f.root,
      env: { ...process.env, PATH, VECTOR_FIXTURE_ARGS: result },
    });
    return fs.readFileSync(result, "utf8");
  });
  assert.equal(observed[0], path.join(f.root, "node_modules/.bin") + path.delimiter + callerPath);
  assert.equal(observed[1], observed[0]);
});
