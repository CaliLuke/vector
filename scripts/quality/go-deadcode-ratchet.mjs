#!/usr/bin/env node
import fs from "node:fs";
import { compareCountMaps, countTotal, readCountBaseline, writeCountBaseline } from "./count-baseline.mjs";

const [mode, reportPath, baselinePath] = process.argv.slice(2);
if (!["check", "write"].includes(mode) || !reportPath || !baselinePath) {
  console.error("usage: go-deadcode-ratchet.mjs <check|write> <deadcode-json> <baseline.json>");
  process.exit(2);
}
const report = JSON.parse(fs.readFileSync(reportPath, "utf8"));
if (
  !Array.isArray(report) ||
  report.some((program) => !program || typeof program.Path !== "string" || !Array.isArray(program.Funcs))
) {
  console.error(`deadcode report must be an array of programs with paths and function lists: ${reportPath}`);
  process.exit(2);
}
const counts = new Map();
for (const program of report) {
  for (const fn of program.Funcs) {
    if (!fn || typeof fn.Name !== "string") {
      console.error(`deadcode report contains a function without a name: ${reportPath}`);
      process.exit(2);
    }
    const key = JSON.stringify([program.Path, fn.Name]);
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }
}
if (mode === "write") {
  writeCountBaseline(baselinePath, counts, (length) => `wrote ${length} deadcode fingerprints to ${baselinePath}`);
  process.exit(0);
}
const allowed = readCountBaseline(baselinePath, 2, "deadcode");
const failures = compareCountMaps(counts, allowed);
if (failures.length) {
  console.error(failures.join("\n"));
  console.error(
    "Review deadcode changes, then refresh the explicit baseline with scripts/quality/go.sh --write-static-baselines."
  );
  process.exit(1);
}
console.log(`deadcode ratchet passed (${countTotal(counts)} existing findings)`);
