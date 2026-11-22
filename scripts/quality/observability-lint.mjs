#!/usr/bin/env bun

import { spawnSync } from "node:child_process";
import { existsSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { parseArgs } from "node:util";

const root = fileURLToPath(new URL("../../", import.meta.url));
const frontendRoots = ["apps/web", "apps/admin", "apps/space", "packages"];
const backendRoot = "apps/go-api";
const frontendExcluded =
  /(?:^|\/)(?:__tests__|__fixtures__|tests?|testdata|fixtures|scripts|public|dist|build|gen|codemods|tailwind-config|typescript-config|logger)(?:\/|$)/;
const backendExcluded = /(?:^|\/)(?:gen|design|testdata)(?:\/|$)/;

function processOutput(program, args, input) {
  const result = spawnSync(program, args, {
    cwd: root,
    encoding: "utf8",
    input,
    timeout: 60_000,
    maxBuffer: 8 * 1024 * 1024,
  });
  if (result.error || result.status !== 0 || result.stderr.trim()) {
    // Emit only fixed tool names and statuses. A parser diagnostic may include source text.
    const reason =
      result.error?.code ?? result.signal ?? (result.status === 0 ? "unexpected stderr" : `exit ${result.status}`);
    throw new Error(`${path.basename(program)} could not complete ${args[0]} (${reason})`);
  }
  return result.stdout;
}

function matches(file, selector) {
  return selector === "" || file === selector || file.startsWith(`${selector}/`);
}

function selectedFiles(selectors) {
  const paths = selectors.map((selector) =>
    path.relative(root, path.resolve(root, selector)).split(path.sep).join("/")
  );
  if (paths.some((selector) => selector === ".." || selector.startsWith("../") || path.isAbsolute(selector))) {
    throw new Error("Pass file or directory paths inside the repository.");
  }
  const inventory = processOutput("git", [
    "ls-files",
    "--cached",
    "--others",
    "--exclude-standard",
    "-z",
    "--",
    ...frontendRoots,
    backendRoot,
  ]);
  const files = [...new Set(inventory.split("\0").filter(Boolean))].filter((file) => existsSync(path.join(root, file)));
  const available = {
    frontend: files.filter(
      (file) =>
        frontendRoots.some((dir) => file.startsWith(`${dir}/`)) &&
        !frontendExcluded.test(file) &&
        /\.[cm]?[jt]sx?$/.test(file) &&
        !/\.(?:d|gen|generated|test|spec|config|stories)\.[^.]+$/.test(file)
    ),
    backend: files.filter(
      (file) =>
        file.startsWith(`${backendRoot}/`) &&
        !backendExcluded.test(file) &&
        file.endsWith(".go") &&
        !file.endsWith("_test.go")
    ),
  };
  const all = [...available.frontend, ...available.backend];
  if (paths.some((selector) => !all.some((file) => matches(file, selector)))) {
    throw new Error("Each selected path must contain supported source files.");
  }
  const selected = (file) => paths.length === 0 || paths.some((selector) => matches(file, selector));
  return { frontend: available.frontend.filter(selected), backend: available.backend.filter(selected) };
}

export function lintFrontend(files) {
  const findings = [];
  for (let start = 0; start < files.length; start += 400) {
    const batch = files.slice(start, start + 400);
    const output = processOutput(process.execPath, [
      "--silent",
      "run",
      "oxlint",
      "--config",
      "scripts/quality/observability/oxlint.json",
      "--format",
      "json",
      "--threads",
      "2",
      "--no-ignore",
      "--",
      ...batch,
    ]);
    const report = JSON.parse(output);
    if (report.number_of_files !== batch.length || !Array.isArray(report.diagnostics)) {
      throw new Error("Oxlint returned an incomplete source inventory.");
    }
    for (const entry of report.diagnostics) {
      const rule = /^vector-observability\(([a-z-]+)\)$/.exec(entry.code)?.[1];
      const span = entry.labels?.[0]?.span;
      if (!rule || !span)
        throw new Error(`Oxlint reported an unexpected diagnostic in ${entry.filename ?? "(unknown file)"}.`);
      findings.push({ file: entry.filename, line: span.line, column: span.column, rule, message: entry.message });
    }
  }
  return findings;
}

function lintBackend(files) {
  if (files.length === 0) return [];
  return JSON.parse(processOutput("go", ["run", "scripts/quality/observability/scan-go.go"], JSON.stringify(files)));
}

function print(line) {
  process.stdout.write(line + "\n");
}

function main() {
  const { values, positionals } = parseArgs({
    options: { check: { type: "boolean" }, format: { type: "string", default: "text" }, help: { type: "boolean" } },
    allowPositionals: true,
  });
  if (values.help) {
    print("Usage: bun run check:observability [--check] [--format text|json] [file-or-directory ...]");
    return;
  }
  if (!["text", "json"].includes(values.format)) throw new Error("Use --format text or --format json.");
  const { frontend, backend } = selectedFiles(positionals);
  if (frontend.length === 0 && backend.length === 0)
    throw new Error("The selection contains no supported source files.");
  const findings = [...lintFrontend(frontend), ...lintBackend(backend)].toSorted(
    (a, b) => a.file.localeCompare(b.file) || a.line - b.line || a.column - b.column || a.rule.localeCompare(b.rule)
  );
  const counts = {};
  for (const { rule } of findings) counts[rule] = (counts[rule] ?? 0) + 1;
  const files = { frontend: frontend.length, backend: backend.length };
  if (values.format === "json") {
    print(JSON.stringify({ files, counts, findings }, null, 2));
  } else {
    for (const { file, line, column, rule, message } of findings)
      print(`${file}:${line}:${column} [${rule}] ${message}`);
    print(
      `Observability audit found ${findings.length} candidates across ${files.frontend} frontend and ${files.backend} Go input files.`
    );
    for (const [rule, count] of Object.entries(counts).toSorted(([left], [right]) => left.localeCompare(right)))
      print(`  ${count} ${rule}`);
  }
  if (values.check && findings.length > 0) process.exitCode = 1;
}

if (import.meta.main) {
  try {
    main();
  } catch (error) {
    console.error(`Observability audit failed: ${error.message}`);
    process.exitCode = 2;
  }
}
