#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";

const [mode, profilePath, baselinePath, moduleRoot] = process.argv.slice(2);
if (!["check", "write"].includes(mode) || !profilePath || !baselinePath || !moduleRoot) {
  console.error("usage: go-coverage-ratchet.mjs <check|write> <coverprofile> <baseline.json> <module-root>");
  process.exit(2);
}

const packages = execFileSync("go", ["list", "-json", "-tags=integration", "./..."], {
  cwd: moduleRoot,
  encoding: "utf8",
  maxBuffer: 64 * 1024 * 1024,
})
  .split(/\n(?=\{)/)
  .filter(Boolean)
  .map((chunk) => JSON.parse(chunk));
const filePackages = new Map();
const expectedPackages = new Set();
for (const pkg of packages) {
  let hasOwnedProductionFiles = false;
  for (const filename of [...(pkg.GoFiles ?? []), ...(pkg.CgoFiles ?? [])]) {
    const fullPath = path.join(pkg.Dir, filename);
    const key = `${pkg.ImportPath}/${filename}`;
    if (fullPath.includes(`${path.sep}gen${path.sep}`)) {
      filePackages.set(key, null);
      continue;
    }
    const header = fs.readFileSync(fullPath, "utf8").split(/\r?\n/).slice(0, 10).join("\n");
    if (/^\/\/ Code generated .* DO NOT EDIT\.$/m.test(header)) {
      filePackages.set(key, null);
      continue;
    }
    hasOwnedProductionFiles = true;
    filePackages.set(key, pkg.ImportPath);
  }
  if (hasOwnedProductionFiles && (pkg.TestGoFiles?.length ?? 0) + (pkg.XTestGoFiles?.length ?? 0) > 0) {
    expectedPackages.add(pkg.ImportPath);
  }
}
const profileLines = fs.readFileSync(profilePath, "utf8").split(/\r?\n/);
if (!/^mode: (set|count|atomic)$/.test(profileLines[0] ?? "")) {
  console.error(`coverage profile has an invalid mode header: ${profileLines[0] ?? "(empty)"}`);
  process.exit(2);
}
const totals = new Map();
for (const line of profileLines.slice(1)) {
  if (!line) continue;
  const match = line.match(/^(.+):\d+\.\d+,\d+\.\d+ (\d+) (\d+)$/);
  if (!match) {
    console.error(`unrecognized coverage record: ${line}`);
    process.exit(2);
  }
  const [, filename, statementsText, countText] = match;
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
  const value = totals.get(importPath) ?? { statements: 0, covered: 0 };
  const statements = Number(statementsText);
  const count = Number(countText);
  if (!Number.isSafeInteger(statements) || statements < 0 || !Number.isSafeInteger(count) || count < 0) {
    console.error(`invalid coverage counts: ${line}`);
    process.exit(2);
  }
  value.statements += statements;
  if (count > 0) value.covered += statements;
  totals.set(importPath, value);
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
  fs.mkdirSync(path.dirname(baselinePath), { recursive: true });
  fs.writeFileSync(baselinePath, `${JSON.stringify({ version: 1, packages: measured }, null, 2)}\n`);
  console.log(`wrote coverage floors for ${Object.keys(measured).length} production packages to ${baselinePath}`);
  process.exit(0);
}

const baseline = JSON.parse(fs.readFileSync(baselinePath, "utf8"));
if (
  baseline.version !== 1 ||
  !baseline.packages ||
  typeof baseline.packages !== "object" ||
  Array.isArray(baseline.packages)
) {
  console.error(`unsupported coverage baseline format: ${baselinePath}`);
  process.exit(2);
}
for (const [name, floor] of Object.entries(baseline.packages)) {
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
if (failures.length) {
  console.error(failures.join("\n"));
  console.error(
    "Review the report, then explicitly refresh floors with scripts/quality/go.sh --write-coverage-baseline."
  );
  process.exit(1);
}
console.log(`coverage ratchet passed (${Object.keys(measured).length} production packages; generated files excluded)`);
