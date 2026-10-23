#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const [mode, inputPath, baselinePath, moduleRoot] = process.argv.slice(2);
if (!["check", "write"].includes(mode) || !inputPath || !baselinePath || !moduleRoot) {
  console.error("usage: go-lint-ratchet.mjs <check|write> <golangci-json> <baseline.json> <module-root>");
  process.exit(2);
}

function readJSON(filename) {
  return JSON.parse(fs.readFileSync(filename, "utf8"));
}

function normalizeText(text) {
  return text.replace(/\bline \d+\b/g, "line #");
}

function diagnosticMap(report) {
  if (
    !report ||
    typeof report !== "object" ||
    !("Issues" in report) ||
    (report.Issues !== null && !Array.isArray(report.Issues))
  ) {
    throw new Error("golangci report must contain an Issues array or null");
  }
  const counts = new Map();
  for (const issue of report.Issues ?? []) {
    if (!issue.FromLinter || !issue.Pos?.Filename || typeof issue.Text !== "string" || !issue.Text) {
      throw new Error("golangci report contains an incomplete issue");
    }
    const file = path.relative(moduleRoot, path.resolve(moduleRoot, issue.Pos.Filename)).split(path.sep).join("/");
    const text = normalizeText(issue.Text);
    const key = JSON.stringify([issue.FromLinter, file, text]);
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }
  return counts;
}

const current = diagnosticMap(readJSON(inputPath));
if (mode === "write") {
  const issues = [...current.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([key, count]) => ({
      fingerprint: JSON.parse(key),
      count,
    }));
  fs.mkdirSync(path.dirname(baselinePath), { recursive: true });
  fs.writeFileSync(baselinePath, `${JSON.stringify({ version: 1, issues }, null, 2)}\n`);
  console.log(
    `wrote ${issues.length} diagnostic fingerprints (${[...current.values()].reduce((sum, n) => sum + n, 0)} occurrences) to ${baselinePath}`
  );
  process.exit(0);
}

const baseline = readJSON(baselinePath);
if (baseline.version !== 1 || !Array.isArray(baseline.issues)) {
  console.error(`unsupported lint baseline format: ${baselinePath}`);
  process.exit(2);
}
const allowed = new Map();
for (const { fingerprint, count } of baseline.issues) {
  if (
    !Array.isArray(fingerprint) ||
    fingerprint.length !== 3 ||
    fingerprint.some((part) => typeof part !== "string") ||
    !Number.isSafeInteger(count) ||
    count <= 0
  ) {
    throw new Error(`invalid lint baseline entry: ${JSON.stringify({ fingerprint, count })}`);
  }
  const key = JSON.stringify([fingerprint[0], fingerprint[1], normalizeText(fingerprint[2])]);
  // Older baselines can contain separate entries whose only difference was a
  // source line number embedded in the diagnostic text. Merge those counts.
  allowed.set(key, (allowed.get(key) ?? 0) + count);
}
const newIssues = [];
const staleIssues = [];
for (const [key, count] of current) {
  const permitted = allowed.get(key) ?? 0;
  if (count > permitted) newIssues.push({ key, count: count - permitted });
}
for (const [key, count] of allowed) {
  const observed = current.get(key) ?? 0;
  if (count > observed) staleIssues.push({ key, count: count - observed });
}
if (newIssues.length || staleIssues.length) {
  for (const { key, count } of newIssues) console.error(`NEW x${count}: ${key}`);
  for (const { key, count } of staleIssues) console.error(`BASELINE CAN SHRINK x${count}: ${key}`);
  console.error("Review changes, then refresh the explicit baseline with scripts/quality/go.sh --write-lint-baseline.");
  process.exit(1);
}
console.log(
  `golangci-lint ratchet passed (${[...current.values()].reduce((sum, n) => sum + n, 0)} existing diagnostics)`
);
