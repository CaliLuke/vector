import { readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { createOxcInstrumenter as createInstrumenter } from "oxc-coverage-instrument/vitest";
import { transformSync } from "esbuild";
import { createFileCoverage } from "@vitest/istanbul-lib-coverage";
import { sourceFiles } from "./frontend-coverage.mjs";

// Istanbul omits modules with no instrumentable statements. Keep those files in
// the ownership inventory; missing executable files must still fail the gate.
const workspace = process.argv[2];
const root = process.cwd();
const scope = JSON.parse(readFileSync(".quality/coverage-scope.json", "utf8"));
const suite = scope.coveredSuites[workspace];
if (!suite) throw new Error(`Unknown coverage workspace: ${workspace}`);
const filename = path.resolve(workspace, "coverage/coverage-summary.json");
const summary = JSON.parse(readFileSync(filename, "utf8"));
for (const file of sourceFiles(root, { [workspace]: suite })) {
  if (summary[file]) continue;
  const code = transformSync(readFileSync(file, "utf8"), {
    loader: file.endsWith(".tsx") ? "tsx" : "ts",
    format: "esm",
  }).code;
  const instrumenter = createInstrumenter({ compat: "istanbul", stripTypescript: false });
  instrumenter.instrumentSync(code, file);
  const coverage = createFileCoverage(instrumenter.lastFileCoverage());
  if (
    coverage.toSummary().statements.total ||
    coverage.toSummary().functions.total ||
    coverage.toSummary().branches.total
  )
    throw new Error(`Coverage omitted executable source: ${file}`);
  summary[file] = coverage.toSummary().toJSON();
}
writeFileSync(filename, JSON.stringify(summary, null, 2) + "\n");
