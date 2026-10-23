import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

const scripts = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(scripts, "../..");

function fixture(run) {
  const directory = fs.mkdtempSync(path.join(os.tmpdir(), "vector-go-ratchet-"));
  try {
    run(directory);
  } finally {
    fs.rmSync(directory, { recursive: true, force: true });
  }
}

function write(directory, name, value) {
  const filename = path.join(directory, name);
  fs.mkdirSync(path.dirname(filename), { recursive: true });
  fs.writeFileSync(filename, typeof value === "string" ? value : `${JSON.stringify(value)}\n`);
  return filename;
}

function invoke(script, args) {
  return spawnSync(process.execPath, [path.join(scripts, script), ...args], { encoding: "utf8" });
}

test("lint ratchet rejects malformed reports and new diagnostics, and normalizes line-only message changes", () => {
  fixture((directory) => {
    const baseline = write(directory, "lint-baseline.json", {
      version: 1,
      issues: [{ fingerprint: ["nilerr", "sample_test.go", "error is non-nil (line #) but returns nil"], count: 1 }],
    });
    const report = write(directory, "lint.json", {
      Issues: [
        {
          FromLinter: "nilerr",
          Text: "error is non-nil (line 24) but returns nil",
          Pos: { Filename: "sample_test.go" },
        },
      ],
    });
    const unchanged = invoke("go-lint-ratchet.mjs", ["check", report, baseline, directory]);
    assert.equal(unchanged.status, 0, unchanged.stderr);

    write(directory, "lint.json", {
      Issues: [{ FromLinter: "errcheck", Text: "ignored error", Pos: { Filename: "sample_test.go" } }],
    });
    const novel = invoke("go-lint-ratchet.mjs", ["check", report, baseline, directory]);
    assert.equal(novel.status, 1);
    assert.match(novel.stderr, /NEW/);

    write(directory, "lint.json", { unexpected: [] });
    const malformed = invoke("go-lint-ratchet.mjs", ["check", report, baseline, directory]);
    assert.notEqual(malformed.status, 0);
  });
});

test("deadcode ratchet rejects malformed reports and newly unreachable functions", () => {
  fixture((directory) => {
    const baseline = write(directory, "deadcode-baseline.json", { version: 1, entries: [] });
    const report = write(directory, "deadcode.json", [{ Path: "example/module", Funcs: [{ Name: "unused" }] }]);
    const novel = invoke("go-deadcode-ratchet.mjs", ["check", report, baseline]);
    assert.equal(novel.status, 1);
    assert.match(novel.stderr, /NEW/);
    write(directory, "deadcode.json", { malformed: true });
    const malformed = invoke("go-deadcode-ratchet.mjs", ["check", report, baseline]);
    assert.equal(malformed.status, 2);
  });
});

test("dupl ratchet rejects malformed records and new clone pairs", () => {
  fixture((directory) => {
    write(directory, "a.go", "func a() {\n return 1\n}\n");
    write(directory, "b.go", "func b() {\n return 1\n}\n");
    const baseline = write(directory, "dupl-baseline.json", { version: 1, entries: [] });
    const report = write(directory, "dupl.txt", "a.go:1-3: duplicate of b.go:1-3\n");
    const novel = invoke("go-dupl-ratchet.mjs", ["check", report, baseline, directory]);
    assert.equal(novel.status, 1);
    assert.match(novel.stderr, /NEW/);
    write(directory, "dupl.txt", "not a dupl record\n");
    const malformed = invoke("go-dupl-ratchet.mjs", ["check", report, baseline, directory]);
    assert.equal(malformed.status, 2);
  });
});

test("coverage ratchet rejects package omissions, regressions, and invalid floors", () => {
  fixture((directory) => {
    write(directory, "go.mod", "module example.test/fixture\n\ngo 1.27.3\n");
    write(directory, "p/p.go", "package p\nfunc F(value bool) int {\n if value { return 1 }\n return 0\n}\n");
    write(directory, "p/p_test.go", 'package p\nimport "testing"\nfunc TestF(t *testing.T) {}\n');
    write(directory, "q/q.go", "package q\nfunc G() int { return 1 }\n");
    write(directory, "q/q_test.go", 'package q\nimport "testing"\nfunc TestG(t *testing.T) {}\n');
    const baseline = write(directory, "coverage-baseline.json", {
      version: 1,
      packages: {
        "example.test/fixture/p": { statements: 2, percent: 100 },
        "example.test/fixture/q": { statements: 1, percent: 100 },
      },
    });
    const profile = write(directory, "coverage.out", "mode: set\nexample.test/fixture/p/p.go:2.1,4.2 2 1\n");
    const omitted = invoke("go-coverage-ratchet.mjs", ["check", profile, baseline, directory]);
    assert.equal(omitted.status, 2);
    assert.match(omitted.stderr, /missing owned test packages/);

    write(
      directory,
      "coverage.out",
      [
        "mode: set",
        "example.test/fixture/p/p.go:2.1,3.2 1 1",
        "example.test/fixture/p/p.go:3.2,4.2 1 0",
        "example.test/fixture/q/q.go:2.1,2.25 1 1",
        "",
      ].join("\n")
    );
    const regression = invoke("go-coverage-ratchet.mjs", ["check", profile, baseline, directory]);
    assert.equal(regression.status, 1);
    assert.match(regression.stderr, /below 100%/);

    write(directory, "coverage-baseline.json", {
      version: 1,
      packages: { "example.test/fixture/p": { statements: 2, percent: "Infinity" } },
    });
    const invalidFloor = invoke("go-coverage-ratchet.mjs", ["check", profile, baseline, directory]);
    assert.equal(invalidFloor.status, 2);
    assert.match(invalidFloor.stderr, /finite percent/);
  });
});
