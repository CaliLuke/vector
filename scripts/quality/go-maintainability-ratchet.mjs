#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { createHash } from "node:crypto";
import { fileURLToPath } from "node:url";
import { scanComplexity } from "./go-function-inventory.mjs";

const floor = 20;
const identity = (entry) => JSON.stringify([entry.file, entry.package, entry.function]);
const hashFile = (moduleRoot, file) =>
  createHash("sha256")
    .update(fs.readFileSync(path.join(moduleRoot, file)))
    .digest("hex");

function diagnosticEntry(issue, locations, moduleRoot) {
  const match =
    /^Function name: (\S+), Cyclomatic Complexity: \d+, Halstead Volume: [\d.]+, Maintainability Index: (\d+)$/.exec(
      issue.Text ?? ""
    );
  if (issue.FromLinter !== "maintidx" || !match || !issue.Pos?.Filename || !Number.isSafeInteger(issue.Pos.Line)) {
    throw new Error("maintainability report contains an unknown or malformed diagnostic");
  }
  const file = path.relative(moduleRoot, path.resolve(moduleRoot, issue.Pos.Filename)).split(path.sep).join("/");
  const fn = locations.get(JSON.stringify([file, issue.Pos.Line]));
  const score = Number(match[2]);
  if (
    !fn ||
    !Number.isSafeInteger(score) ||
    score < 0 ||
    score >= floor ||
    !(fn.function === match[1] || fn.function.endsWith(`.${match[1]}`))
  ) {
    throw new Error(`maintainability diagnostic does not match an owned function: ${file}:${issue.Pos.Line}`);
  }
  return {
    file,
    package: fn.package,
    function: fn.function,
    score,
    ...(score === 0 ? { sourceHash: hashFile(moduleRoot, file) } : {}),
  };
}

export function maintainabilityEntries(report, functions, moduleRoot) {
  if (
    !report ||
    !("Issues" in report) ||
    (report.Issues !== null && !Array.isArray(report.Issues)) ||
    !report.Report?.Linters?.some((linter) => linter.Name === "maintidx" && linter.Enabled === true)
  )
    throw new Error("maintainability report is incomplete or maintidx was not enabled");
  const locations = new Map(functions.map((entry) => [JSON.stringify([entry.file, entry.line]), entry]));
  const entries = [];
  const seen = new Set();
  for (const issue of report.Issues ?? []) {
    const entry = diagnosticEntry(issue, locations, moduleRoot);
    if (seen.has(identity(entry))) throw new Error("duplicate maintainability function diagnostic");
    seen.add(identity(entry));
    entries.push(entry);
  }
  return entries.toSorted((left, right) => identity(left).localeCompare(identity(right)));
}

export function readMaintainabilityBaseline(filename) {
  const baseline = JSON.parse(fs.readFileSync(filename, "utf8"));
  if (baseline.version !== 1 || baseline.floor !== floor || !Array.isArray(baseline.entries)) {
    throw new Error("maintainability baseline must use version 1, floor 20, and an entries array");
  }
  const allowed = new Map();
  for (const entry of baseline.entries) {
    if (
      !entry ||
      [entry.file, entry.package, entry.function].some((value) => typeof value !== "string" || !value) ||
      !Number.isSafeInteger(entry.score) ||
      entry.score < 0 ||
      entry.score >= floor ||
      (entry.score === 0 && !/^[a-f0-9]{64}$/.test(entry.sourceHash ?? "")) ||
      allowed.has(identity(entry))
    ) {
      throw new Error("invalid or duplicate maintainability allowance");
    }
    allowed.set(identity(entry), entry);
  }
  return allowed;
}

export function compareMaintainability(entries, allowed) {
  const failures = [];
  const improvements = [];
  const seen = new Set();
  for (const entry of entries) {
    const key = identity(entry);
    const previous = allowed.get(key);
    seen.add(key);
    if (!previous || entry.score < previous.score) {
      failures.push(
        `${previous ? "REGRESSION" : "NEW"} ${entry.file} ${entry.function}: maintainability ${entry.score} < ${previous?.score ?? floor}`
      );
    } else if (entry.score === 0 && entry.sourceHash !== previous.sourceHash) {
      failures.push(`REVIEW ${entry.file} ${entry.function}: score is already zero; changed source requires review`);
    } else if (entry.score > previous.score) {
      improvements.push(`${entry.file} ${entry.function}: ${previous.score} -> ${entry.score}`);
    }
  }
  for (const key of allowed.keys()) if (!seen.has(key)) improvements.push(`resolved allowance ${key}`);
  return { failures, improvements };
}

function main() {
  const [mode, moduleRoot, inventoryPath, baselinePath, reportPath] = process.argv.slice(2);
  if (!["check", "write"].includes(mode) || !reportPath || !baselinePath || !inventoryPath || !moduleRoot) {
    throw new Error(
      "usage: go-maintainability-ratchet.mjs <check|write> <module-root> <owned-files.txt> <baseline.json> <golangci-report.json>"
    );
  }
  const files = fs
    .readFileSync(inventoryPath, "utf8")
    .trim()
    .split(/\r?\n/)
    .map((file) => file.replace(/^\.\//, ""));
  const functions = scanComplexity(moduleRoot, files);
  const entries = maintainabilityEntries(JSON.parse(fs.readFileSync(reportPath, "utf8")), functions, moduleRoot);
  if (mode === "write") {
    fs.writeFileSync(baselinePath, `${JSON.stringify({ version: 1, floor, entries }, null, 2)}\n`);
    process.stdout.write(
      `Wrote ${entries.length} maintainability allowances; review the baseline diff before committing.\n`
    );
    return;
  }
  const { failures, improvements } = compareMaintainability(entries, readMaintainabilityBaseline(baselinePath));
  if (improvements.length)
    process.stdout.write(
      `Maintainability improvements available for baseline tightening:\n${improvements.join("\n")}\n`
    );
  if (failures.length) {
    console.error(failures.join("\n"));
    process.exitCode = 1;
  } else {
    process.stdout.write(
      `Maintainability ratchet passed (${entries.length} existing allowances below ${floor}; higher scores are better).\n`
    );
  }
}

if (process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1])) {
  try {
    main();
  } catch (error) {
    console.error(error.message);
    process.exitCode = 2;
  }
}
