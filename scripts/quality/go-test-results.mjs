#!/usr/bin/env node
import { readFile } from "node:fs/promises";

const allowedActions = new Set(["start", "run", "pause", "cont", "output", "pass", "fail", "skip", "bench"]);

function fail(message) {
  throw new Error(message);
}

export function validateGoTestJSON(text, allowlist) {
  if (!text.trim()) fail("Go test JSON report is empty.");
  if (!text.endsWith("\n")) fail("Go test JSON report is truncated (missing final newline).");

  const tests = new Map();
  const packages = new Map();
  const startedPackages = new Set();
  const noTestFiles = new Set();
  let passed = 0;
  const skips = [];

  const lines = text.split("\n");
  lines.pop();
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];
    if (!line.trim()) fail(`Malformed empty JSONL record at line ${index + 1}.`);
    let event;
    try {
      event = JSON.parse(line);
    } catch (error) {
      fail(`Malformed JSON at line ${index + 1}: ${error.message}`);
    }
    if (
      !event ||
      typeof event !== "object" ||
      typeof event.Package !== "string" ||
      !event.Package ||
      !allowedActions.has(event.Action)
    ) {
      fail(`Invalid go test event at line ${index + 1}.`);
    }
    const pkg = event.Package;
    if (event.Action === "start") startedPackages.add(pkg);
    if (event.Action === "output" && typeof event.Output === "string" && event.Output.includes("[no test files]")) {
      noTestFiles.add(pkg);
    }

    if (event.Test) {
      const key = `${pkg}\0${event.Test}`;
      const state = tests.get(key) ?? { pkg, name: event.Test, started: false, terminal: false, output: "" };
      if (event.Action === "output" && typeof event.Output === "string") state.output += event.Output;
      if (event.Action === "run") {
        if (state.started || state.terminal) fail(`Duplicate test start: ${pkg} ${event.Test}.`);
        state.started = true;
      }
      if (["pass", "fail", "skip"].includes(event.Action)) {
        if (!state.started || state.terminal)
          fail(`Test terminal event without one active start: ${pkg} ${event.Test} (${event.Action}).`);
        state.terminal = true;
        if (event.Action === "pass") passed += 1;
        if (event.Action === "fail")
          fail(`Go test failed: ${pkg} ${event.Test}${state.output ? `\n${state.output.trim()}` : ""}`);
        if (event.Action === "skip") skips.push({ pkg, name: event.Test, output: state.output.trim() });
      }
      tests.set(key, state);
      continue;
    }

    if (["pass", "fail", "skip"].includes(event.Action)) {
      if (event.Action === "fail") fail(`Go package failed: ${pkg}.`);
      if (event.Action === "skip" && !noTestFiles.has(pkg))
        fail(`Package skipped without a no-test-files marker: ${pkg}.`);
      packages.set(pkg, event.Action);
    }
  }

  for (const { pkg, name, started, terminal } of tests.values()) {
    if (!started || !terminal)
      fail(`Incomplete test lifecycle: ${pkg} ${name} (started=${started}, terminal=${terminal}).`);
  }
  for (const [pkg, state] of tests.values().reduce((map, test) => map.set(test.pkg, test.terminal), new Map())) {
    if (!state) fail(`Package has a non-terminal test: ${pkg}.`);
  }
  for (const [pkg, status] of packages) {
    if (status !== "pass" && !(status === "skip" && noTestFiles.has(pkg)))
      fail(`Package did not pass: ${pkg} (${status}).`);
  }
  for (const pkg of startedPackages) {
    if (!packages.has(pkg)) fail(`Package has no terminal result: ${pkg}.`);
  }
  for (const pkg of new Set([...tests.values()].map((test) => test.pkg))) {
    if (!packages.has(pkg)) fail(`Package has no terminal result: ${pkg}.`);
  }
  const allowed = new Map((allowlist.skips ?? []).map((entry) => [`${entry.package}\0${entry.test}`, entry]));
  const novel = skips.filter((skip) => !allowed.has(`${skip.pkg}\0${skip.name}`));
  if (novel.length) fail(`Unreviewed skipped tests:\n${novel.map((skip) => `  ${skip.pkg} ${skip.name}`).join("\n")}`);
  if (passed === 0) fail("Go test report contains zero executed passing tests.");

  return { passed, skips, noTestFiles: [...noTestFiles], packages: packages.size };
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
