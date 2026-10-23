import test from "node:test";
import assert from "node:assert/strict";
import { validateGoTestJSON } from "./go-test-results.mjs";

const pkg = "example.test/project/unit";
const allowlist = {
  version: 1,
  skips: [{ package: pkg, test: "TestExternal", reason: "needs service", scope: "unit fixture" }],
};
const event = (Action, extra = {}) => JSON.stringify({ Package: pkg, Action, ...extra });
const passing = `${event("run", { Test: "TestWorks" })}\n${event("pass", { Test: "TestWorks" })}\n${event("pass", { Elapsed: 0 })}\n`;

test("accepts complete passing tests and package with no test files", () => {
  const input =
    passing +
    `${event("output", { Package: "example.test/project/generated", Output: "? example.test/project/generated [no test files]\n" })}\n${event("skip", { Package: "example.test/project/generated", Elapsed: 0 })}\n`;
  const result = validateGoTestJSON(input, allowlist);
  assert.equal(result.passed, 1);
  assert.deepEqual(result.noTestFiles, ["example.test/project/generated"]);
});

test("rejects malformed, empty, and truncated JSONL", () => {
  assert.throws(() => validateGoTestJSON("", allowlist), /empty/);
  assert.throws(() => validateGoTestJSON("{bad}\n", allowlist), /Malformed JSON/);
  assert.throws(() => validateGoTestJSON(passing.trimEnd(), allowlist), /missing final newline/);
});

test("rejects test failures and package failures", () => {
  const testFailure = `${event("run", { Test: "TestBroken" })}\n${event("fail", { Test: "TestBroken" })}\n`;
  assert.throws(() => validateGoTestJSON(testFailure, allowlist), /Go test failed/);
  assert.throws(() => validateGoTestJSON(`${event("fail")}\n`, allowlist), /Go package failed/);
});

test("rejects incomplete lifecycle and zero executed passing tests", () => {
  assert.throws(
    () => validateGoTestJSON(`${event("run", { Test: "TestStuck" })}\n`, allowlist),
    /Incomplete test lifecycle/
  );
  const onlySkip = `${event("run", { Test: "TestExternal" })}\n${event("skip", { Test: "TestExternal" })}\n${event("pass")}\n`;
  assert.throws(() => validateGoTestJSON(onlySkip, allowlist), /zero executed passing tests/);
});

test("rejects skips outside the exact package and test allowlist", () => {
  const skips = `${event("run", { Test: "TestNew" })}\n${event("skip", { Test: "TestNew" })}\n${event("pass")}\n`;
  assert.throws(() => validateGoTestJSON(skips, allowlist), /Unreviewed skipped tests/);
});

test("reports an allowlisted skip while requiring other tests to pass", () => {
  const input = `${event("run", { Test: "TestExternal" })}\n${event("skip", { Test: "TestExternal" })}\n${event("run", { Test: "TestWorks" })}\n${event("pass", { Test: "TestWorks" })}\n${event("pass")}\n`;
  const result = validateGoTestJSON(input, allowlist);
  assert.equal(result.passed, 1);
  assert.deepEqual(
    result.skips.map(({ pkg: packageName, name }) => [packageName, name]),
    [[pkg, "TestExternal"]]
  );
});
