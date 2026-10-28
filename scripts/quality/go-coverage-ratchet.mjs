#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { createInterface } from "node:readline";
import { coverageInventory } from "./go-coverage-inventory.mjs";

const [mode, profilePath, baselinePath, moduleRoot] = process.argv.slice(2);
if (!["check", "write"].includes(mode) || !profilePath || !baselinePath || !moduleRoot) {
  console.error("usage: go-coverage-ratchet.mjs <check|write> <coverprofile> <baseline.json> <module-root>");
  process.exit(2);
}

const { filePackages, expectedPackages } = coverageInventory(moduleRoot);
// Cross-package profiles repeat blocks from each test binary and can exceed
// V8's maximum string size. Retain unique blocks, not the complete input text.
const profileLines = createInterface({ input: fs.createReadStream(profilePath), crlfDelay: Infinity });
let headerRead = false;
const blocks = new Map();
for await (const line of profileLines) {
  if (!headerRead) {
    if (!/^mode: (set|count|atomic)$/.test(line)) {
      console.error(`coverage profile has an invalid mode header: ${line || "(empty)"}`);
      process.exit(2);
    }
    headerRead = true;
    continue;
  }
  if (!line) continue;
  const match = line.match(/^((.+):\d+\.\d+,\d+\.\d+) (\d+) (\d+)$/);
  if (!match) {
    console.error(`unrecognized coverage record: ${line}`);
    process.exit(2);
  }
  const [, location, filename, statementsText, countText] = match;
  if (path.posix.isAbsolute(filename) || filename.split("/").includes("..")) {
    console.error(`coverage record has an unsafe filename: ${line}`);
    process.exit(2);
  }
  const importPath = filePackages.get(filename);
  if (importPath === null) continue;
  if (importPath === undefined) {
    console.error(`coverage profile references an unknown source file: ${filename}`);
    process.exit(2);
  }
  const statements = Number(statementsText);
  const count = Number(countText);
  if (!Number.isSafeInteger(statements) || statements < 0 || !Number.isSafeInteger(count) || count < 0) {
    console.error(`invalid coverage counts: ${line}`);
    process.exit(2);
  }
  const previous = blocks.get(location);
  if (previous && previous.statements !== statements) {
    console.error(`coverage records disagree on statement count: ${location}`);
    process.exit(2);
  }
  // -coverpkg emits the same block from each test binary that imports it.
  // Count its statements once and retain hits from any of those binaries.
  blocks.set(location, { importPath, filename, statements, covered: count > 0 || previous?.covered === true });
}
if (!headerRead) {
  console.error("coverage profile has an invalid mode header: (empty)");
  process.exit(2);
}
const totals = new Map();
const fileTotals = new Map();
for (const { importPath, filename, statements, covered } of blocks.values()) {
  for (const [target, key] of [
    [totals, importPath],
    [fileTotals, filename],
  ]) {
    const value = target.get(key) ?? { statements: 0, covered: 0 };
    value.statements += statements;
    if (covered) value.covered += statements;
    target.set(key, value);
  }
}
if (totals.size === 0) {
  console.error("coverage profile contains no non-generated production statements");
  process.exit(2);
}
const missingPackages = [...expectedPackages].filter((name) => !totals.has(name));
if (missingPackages.length) {
  console.error(`coverage profile is missing owned test packages: ${missingPackages.join(", ")}`);
  process.exit(2);
}

const measured = Object.fromEntries(
  [...totals.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, value]) => [
      name,
      {
        statements: value.statements,
        percent: Number(((value.covered / value.statements) * 100).toFixed(2)),
      },
    ])
);
if (mode === "write") {
  // File floors protect established contracts when a package grows. Refreshing
  // aggregate measurements must not silently lower those independent floors.
  const fileFloors = fs.existsSync(baselinePath) ? JSON.parse(fs.readFileSync(baselinePath, "utf8")).files : undefined;
  fs.mkdirSync(path.dirname(baselinePath), { recursive: true });
  fs.writeFileSync(
    baselinePath,
    `${JSON.stringify({ version: 1, packages: measured, ...(fileFloors ? { files: fileFloors } : {}) }, null, 2)}\n`
  );
  console.log(`wrote coverage floors for ${Object.keys(measured).length} production packages to ${baselinePath}`);
  process.exit(0);
}

const baseline = JSON.parse(fs.readFileSync(baselinePath, "utf8"));
if (
  baseline.version !== 1 ||
  !baseline.packages ||
  typeof baseline.packages !== "object" ||
  Array.isArray(baseline.packages) ||
  (baseline.files !== undefined &&
    (!baseline.files || typeof baseline.files !== "object" || Array.isArray(baseline.files)))
) {
  console.error(`unsupported coverage baseline format: ${baselinePath}`);
  process.exit(2);
}
for (const [name, floor] of [...Object.entries(baseline.packages), ...Object.entries(baseline.files ?? {})]) {
  if (
    !name ||
    !floor ||
    !Number.isSafeInteger(floor.statements) ||
    floor.statements <= 0 ||
    !Number.isFinite(floor.percent) ||
    floor.percent < 0 ||
    floor.percent > 100
  ) {
    console.error(
      `invalid coverage floor for ${name}: expected positive statements and a finite percent from 0 to 100`
    );
    process.exit(2);
  }
}
const failures = [];
for (const [name, current] of Object.entries(measured)) {
  const floor = baseline.packages[name]?.percent;
  if (floor === undefined) failures.push(`${name}: no baseline; measured ${current.percent}%`);
  else if (current.percent < floor) failures.push(`${name}: ${current.percent}% is below ${floor}%`);
}
for (const name of Object.keys(baseline.packages)) {
  if (!(name in measured)) failures.push(`${name}: stale baseline entry; remove it after review`);
}
for (const [name, floor] of Object.entries(baseline.files ?? {})) {
  const current = fileTotals.get(name);
  if (!current) failures.push(`${name}: file floor is missing from coverage; remove it only after review`);
  else {
    const percent = Number(((current.covered / current.statements) * 100).toFixed(2));
    if (percent < floor.percent) failures.push(`${name}: ${percent}% is below file floor ${floor.percent}%`);
  }
}
if (failures.length) {
  console.error(failures.join("\n"));
  console.error(
    "Review the report, then explicitly refresh floors with scripts/quality/go.sh --write-coverage-baseline."
  );
  process.exit(1);
}
console.log(`coverage ratchet passed (${Object.keys(measured).length} production packages; generated files excluded)`);
