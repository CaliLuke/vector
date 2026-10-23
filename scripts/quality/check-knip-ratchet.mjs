#!/usr/bin/env node

import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import path from "node:path";
import process from "node:process";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../../", import.meta.url));
const baselinePath = path.join(root, ".quality", "knip-baseline.json");
const writeBaseline = process.argv.includes("--write-baseline");
const result = spawnSync("pnpm", ["exec", "knip", "--no-progress", "--reporter", "json"], {
  cwd: root,
  encoding: "utf8",
  maxBuffer: 64 * 1024 * 1024,
});
if (result.error || result.signal || ![0, 1].includes(result.status)) {
  console.error(
    `Knip failed to run cleanly${result.error ? `: ${result.error.message}` : ""}${result.signal ? ` (signal ${result.signal})` : ` (exit ${result.status})`}`
  );
  if (result.stderr) console.error(result.stderr.trim());
  process.exit(1);
}

let report;
try {
  report = JSON.parse(result.stdout);
} catch (error) {
  console.error(`Knip report could not be read: ${error.message}`);
  if (result.stderr) console.error(result.stderr.trim());
  process.exit(1);
}
if (!Array.isArray(report.issues)) {
  console.error("Knip JSON report has no issues array");
  process.exit(1);
}
const categories = [
  "binaries",
  "catalog",
  "catalogReferences",
  "dependencies",
  "devDependencies",
  "enumMembers",
  "exports",
  "files",
  "namespaceMembers",
  "optionalPeerDependencies",
  "types",
  "unlisted",
  "unresolved",
];
const findings = [];
for (const issue of report.issues) {
  if (typeof issue.file !== "string" || !issue.file) {
    console.error("Knip JSON report contains an issue without a file");
    process.exit(1);
  }
  for (const category of categories) {
    if (!Array.isArray(issue[category])) {
      console.error(`Knip JSON report has an invalid ${category} list for ${issue.file}`);
      process.exit(1);
    }
    for (const item of issue[category]) {
      const name = typeof item === "string" ? item : (item?.name ?? item?.file);
      if (typeof name !== "string" || !name) {
        console.error(`Knip JSON report has an invalid ${category} finding for ${issue.file}`);
        process.exit(1);
      }
      const finding = { category, file: issue.file, name };
      if (category === "enumMembers") {
        if (typeof item !== "object" || typeof item.namespace !== "string" || !item.namespace) {
          console.error(`Knip JSON report has an enum member without a namespace for ${issue.file}`);
          process.exit(1);
        }
        finding.namespace = item.namespace;
      }
      findings.push(finding);
    }
  }
}

if (result.status === 1 && findings.length === 0) {
  console.error("Knip exited with failure but reported no findings");
  process.exit(1);
}
if (result.status === 0 && findings.length > 0) {
  console.error("Knip reported findings but exited successfully");
  process.exit(1);
}

const keyFor = ({ category, file, name, namespace }) =>
  JSON.stringify({ category, file, ...(namespace ? { namespace } : {}), name });
const counts = new Map();
for (const finding of findings) counts.set(keyFor(finding), (counts.get(keyFor(finding)) ?? 0) + 1);

if (writeBaseline) {
  mkdirSync(path.dirname(baselinePath), { recursive: true });
  const entries = [...counts.entries()]
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([key, count]) => ({ finding: JSON.parse(key), count }));
  writeFileSync(baselinePath, `${JSON.stringify({ version: 1, findings: entries }, null, 2)}\n`);
  console.log(`Knip baseline written with ${findings.length} findings.`);
  process.exit(0);
}

let baseline;
try {
  baseline = JSON.parse(readFileSync(baselinePath, "utf8"));
} catch (error) {
  console.error(`Knip baseline is missing or invalid: ${error.message}`);
  process.exit(1);
}
if (baseline.version !== 1 || !Array.isArray(baseline.findings)) {
  console.error("Knip baseline has an unsupported format");
  process.exit(1);
}
const allowed = new Map();
for (const entry of baseline.findings) {
  if (!entry?.finding || !Number.isInteger(entry.count) || entry.count < 1) {
    console.error("Knip baseline contains an invalid finding entry");
    process.exit(1);
  }
  allowed.set(keyFor(entry.finding), entry.count);
}
const failures = [];
for (const [key, count] of counts) {
  const previousCount = allowed.get(key) ?? 0;
  if (count > previousCount) failures.push({ finding: JSON.parse(key), added: count - previousCount });
}
if (failures.length) {
  console.error(`Knip introduced ${failures.reduce((sum, item) => sum + item.added, 0)} finding(s):`);
  for (const { finding, added } of failures.slice(0, 40)) {
    console.error(`  ${finding.file}: ${finding.category} ${finding.name} (+${added})`);
  }
  if (failures.length > 40) console.error(`  ... and ${failures.length - 40} more findings`);
  process.exit(1);
}
console.log(`Knip ratchet passed: ${findings.length} findings; no finding increased.`);
