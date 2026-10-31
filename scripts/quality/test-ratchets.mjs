#!/usr/bin/env node

import assert from "node:assert/strict";
import { chmodSync, copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const fixtureRoot = mkdtempSync(path.join(os.tmpdir(), "vector-ratchets-"));
const qualityDirectory = path.join(fixtureRoot, ".quality");
const scriptDirectory = path.join(fixtureRoot, "scripts", "quality");
const fakeBin = path.join(fixtureRoot, "bin");

function writeJson(filename, value) {
  mkdirSync(path.dirname(filename), { recursive: true });
  writeFileSync(filename, `${JSON.stringify(value, null, 2)}\n`);
}

function runScript(script, args = [], env = {}) {
  return spawnSync(process.execPath, [path.join(scriptDirectory, script), ...args], {
    cwd: fixtureRoot,
    encoding: "utf8",
    env: {
      ...process.env,
      PATH: `${fakeBin}${path.delimiter}${process.env.PATH}`,
      ...env,
    },
  });
}

function assertFails(result, description) {
  assert.notEqual(result.status, 0, `${description} should fail`);
}

const coverageMetrics = (pct) => ({ lines: { pct }, statements: { pct }, functions: { pct }, branches: { pct } });

try {
  mkdirSync(qualityDirectory, { recursive: true });
  mkdirSync(scriptDirectory, { recursive: true });
  mkdirSync(fakeBin, { recursive: true });

  for (const script of ["check-coverage-ratchet.mjs", "check-knip-ratchet.mjs", "check-oxlint-baseline.mjs"]) {
    copyFileSync(path.join(repositoryRoot, "scripts", "quality", script), path.join(scriptDirectory, script));
  }

  const fakePnpm = path.join(fakeBin, "pnpm");
  writeFileSync(
    fakePnpm,
    `#!/bin/sh
case "$*" in
  *oxlint*)
    case "$*" in
      *--ignore-pattern*)
        if [ -n "$OX_REST_REPORT" ]; then OX_REPORT="$OX_REST_REPORT"; fi
        if [ -n "$OX_REST_STATUS" ]; then OX_STATUS="$OX_REST_STATUS"; fi
        if [ -n "$OX_REST_STDERR" ]; then OX_STDERR="$OX_REST_STDERR"; fi
        if [ -n "$OX_REST_SIGNAL" ]; then OX_SIGNAL="$OX_REST_SIGNAL"; fi
        ;;
    esac
    if [ "$OX_SIGNAL" = "1" ]; then kill -TERM "$$"; fi
    if [ -z "$OX_REPORT" ] || [ ! -f "$OX_REPORT" ]; then exit 2; fi
    if [ -n "$OX_STDERR" ]; then printf '%s\\n' "$OX_STDERR" >&2; fi
    cat "$OX_REPORT"
    exit "$OX_STATUS"
    ;;
  *knip*)
    printf '%s\\n' "$KNIP_OUTPUT"
    exit "$KNIP_STATUS"
    ;;
  *) exit 2 ;;
esac
`
  );
  chmodSync(fakePnpm, 0o755);
  const fakeGit = path.join(fakeBin, "git");
  writeFileSync(fakeGit, '#!/bin/sh\nprintf "%s\\n" "$GIT_SCOPE_FILES"\n');
  chmodSync(fakeGit, 0o755);

  const diagnostic = {
    filename: "src/a.ts",
    severity: "warning",
    code: "no-unused-vars",
    message: "unused binding",
  };
  writeJson(path.join(qualityDirectory, "oxlint-fast-baseline.json"), {
    version: 1,
    mode: "fast",
    fileCount: 1,
    diagnostics: [{ diagnostic, count: 1 }],
  });
  writeJson(path.join(qualityDirectory, "oxlint-full-baseline.json"), {
    version: 1,
    mode: "full",
    fileCount: 1,
    diagnostics: [],
  });

  const oxReport = path.join(fixtureRoot, "oxlint-report.json");
  writeJson(oxReport, { number_of_files: 1, diagnostics: [diagnostic] });
  assert.equal(runScript("check-oxlint-baseline.mjs", ["fast"], { OX_REPORT: oxReport, OX_STATUS: "1" }).status, 0);
  writeJson(oxReport, { number_of_files: 1, diagnostics: [] });
  assert.equal(runScript("check-oxlint-baseline.mjs", ["full"], { OX_REPORT: oxReport, OX_STATUS: "0" }).status, 0);
  assertFails(
    runScript("check-oxlint-baseline.mjs", ["full"], { OX_REPORT: oxReport, OX_STATUS: "1" }),
    "silent Oxlint failure"
  );
  writeJson(oxReport, {
    number_of_files: 1,
    diagnostics: [{ ...diagnostic, severity: "error", code: "typescript(TS2322)" }],
  });
  assertFails(
    runScript("check-oxlint-baseline.mjs", ["full"], { OX_REPORT: oxReport, OX_STATUS: "1" }),
    "TypeScript diagnostic must not be baselined"
  );
  writeJson(oxReport, { number_of_files: 0, diagnostics: [] });
  assertFails(
    runScript("check-oxlint-baseline.mjs", ["fast"], { OX_REPORT: oxReport, OX_STATUS: "0" }),
    "empty Oxlint file set"
  );
  const fastBaseline = path.join(qualityDirectory, "oxlint-fast-baseline.json");
  writeJson(fastBaseline, {
    version: 1,
    mode: "fast",
    fileCount: 2,
    diagnostics: [{ diagnostic, count: 1 }],
  });
  writeJson(oxReport, { number_of_files: 2, diagnostics: [] });
  assert.equal(
    runScript("check-oxlint-baseline.mjs", ["fast"], { OX_REPORT: oxReport, OX_STATUS: "0" }).status,
    0,
    "clean Oxlint report"
  );
  writeJson(oxReport, { number_of_files: 1, diagnostics: [] });
  assertFails(
    runScript("check-oxlint-baseline.mjs", ["fast"], { OX_REPORT: oxReport, OX_STATUS: "0" }),
    "Oxlint file count regression"
  );
  writeJson(fastBaseline, {
    version: 1,
    mode: "fast",
    fileCount: 1,
    diagnostics: [{ diagnostic, count: 1 }],
  });
  writeJson(oxReport, { number_of_files: 2, diagnostics: [] });
  assertFails(
    runScript("check-oxlint-baseline.mjs", ["fast"], {
      OX_REPORT: oxReport,
      OX_STATUS: "0",
      OX_STDERR: "unexpected runtime warning",
    }),
    "Oxlint stderr output"
  );
  writeJson(oxReport, { number_of_files: 1, diagnostics: [{ message: "incomplete" }] });
  assertFails(
    runScript("check-oxlint-baseline.mjs", ["fast"], { OX_REPORT: oxReport, OX_STATUS: "1" }),
    "incomplete Oxlint diagnostic"
  );

  writeJson(oxReport, { number_of_files: 1, diagnostics: [diagnostic, diagnostic] });
  assertFails(
    runScript("check-oxlint-baseline.mjs", ["fast"], { OX_REPORT: oxReport, OX_STATUS: "1" }),
    "new diagnostic regression"
  );
  writeFileSync(oxReport, "not json\n");
  assertFails(
    runScript("check-oxlint-baseline.mjs", ["fast"], { OX_REPORT: oxReport, OX_STATUS: "1" }),
    "malformed diagnostic report"
  );
  assertFails(runScript("check-oxlint-baseline.mjs", ["fast"], { OX_STATUS: "1" }), "missing diagnostic report");
  assertFails(
    runScript("check-oxlint-baseline.mjs", ["fast"], { OX_REPORT: oxReport, OX_STATUS: "2" }),
    "unexpected Oxlint exit status"
  );
  const signaledOxlint = runScript("check-oxlint-baseline.mjs", ["fast"], {
    OX_REPORT: oxReport,
    OX_SIGNAL: "1",
    OX_STATUS: "1",
  });
  assertFails(signaledOxlint, "Oxlint signal termination");
  assert.match(signaledOxlint.stderr, /signal SIGTERM/);

  const fullBaseline = path.join(qualityDirectory, "oxlint-full-baseline.json");
  const webDiagnostic = { ...diagnostic, filename: "apps/web/a.ts" };
  const restReport = path.join(fixtureRoot, "oxlint-rest-report.json");
  writeJson(fullBaseline, {
    version: 1,
    mode: "full",
    fileCount: 5,
    diagnostics: [
      { diagnostic: webDiagnostic, count: 1 },
      { diagnostic, count: 1 },
    ],
  });
  writeJson(oxReport, { number_of_files: 2, diagnostics: [webDiagnostic] });
  writeJson(restReport, { number_of_files: 3, diagnostics: [diagnostic] });
  const partitionEnv = { OX_REPORT: oxReport, OX_STATUS: "1", OX_REST_REPORT: restReport, OX_REST_STATUS: "1" };
  const combined = runScript("check-oxlint-baseline.mjs", ["full"], partitionEnv);
  assert.equal(combined.status, 0, combined.stderr);
  assert.match(combined.stdout, /5 files, 0 errors and 2 warnings/);
  assert.equal(runScript("check-oxlint-baseline.mjs", ["full", "--write-baseline"], partitionEnv).status, 0);
  const savedFullBaseline = readFileSync(fullBaseline, "utf8");
  const savedFull = JSON.parse(savedFullBaseline);
  assert.equal(savedFull.fileCount, 5);
  assert.deepEqual(
    new Set(savedFull.diagnostics.map(({ diagnostic: entry }) => entry.filename)),
    new Set(["apps/web/a.ts", "src/a.ts"])
  );

  for (const { report, status, expected } of [
    {
      report: { number_of_files: 3, diagnostics: [diagnostic, diagnostic] },
      status: "1",
      expected: /introduced 1 diagnostic/,
    },
    { report: { number_of_files: 3, diagnostics: [] }, status: "1", expected: /failure but reported no diagnostics/ },
    {
      report: { number_of_files: 3, diagnostics: [diagnostic] },
      status: "0",
      expected: /successfully while reporting errors or warnings/,
    },
    { report: { number_of_files: 0, diagnostics: [] }, status: "0", expected: /no valid file count/ },
    { report: { number_of_files: 2, diagnostics: [] }, status: "0", expected: /file count dropped from 5 to 4/ },
    {
      report: { number_of_files: 3, diagnostics: [{ message: "incomplete" }] },
      status: "1",
      expected: /malformed diagnostic/,
    },
  ]) {
    writeJson(restReport, report);
    const result = runScript("check-oxlint-baseline.mjs", ["full"], { ...partitionEnv, OX_REST_STATUS: status });
    assertFails(result, "invalid remaining-files report");
    assert.match(result.stderr, expected);
  }
  writeJson(restReport, {
    number_of_files: 3,
    diagnostics: [{ ...diagnostic, severity: "error", code: "typescript(TS2322)" }],
  });
  const compilerFailure = runScript("check-oxlint-baseline.mjs", ["full", "--write-baseline"], partitionEnv);
  assertFails(compilerFailure, "remaining-files compiler error");
  assert.match(compilerFailure.stderr, /compiler diagnostics cannot be baselined/);
  assert.equal(readFileSync(fullBaseline, "utf8"), savedFullBaseline);
  writeJson(restReport, { number_of_files: 3, diagnostics: [diagnostic] });
  for (const { env, expected } of [
    { env: { OX_REST_STATUS: "2" }, expected: /failed to run cleanly/ },
    { env: { OX_REST_SIGNAL: "1" }, expected: /signal SIGTERM/ },
    { env: { OX_REST_STDERR: "unexpected warning" }, expected: /unexpected stderr/ },
  ]) {
    const result = runScript("check-oxlint-baseline.mjs", ["full"], { ...partitionEnv, ...env });
    assertFails(result, "remaining-files process failure");
    assert.match(result.stderr, expected);
  }
  writeFileSync(restReport, "invalid JSON");
  const malformedRest = runScript("check-oxlint-baseline.mjs", ["full"], partitionEnv);
  assertFails(malformedRest, "remaining-files invalid JSON");
  assert.match(malformedRest.stderr, /report could not be read/);

  const knipBaseline = path.join(qualityDirectory, "knip-baseline.json");
  const knipReport = (dependencies = []) => ({
    issues: [
      {
        file: "apps/web/package.json",
        binaries: [],
        catalog: [],
        catalogReferences: [],
        dependencies,
        devDependencies: [],
        enumMembers: [],
        exports: [],
        files: [],
        namespaceMembers: [],
        optionalPeerDependencies: [],
        types: [],
        unlisted: [],
        unresolved: [],
      },
    ],
  });
  writeJson(knipBaseline, {
    version: 1,
    findings: [{ finding: { category: "dependencies", file: "apps/web/package.json", name: "old-dep" }, count: 1 }],
  });
  assert.equal(
    runScript("check-knip-ratchet.mjs", [], {
      KNIP_OUTPUT: JSON.stringify(knipReport([{ name: "old-dep", line: 2 }])),
      KNIP_STATUS: "1",
    }).status,
    0
  );
  assertFails(
    runScript("check-knip-ratchet.mjs", [], {
      KNIP_OUTPUT: JSON.stringify(knipReport([{ name: "new-dep", line: 3 }])),
      KNIP_STATUS: "1",
    }),
    "new Knip finding"
  );
  assertFails(
    runScript("check-knip-ratchet.mjs", [], { KNIP_OUTPUT: "not json", KNIP_STATUS: "1" }),
    "malformed Knip report"
  );
  assertFails(
    runScript("check-knip-ratchet.mjs", [], { KNIP_OUTPUT: JSON.stringify({ issues: [] }), KNIP_STATUS: "1" }),
    "silent Knip failure"
  );
  assertFails(
    runScript("check-knip-ratchet.mjs", [], { KNIP_OUTPUT: JSON.stringify(knipReport()), KNIP_STATUS: "1" }),
    "Knip failure with empty issue objects"
  );
  writeJson(knipBaseline, { version: 1, findings: "invalid" });
  assertFails(
    runScript("check-knip-ratchet.mjs", [], { KNIP_OUTPUT: JSON.stringify(knipReport()), KNIP_STATUS: "0" }),
    "malformed Knip baseline"
  );
  rmSync(knipBaseline);
  assertFails(
    runScript("check-knip-ratchet.mjs", [], { KNIP_OUTPUT: JSON.stringify(knipReport()), KNIP_STATUS: "0" }),
    "missing Knip baseline"
  );
  writeJson(knipBaseline, {
    version: 1,
    findings: [
      { finding: { category: "enumMembers", file: "src/enums.ts", namespace: "OldEnum", name: "Shared" }, count: 1 },
    ],
  });
  const enumReport = knipReport();
  enumReport.issues[0].file = "src/enums.ts";
  enumReport.issues[0].enumMembers = [{ namespace: "NewEnum", name: "Shared" }];
  assertFails(
    runScript("check-knip-ratchet.mjs", [], {
      KNIP_OUTPUT: JSON.stringify(enumReport),
      KNIP_STATUS: "1",
    }),
    "same Knip enum member name in another namespace"
  );
  enumReport.issues[0].enumMembers = [{ name: "Shared" }];
  assertFails(
    runScript("check-knip-ratchet.mjs", [], {
      KNIP_OUTPUT: JSON.stringify(enumReport),
      KNIP_STATUS: "1",
    }),
    "Knip enum member without namespace"
  );

  const coverageSummary = path.join(fixtureRoot, "coverage", "coverage-summary.json");
  const coverageBaseline = path.join(qualityDirectory, "coverage-baseline.json");
  writeJson(coverageSummary, { "src/a.ts": coverageMetrics(90), total: coverageMetrics(90) });
  writeJson(coverageBaseline, {
    version: 1,
    files: { "src/a.ts": { lines: 90, statements: 90, functions: 90, branches: 90 } },
  });
  const coverageArgs = [
    "--root",
    fixtureRoot,
    "--summary",
    "coverage/coverage-summary.json",
    "--baseline",
    ".quality/coverage-baseline.json",
    "--new-file-min",
    "80",
  ];
  assert.equal(runScript("check-coverage-ratchet.mjs", coverageArgs).status, 0);

  writeJson(coverageSummary, { "src/a.ts": coverageMetrics(80), total: coverageMetrics(80) });
  assertFails(runScript("check-coverage-ratchet.mjs", coverageArgs), "coverage regression");
  writeJson(coverageSummary, {
    "src/a.ts": coverageMetrics(90),
    "src/new.ts": coverageMetrics(70),
    total: coverageMetrics(80),
  });
  assertFails(runScript("check-coverage-ratchet.mjs", coverageArgs), "new uncovered source file");
  writeFileSync(coverageSummary, "not json\n");
  assertFails(runScript("check-coverage-ratchet.mjs", coverageArgs), "malformed coverage report");
  rmSync(coverageSummary);
  assertFails(runScript("check-coverage-ratchet.mjs", coverageArgs), "missing coverage report");

  const suiteRoot = path.join(fixtureRoot, "packages", "app");
  mkdirSync(path.join(suiteRoot, "coverage"), { recursive: true });
  mkdirSync(path.join(suiteRoot, "src"), { recursive: true });
  writeFileSync(path.join(suiteRoot, "src", "a.ts"), "export const value = 1;\n");
  const scopedSummary = path.join(suiteRoot, "coverage", "coverage-summary.json");
  const scopedBaseline = path.join(suiteRoot, ".quality", "coverage-baseline.json");
  const scopeFile = path.join(fixtureRoot, "coverage-scope.json");
  writeJson(scopeFile, {
    coveredSuites: {
      "packages/app": {
        sourcePatterns: ["src/**/*.{ts,tsx}"],
        exclusions: ["src/**/*.test.*", "src/**/types.ts", "src/**/*.stories.*"],
      },
    },
  });
  writeJson(scopedSummary, { "src/a.ts": coverageMetrics(90), total: coverageMetrics(90) });
  writeJson(scopedBaseline, {
    version: 1,
    files: { "src/a.ts": { lines: 90, statements: 90, functions: 90, branches: 90 } },
  });
  const scopeArgs = [
    "--root",
    suiteRoot,
    "--summary",
    "coverage/coverage-summary.json",
    "--baseline",
    ".quality/coverage-baseline.json",
    "--scope-file",
    scopeFile,
    "--scope-key",
    "packages/app",
    "--new-file-min",
    "80",
  ];
  const scopeEnvironment = {
    GIT_SCOPE_FILES:
      "packages/app/src/a.ts\npackages/app/src/a.test.ts\npackages/app/src/types.ts\npackages/app/src/a.stories.tsx\npackages/app/src/deleted.ts",
  };
  assert.equal(runScript("check-coverage-ratchet.mjs", scopeArgs, scopeEnvironment).status, 0);
  writeJson(scopedSummary, { total: coverageMetrics(90) });
  assertFails(
    runScript("check-coverage-ratchet.mjs", scopeArgs, scopeEnvironment),
    "owned source missing from coverage report"
  );

  process.stdout.write(
    "Ratchet fixture tests passed (diagnostics, per-finding Knip debt, and coverage regressions fail closed).\n"
  );
} finally {
  rmSync(fixtureRoot, { recursive: true, force: true });
}
