import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import {
  compareMaintainability,
  maintainabilityEntries,
  readMaintainabilityBaseline,
} from "./go-maintainability-ratchet.mjs";
import { parseComplexity, scanComplexity } from "./go-function-inventory.mjs";

const bin = new URL("../../.quality/bin/", import.meta.url).pathname;
const entry = { file: "work.go", package: "example", function: "(*Worker).Work", score: 15 };
const key = JSON.stringify([entry.file, entry.package, entry.function]);

function fixture(t) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "vector-maintainability-"));
  t.after(() => fs.rmSync(directory, { recursive: true, force: true }));
  return directory;
}

void test("numeric floors reject new and worsening scores while retaining improvements", () => {
  const allowed = new Map([[key, entry]]);
  assert.deepEqual(compareMaintainability([entry], allowed).failures, []);
  assert.match(compareMaintainability([{ ...entry, score: 10 }], allowed).failures[0], /REGRESSION.*10 < 15/);
  assert.match(compareMaintainability([{ ...entry, function: "New", score: 19 }], allowed).failures[0], /NEW.*19 < 20/);
  const improved = compareMaintainability([{ ...entry, score: 18 }], allowed);
  assert.deepEqual(improved.failures, []);
  assert.match(improved.improvements[0], /15 -> 18/);
  assert.equal(compareMaintainability([], allowed).improvements.length, 1);
  const zero = { ...entry, score: 0, sourceHash: "a".repeat(64) };
  assert.match(
    compareMaintainability([{ ...zero, sourceHash: "b".repeat(64) }], new Map([[key, zero]])).failures[0],
    /REVIEW.*already zero/
  );
});

void test("baseline rejects weakened floors, duplicate functions, and unreviewable zero scores", (t) => {
  const filename = path.join(fixture(t), "baseline.json");
  const write = (value) => fs.writeFileSync(filename, JSON.stringify(value));
  write({ version: 1, floor: 20, entries: [entry] });
  assert.equal(readMaintainabilityBaseline(filename).get(key).score, 15);
  for (const baseline of [
    { version: 1, floor: 10, entries: [entry] },
    { version: 1, floor: 20, entries: [entry, entry] },
    { version: 1, floor: 20, entries: [{ ...entry, score: 0 }] },
  ]) {
    write(baseline);
    assert.throws(() => readMaintainabilityBaseline(filename));
  }
});

void test("real maintidx diagnostics resolve receiver identity and reject an unbaselined function", (t) => {
  const directory = fixture(t);
  fs.writeFileSync(path.join(directory, "go.mod"), "module example.test/maintainability\n\ngo 1.27.3\n");
  fs.writeFileSync(
    path.join(directory, "work.go"),
    `package example\ntype Worker struct{}\nfunc (*Worker) Work(n int) int {\nresult := 0\n${Array.from({ length: 100 }, (_, n) => `if n == ${n} { result += ${n} }`).join("\n")}\nreturn result\n}\n`
  );
  const reportPath = path.join(directory, "report.json");
  const result = spawnSync(
    path.join(bin, "golangci-lint"),
    [
      "run",
      "--no-config",
      "--allow-parallel-runners",
      "--enable-only",
      "maintidx",
      "--issues-exit-code=0",
      `--output.json.path=${reportPath}`,
      "./...",
    ],
    {
      cwd: directory,
      encoding: "utf8",
      timeout: 60_000,
      env: { ...process.env, GOMAXPROCS: "2", GOLANGCI_LINT_CACHE: path.join(directory, "lint-cache") },
    }
  );
  assert.equal(result.status, 0, result.stderr);
  const functions = scanComplexity(directory, ["work.go"], path.join(bin, "gocyclo"));
  const entries = maintainabilityEntries(JSON.parse(fs.readFileSync(reportPath, "utf8")), functions, directory);
  assert.equal(entries.length, 1);
  assert.equal(entries[0].function, "(*Worker).Work");
  assert.ok(entries[0].score < 20);
  assert.match(compareMaintainability(entries, new Map()).failures[0], /NEW/);
});

void test("disabled analysis, malformed diagnostics, missing tools and truncated inventories fail closed", (t) => {
  const directory = fixture(t);
  assert.throws(() => maintainabilityEntries({ Issues: [] }, [], directory), /incomplete/);
  const report = {
    Issues: [{ FromLinter: "maintidx", Text: "unknown format" }],
    Report: { Linters: [{ Name: "maintidx", Enabled: true }] },
  };
  assert.throws(() => maintainabilityEntries(report, [], directory), /malformed/);
  fs.writeFileSync(path.join(directory, "work.go"), "package example\nfunc Work() {}\n");
  assert.throws(() => scanComplexity(directory, ["work.go"], path.join(directory, "missing-tool")));
  assert.throws(() => scanComplexity(directory, ["absent.go"], path.join(bin, "gocyclo")));
  assert.throws(() => parseComplexity("1 example Work work.go:2:1\n", new Set(["work.go"])), /incomplete/);
  fs.writeFileSync(path.join(directory, "work.go"), "package example\nfunc Broken(\n");
  assert.throws(() => scanComplexity(directory, ["work.go"], path.join(bin, "gocyclo")));
});
