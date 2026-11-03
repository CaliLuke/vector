#!/usr/bin/env bun

import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, isAbsolute, matchesGlob, relative, resolve, sep } from "node:path";
import { execFileSync } from "node:child_process";

const METRICS = ["lines", "statements", "functions", "branches"];

function fail(message) {
  console.error(`Coverage ratchet: ${message}`);
  process.exit(1);
}

function parseArgs(argv) {
  const options = {
    baseline: ".quality/coverage-baseline.json",
    summary: "coverage/coverage-summary.json",
    root: process.cwd(),
    tolerance: 0.01,
    writeBaseline: false,
    scopeFile: undefined,
    scopeKey: undefined,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === "--write-baseline") {
      options.writeBaseline = true;
    } else if (argument === "--help") {
      process.stdout.write(
        "Usage: check-coverage-ratchet.mjs [--summary FILE] [--baseline FILE] [--root DIR] [--scope-file FILE --scope-key DIR] --new-file-min PERCENT [--tolerance PERCENT] [--write-baseline]\n"
      );
      process.exit(0);
    } else if (
      ["--summary", "--baseline", "--root", "--new-file-min", "--tolerance", "--scope-file", "--scope-key"].includes(
        argument
      )
    ) {
      const value = argv[index + 1];
      if (value === undefined) fail(`${argument} requires a value`);
      index += 1;
      const key = {
        "--summary": "summary",
        "--baseline": "baseline",
        "--root": "root",
        "--new-file-min": "newFileMin",
        "--tolerance": "tolerance",
        "--scope-file": "scopeFile",
        "--scope-key": "scopeKey",
      }[argument];
      options[key] = ["--tolerance", "--new-file-min"].includes(argument) ? Number(value) : value;
    } else {
      fail(`unknown argument: ${argument}`);
    }
  }

  options.root = resolve(options.root);
  options.summary = resolve(options.root, options.summary);
  options.baseline = resolve(options.root, options.baseline);
  if (options.scopeFile) options.scopeFile = resolve(process.cwd(), options.scopeFile);
  options.newFileMin = Number(options.newFileMin);

  if (!Number.isFinite(options.newFileMin) || options.newFileMin < 0 || options.newFileMin > 100) {
    fail("--new-file-min must be a number from 0 through 100");
  }
  if (!Number.isFinite(options.tolerance) || options.tolerance < 0) {
    fail("--tolerance must be a non-negative number");
  }
  return options;
}

function validateScope(files, options) {
  if (!options.scopeFile && !options.scopeKey) return;
  if (!options.scopeFile || !options.scopeKey) fail("--scope-file and --scope-key must be supplied together");
  const scope = readJson(options.scopeFile, "coverage scope");
  const suite = scope.coveredSuites?.[options.scopeKey];
  if (!suite || !Array.isArray(suite.sourcePatterns) || !Array.isArray(suite.exclusions)) {
    fail(`coverage scope has no valid suite definition for ${options.scopeKey}`);
  }
  const repositoryRoot = resolve(options.root, "..", "..");
  const tracked = execFileSync(
    "git",
    ["ls-files", "--cached", "--others", "--exclude-standard", "--", `${options.scopeKey}/`],
    {
      cwd: repositoryRoot,
      encoding: "utf8",
    }
  )
    .trim()
    .split(/\r?\n/)
    .filter(Boolean);
  const expected = new Set(
    tracked
      .filter((file) => {
        if (!existsSync(resolve(repositoryRoot, file))) return false;
        const local = relative(repositoryRoot, resolve(repositoryRoot, file))
          .split(sep)
          .join("/")
          .slice(`${options.scopeKey}/`.length);
        return (
          suite.sourcePatterns.some((pattern) => matchesGlob(local, pattern)) &&
          !suite.exclusions.some((pattern) => matchesGlob(local, pattern))
        );
      })
      .map((file) => file.slice(`${options.scopeKey}/`.length))
  );
  const missing = [...expected].filter((file) => !files[file]);
  const unexpected = Object.keys(files).filter((file) => !expected.has(file));
  if (missing.length || unexpected.length) {
    fail(
      `coverage source set does not match ${options.scopeKey}: ${missing.length} missing, ${unexpected.length} unexpected\n  - ${[...missing.map((file) => `missing ${file}`), ...unexpected.map((file) => `unexpected ${file}`)].join("\n  - ")}`
    );
  }
}

function readJson(path, label) {
  if (!existsSync(path)) fail(`${label} is missing: ${path}`);
  try {
    return JSON.parse(readFileSync(path, "utf8"));
  } catch (error) {
    fail(`${label} is invalid JSON: ${error.message}`);
  }

  return undefined;
}

