#!/usr/bin/env node

import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../../", import.meta.url));
const mode = process.argv[2];
const writeBaseline = process.argv.includes("--write-baseline");
if (!new Set(["fast", "full"]).has(mode)) {
  console.error("Usage: node scripts/quality/check-oxlint-baseline.mjs fast|full [--write-baseline]");
  process.exit(2);
}

const baselinePath = path.join(root, ".quality", `oxlint-${mode}-baseline.json`);
// Keep the largest workspace separate so parser state and type-checker heaps
// fit within the same process-tree budget. The two passes cover the entire tree.
const inputGroups = mode === "full" ? [["apps/web"], [".", "--ignore-pattern", "apps/web/**"]] : [["."]];
const reports = [];
for (const inputs of inputGroups) {
  const args = ["exec", "oxlint", "--threads", "2", "--format", "json", "--deny-warnings"];
  if (mode === "full") args.push("--type-aware", "--type-check");
  args.push(...inputs);

  const result = spawnSync("pnpm", args, {
    cwd: root,
    encoding: "utf8",
    maxBuffer: 128 * 1024 * 1024,
  });
  if (result.error || result.signal || ![0, 1].includes(result.status)) {
    console.error(
      `Oxlint failed to run cleanly${result.error ? `: ${result.error.message}` : ""}${result.signal ? ` (signal ${result.signal})` : ` (exit ${result.status})`}`
    );
    process.exit(1);
  }
  if (result.stderr?.trim()) {
    console.error(`Oxlint emitted unexpected stderr:\n${result.stderr.trim()}`);
    process.exit(1);
  }

  let report;
  try {
    report = JSON.parse(result.stdout);
  } catch (error) {
    console.error(`Oxlint report could not be read: ${error.message}`);
    if (result.stderr) console.error(result.stderr.trim());
    process.exit(1);
  }

  if (!Number.isInteger(report.number_of_files) || report.number_of_files < 1 || !Array.isArray(report.diagnostics)) {
    console.error("Oxlint JSON report has no valid file count or diagnostics array");
    process.exit(1);
  }
  if (result.status === 1 && report.diagnostics.length === 0) {
    console.error("Oxlint exited with failure but reported no diagnostics");
    process.exit(1);
  }
  const compilerDiagnostics = [];
  for (const diagnostic of report.diagnostics) {
    if (
      !diagnostic ||
      typeof diagnostic.filename !== "string" ||
      !diagnostic.filename ||
      typeof diagnostic.severity !== "string" ||
      !["error", "warning", "advice"].includes(diagnostic.severity) ||
      typeof diagnostic.code !== "string" ||
      !diagnostic.code ||
      typeof diagnostic.message !== "string"
    ) {
      console.error("Oxlint JSON report contains a malformed diagnostic");
      process.exit(1);
    }
    if (
      mode === "full" &&
      diagnostic.severity === "error" &&
      /^typescript\((?:TS\d+|tsconfig-error)\)$/.test(diagnostic.code)
    ) {
      compilerDiagnostics.push(diagnostic);
    }
  }
  if (compilerDiagnostics.length > 0) {
    console.error(`TypeScript compiler diagnostics cannot be baselined (${compilerDiagnostics.length}):`);
    for (const diagnostic of compilerDiagnostics) {
      console.error(`  ${diagnostic.filename}: ${diagnostic.code} ${diagnostic.message}`);
    }
    process.exit(1);
  }
  if (
    result.status === 0 &&
    report.diagnostics.some(({ severity }) => severity === "error" || severity === "warning")
  ) {
    console.error("Oxlint exited successfully while reporting errors or warnings");
    process.exit(1);
  }

  reports.push(report);
}
const report = {
  number_of_files: reports.reduce((total, part) => total + part.number_of_files, 0),
  diagnostics: reports.flatMap((part) => part.diagnostics),
};

const normalized = report.diagnostics.map((diagnostic) => ({
  filename: path.relative(root, path.resolve(root, diagnostic.filename)).split(path.sep).join("/"),
  severity: diagnostic.severity,
  code: diagnostic.code,
  message: diagnostic.message,
}));
if (normalized.some(({ filename }) => filename === ".." || filename.startsWith("../"))) {
  console.error("Oxlint JSON report contains a diagnostic outside the repository root");
  process.exit(1);
}
const keyFor = (diagnostic) => JSON.stringify(diagnostic);
const counts = new Map();
for (const diagnostic of normalized) {
  const key = keyFor(diagnostic);
  counts.set(key, (counts.get(key) ?? 0) + 1);
}

if (writeBaseline) {
  mkdirSync(path.dirname(baselinePath), { recursive: true });
  const baseline = [...counts.entries()]
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([key, count]) => ({ diagnostic: JSON.parse(key), count }));
  writeFileSync(
    baselinePath,
    `${JSON.stringify({ version: 1, mode, fileCount: report.number_of_files, diagnostics: baseline }, null, 2)}\n`
  );
  console.log(
    `Oxlint ${mode} baseline written with ${normalized.length} diagnostics across ${report.number_of_files} files.`
  );
  process.exit(0);
}

let baseline;
try {
  baseline = JSON.parse(readFileSync(baselinePath, "utf8"));
} catch (error) {
  console.error(`Oxlint ${mode} baseline is missing or invalid: ${error.message}`);
  process.exit(1);
}
if (baseline.version !== 1 || baseline.mode !== mode || !Array.isArray(baseline.diagnostics)) {
  console.error(`Oxlint ${mode} baseline has an unsupported format`);
  process.exit(1);
}
if (!Number.isInteger(baseline.fileCount) || baseline.fileCount < 1 || report.number_of_files < baseline.fileCount) {
  console.error(
    `Oxlint ${mode} file count dropped from ${baseline.fileCount} to ${report.number_of_files}; review whether source files were omitted and update the baseline deliberately.`
  );
  process.exit(1);
}

const allowed = new Map(baseline.diagnostics.map(({ diagnostic, count }) => [keyFor(diagnostic), count]));
const failures = [];
for (const [key, count] of counts) {
  const previousCount = allowed.get(key) ?? 0;
  if (count > previousCount) {
    failures.push({ diagnostic: JSON.parse(key), added: count - previousCount });
  }
}

if (failures.length > 0) {
  console.error(
    `Oxlint ${mode} introduced ${failures.reduce((sum, failure) => sum + failure.added, 0)} diagnostic(s):`
  );
  for (const { diagnostic, added } of failures.slice(0, 40)) {
    console.error(
      `  ${diagnostic.filename}: ${diagnostic.severity} ${diagnostic.code} (+${added}) ${diagnostic.message}`
    );
  }
  if (failures.length > 40) console.error(`  ... and ${failures.length - 40} more diagnostic groups`);
  process.exit(1);
}

const errorCount = normalized.filter(({ severity }) => severity === "error").length;
const warningCount = normalized.filter(({ severity }) => severity === "warning").length;
console.log(
  `Oxlint ${mode} ratchet passed: ${report.number_of_files} files, ${errorCount} errors and ${warningCount} warnings; no diagnostic increased.`
);
