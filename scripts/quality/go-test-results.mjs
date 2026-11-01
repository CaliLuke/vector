#!/usr/bin/env node
import { readFile } from "node:fs/promises";

const allowedActions = new Set(["start", "run", "pause", "cont", "output", "pass", "fail", "skip", "bench"]);

function fail(message) {
  throw new Error(message);
}

function parseEvent(line, lineNumber) {
  if (!line.trim()) fail(`Malformed empty JSONL record at line ${lineNumber}.`);
  let event;
  try {
    event = JSON.parse(line);
  } catch (error) {
    fail(`Malformed JSON at line ${lineNumber}: ${error.message}`);
  }
  if (
    !event ||
    typeof event !== "object" ||
    typeof event.Package !== "string" ||
    !event.Package ||
    !allowedActions.has(event.Action)
  )
    fail(`Invalid go test event at line ${lineNumber}.`);
  return event;
}

const terminalActions = new Set(["pass", "fail", "skip"]);

class GoTestReport {
  tests = new Map();
  packages = new Map();
  startedPackages = new Set();
  noTestFiles = new Set();
  passed = 0;
  skips = [];

  accept(event) {
    const pkg = event.Package;
    if (event.Action === "start") this.startedPackages.add(pkg);
    if (event.Action === "output" && typeof event.Output === "string" && event.Output.includes("[no test files]")) {
      this.noTestFiles.add(pkg);
    }
    if (event.Test) this.acceptTest(event);
    else this.acceptPackage(event);
  }

  acceptTest(event) {
    const pkg = event.Package;
    const key = `${pkg}\0${event.Test}`;
    const state = this.tests.get(key) ?? { pkg, name: event.Test, started: false, terminal: false, output: "" };
    if (event.Action === "output" && typeof event.Output === "string") state.output += event.Output;
    if (event.Action === "run") {
      if (state.started || state.terminal) fail(`Duplicate test start: ${pkg} ${event.Test}.`);
      state.started = true;
    }
    if (terminalActions.has(event.Action)) this.finishTest(state, event.Action);
    this.tests.set(key, state);
  }

  finishTest(state, action) {
    const { pkg, name } = state;
    if (!state.started || state.terminal)
      fail(`Test terminal event without one active start: ${pkg} ${name} (${action}).`);
    state.terminal = true;
    if (action === "pass") this.passed += 1;
    if (action === "fail") fail(`Go test failed: ${pkg} ${name}${state.output ? `\n${state.output.trim()}` : ""}`);
    if (action === "skip") this.skips.push({ pkg, name, output: state.output.trim() });
  }

  acceptPackage(event) {
    if (!terminalActions.has(event.Action)) return;
    const pkg = event.Package;
    if (event.Action === "fail") fail(`Go package failed: ${pkg}.`);
    if (event.Action === "skip" && !this.noTestFiles.has(pkg))
      fail(`Package skipped without a no-test-files marker: ${pkg}.`);
    this.packages.set(pkg, event.Action);
  }

  complete(allowlist) {
    const requiredPackages = new Set(this.startedPackages);
    for (const { pkg, name, started, terminal } of this.tests.values()) {
      if (!started || !terminal)
        fail(`Incomplete test lifecycle: ${pkg} ${name} (started=${started}, terminal=${terminal}).`);
      requiredPackages.add(pkg);
    }
    for (const pkg of requiredPackages) {
      if (!this.packages.has(pkg)) fail(`Package has no terminal result: ${pkg}.`);
    }
    const allowed = new Map((allowlist.skips ?? []).map((entry) => [`${entry.package}\0${entry.test}`, entry]));
    const novel = this.skips.filter((skip) => !allowed.has(`${skip.pkg}\0${skip.name}`));
    if (novel.length)
      fail(`Unreviewed skipped tests:\n${novel.map((skip) => `  ${skip.pkg} ${skip.name}`).join("\n")}`);
    if (this.passed === 0) fail("Go test report contains zero executed passing tests.");
    return { passed: this.passed, skips: this.skips, noTestFiles: [...this.noTestFiles], packages: this.packages.size };
  }
}

export function validateGoTestJSON(text, allowlist) {
  if (!text.trim()) fail("Go test JSON report is empty.");
  if (!text.endsWith("\n")) fail("Go test JSON report is truncated (missing final newline).");
  const report = new GoTestReport();
  const lines = text.split("\n");
  lines.pop();
  for (const [index, line] of lines.entries()) report.accept(parseEvent(line, index + 1));
  return report.complete(allowlist);
}

async function main() {
  const [reportPath, allowlistPath = ".quality/go-test-skips.json"] = process.argv.slice(2);
  if (!reportPath) fail("Usage: go-test-results.mjs <go-test-jsonl-path> [allowlist-json-path]");
  const [report, allowlistText] = await Promise.all([readFile(reportPath, "utf8"), readFile(allowlistPath, "utf8")]);
  const allowlist = JSON.parse(allowlistText);
  if (allowlist.version !== 1 || !Array.isArray(allowlist.skips)) fail(`Invalid skip allowlist: ${allowlistPath}`);
  const result = validateGoTestJSON(report, allowlist);
  process.stdout.write(`Go test JSON validated: ${result.passed} passing tests; ${result.packages} packages.\n`);
  for (const skip of result.skips) {
    const entry = allowlist.skips.find((item) => item.package === skip.pkg && item.test === skip.name);
    process.stdout.write(`SKIP ${skip.pkg} ${skip.name}\n  reason: ${entry.reason}\n  scope: ${entry.scope}\n`);
  }
  for (const pkg of result.noTestFiles) process.stdout.write(`NO TEST FILES ${pkg}\n`);
}

if (process.argv[1] && import.meta.url === new URL(`file://${process.argv[1]}`).href) {
  main().catch((error) => {
    process.stderr.write(`${error.message}\n`);
    process.exitCode = 1;
  });
}