function normalizeFile(file, root) {
  const absolute = isAbsolute(file) ? file : resolve(root, file);
  const normalized = relative(root, absolute).split(sep).join("/");
  if (normalized === ".." || normalized.startsWith("../")) {
    fail(`coverage file is outside --root: ${file}`);
  }
  return normalized;
}

function uncoveredCount(coverage) {
  const { total, covered } = coverage ?? {};
  return Number.isInteger(total) && Number.isInteger(covered) && total >= covered && covered >= 0
    ? total - covered
    : undefined;
}

function extractSummary(summary, root) {
  const files = {};
  for (const [file, coverage] of Object.entries(summary)) {
    if (file === "total") continue;
    const normalized = normalizeFile(file, root);
    files[normalized] = {};
    for (const metric of METRICS) {
      const percent = Number(coverage?.[metric]?.pct);
      if (!Number.isFinite(percent) || percent < 0 || percent > 100) {
        fail(`invalid ${metric} percentage for ${normalized}`);
      }
      files[normalized][metric] = { percent, uncovered: uncoveredCount(coverage?.[metric]) };
    }
  }
  if (Object.keys(files).length === 0) fail("coverage summary contains no source files");
  return Object.fromEntries(Object.entries(files).toSorted(([left], [right]) => left.localeCompare(right)));
}

function writeBaseline(path, files) {
  const measured = Object.fromEntries(
    Object.entries(files).map(([file, metrics]) => [
      file,
      Object.fromEntries(
        METRICS.map((metric) => {
          const { percent, uncovered } = metrics[metric];
          if (uncovered === undefined) fail(`missing or invalid ${metric} counts for ${file}`);
          return [metric, [percent, uncovered]];
        })
      ),
    ])
  );
  mkdirSync(dirname(path), { recursive: true });
  writeFileSync(path, `${JSON.stringify({ version: 2, files: measured }, null, 2)}\n`);
  process.stdout.write(`Coverage baseline written: ${path}\n`);
}

function compareUncovered(file, metric, uncovered, ceiling) {
  if (!Number.isInteger(ceiling) || ceiling < 0 || uncovered === undefined)
    return [`${file}: missing or invalid ${metric} counts`];
  if (uncovered > ceiling) return [`${file}: uncovered ${metric} increased from ${ceiling} to ${uncovered}`];
  return [];
}

function compareExistingFile(file, current, expected, tolerance, version) {
  const failures = [];
  for (const metric of METRICS) {
    const entry = expected?.[metric];
    if (version === 2 && (!Array.isArray(entry) || entry.length !== 2)) {
      failures.push(`${file}: invalid baseline ${metric}`);
      continue;
    }
    const floor = Number(version === 2 ? entry[0] : entry);
    if (!Number.isFinite(floor) || floor < 0 || floor > 100) failures.push(`${file}: invalid baseline ${metric}`);
    else if (current[metric].percent + tolerance < floor) {
      failures.push(`${file}: ${metric} fell from ${floor}% to ${current[metric].percent}%`);
    }
    if (version === 2) failures.push(...compareUncovered(file, metric, current[metric].uncovered, entry[1]));
  }
  return failures;
}

function compare(files, baseline, options) {
  if (![1, 2].includes(baseline.version) || !baseline.files || typeof baseline.files !== "object") {
    fail("baseline must contain version 1 or 2 and a files object");
  }

  const failures = [];
  for (const [file, expected] of Object.entries(baseline.files)) {
    const current = files[file];
    if (!current) {
      if (existsSync(resolve(options.root, file))) failures.push(`${file}: missing from coverage summary`);
      continue;
    }
    failures.push(...compareExistingFile(file, current, expected, options.tolerance, baseline.version));
  }

  for (const [file, current] of Object.entries(files)) {
    if (baseline.files[file]) continue;
    for (const metric of METRICS) {
      if (current[metric].percent + options.tolerance < options.newFileMin) {
        failures.push(`${file}: new-file ${metric} is ${current[metric].percent}%, below ${options.newFileMin}%`);
      }
    }
  }

  if (failures.length > 0) fail(`failed:\n  - ${failures.join("\n  - ")}`);
  process.stdout.write(`Coverage ratchet passed for ${Object.keys(files).length} files.\n`);
}

const options = parseArgs(process.argv.slice(2));
const files = extractSummary(readJson(options.summary, "coverage summary"), options.root);
validateScope(files, options);

if (options.writeBaseline) writeBaseline(options.baseline, files);
else compare(files, readJson(options.baseline, "coverage baseline"), options);
