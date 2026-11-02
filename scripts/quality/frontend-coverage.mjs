import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, readdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { parseArgs } from "node:util";
import { parse } from "acorn";
import { convert } from "ast-v8-to-istanbul";
import { mergeScriptCovs } from "@bcoe/v8-coverage";
import { createCoverageMap, createFileCoverage } from "@vitest/istanbul-lib-coverage";
import { transformSync } from "esbuild";
import { sourceMapFor } from "../../tests/frontend/helpers/coverage-source.mjs";

function readJSON(filename) {
  return JSON.parse(readFileSync(filename, "utf8"));
}
export function sourceFiles(root, suites) {
  const files = execFileSync(
    "git",
    [
      "ls-files",
      "--cached",
      "--others",
      "--exclude-standard",
      "-z",
      "--",
      ...Object.keys(suites).map((workspace) => `${workspace}/`),
    ],
    {
      cwd: root,
      encoding: "utf8",
      maxBuffer: 4 * 1024 * 1024,
    }
  )
    .split("\0")
    .filter(Boolean);
  return files
    .filter(
      (filename) =>
        existsSync(path.resolve(root, filename)) &&
        Object.entries(suites).some(([workspace, suite]) => {
          if (!filename.startsWith(`${workspace}/`)) return false;
          const local = filename.slice(workspace.length + 1);
          return (
            suite.sourcePatterns.some((pattern) => path.matchesGlob(local, pattern)) &&
            !suite.exclusions.some((pattern) => path.matchesGlob(local, pattern))
          );
        })
    )
    .map((filename) => path.resolve(root, filename));
}

function readSources(directory) {
  const byProcess = new Map();
  const versions = new Map();
  for (const filename of readdirSync(path.join(directory, "sources"))) {
    const pid = /^(\d+)\.jsonl$/.exec(filename)?.[1];
    if (!pid) throw new Error(`Invalid captured source manifest: ${filename}`);
    for (const digest of readFileSync(path.join(directory, "sources", filename), "utf8")
      .trim()
      .split("\n")) {
      if (!/^[a-f0-9]{64}$/.test(digest)) throw new Error(`Invalid captured source reference: ${filename}`);
      if (!versions.has(digest)) {
        const payload = path.join(directory, "variants", `${digest}.json`);
        if (!existsSync(payload)) throw new Error(`Missing captured source variant: ${digest}`);
        const source = readJSON(payload);
        if (typeof source.url !== "string" || typeof source.code !== "string")
          throw new Error(`Invalid captured source: ${digest}`);
        const actual = createHash("sha256")
          .update(source.url)
          .update(source.code)
          .update(JSON.stringify(source.map ?? null))
          .digest("hex");
        if (actual !== digest) throw new Error(`Captured source digest mismatch: ${digest}`);
        versions.set(digest, { ...source, digest });
      }
      const source = versions.get(digest);
      const key = `${pid}:${source.url}`;
      const previous = byProcess.get(key);
      if (previous && previous.digest !== digest)
        throw new Error(`Multiple transforms for ${source.url} in process ${pid}`);
      byProcess.set(key, source);
    }
  }
  return byProcess;
}

function ownedURL(url, root, suites) {
  if (!url.startsWith("file:")) return false;
  const local = path.relative(root, fileURLToPath(url));
  return (
    !local.includes(`${path.sep}node_modules${path.sep}`) &&
    Object.keys(suites).some((workspace) => local.startsWith(`${workspace}/`))
  );
}

function linkOnlyModule(code) {
  return parse(code, { ecmaVersion: "latest", sourceType: "module" }).body.every(
    (node) =>
      node.type === "ImportDeclaration" ||
      node.type === "ExportAllDeclaration" ||
      (node.type === "ExportNamedDeclaration" && node.declaration === null)
  );
}

function capturedScripts(directory, root, suites) {
  const sources = readSources(directory);
  const grouped = new Map();
  let processes = 0;
  for (const filename of readdirSync(path.join(directory, "raw"))) {
    const pid = /^coverage-(\d+)-.*\.json$/.exec(filename)?.[1];
    if (!pid) continue;
    const raw = readJSON(path.join(directory, "raw", filename));
    if (!Array.isArray(raw.result)) throw new Error(`Invalid V8 report: ${filename}`);
    processes++;
    for (const coverage of raw.result) {
      if (!ownedURL(coverage.url, root, suites)) continue;
      const source = sources.get(`${pid}:${coverage.url}`);
      if (!source) throw new Error(`No executed source captured for ${coverage.url} in process ${pid}`);
      if (coverage.url.includes("/dist/") && !source.map) {
        // Bundlers omit maps for generated entry points that only re-export
        // mapped chunks. They contain no owned executable statements.
        if (linkOnlyModule(source.code)) continue;
        throw new Error(`Missing build source map for ${coverage.url}`);
      }
      const digest = source.digest;
      const group = grouped.get(digest) ?? { source, coverage: [] };
      group.coverage.push(coverage);
      grouped.set(digest, group);
    }
  }
  if (!processes || !grouped.size) throw new Error("No owned V8 coverage was collected");
  return Array.from(grouped.values());
}

function validateMappedSources(source, files) {
  if (!source.map) {
    const filename = fileURLToPath(source.url);
    if (files.has(filename) && source.code !== readFileSync(filename, "utf8"))
      throw new Error(`Captured source is stale: ${filename}`);
    return;
  }
  for (const [index, filename] of source.map.sources.entries()) {
    if (!files.has(filename)) continue;
    if (source.map.sourcesContent?.[index] !== readFileSync(filename, "utf8"))
      throw new Error(`Build or transform source is stale: ${filename}`);
  }
}

async function remap(source, functions) {
  const ast = parse(source.code, { ecmaVersion: "latest", sourceType: "module", locations: true });
  return convert({
    code: source.code,
    sourceMap: source.map,
    ast,
    coverage: { url: source.url, functions },
    wrapperLength: 0,
  });
}

// Two conversions at a time keep the coverage pass within the existing guard.
async function visitBounded(items, visit) {
  let index = 0;
  const next = async () => {
    if (index >= items.length) return;
    await visit(items[index++]);
    return next();
  };
  await Promise.all([next(), next()]);
}

export async function collectFrontendCoverage({ root, directory, suites, files = sourceFiles(root, suites) }) {
  const owned = new Set(files);
  const map = createCoverageMap({});
  const mergeOwned = (converted) => {
    for (const [filename, coverage] of Object.entries(converted)) {
      const normalized = filename.startsWith("file:") ? fileURLToPath(filename) : filename;
      if (owned.has(normalized)) map.addFileCoverage({ ...coverage, path: normalized });
    }
  };
  await visitBounded(capturedScripts(directory, root, suites), async ({ source, coverage }) => {
    validateMappedSources(source, owned);
    const merged = mergeScriptCovs(coverage);
    if (!merged) throw new Error(`Cannot merge coverage for ${source.url}`);
    mergeOwned(await remap(source, merged.functions));
  });
  const measured = new Set(map.files());
  await visitBounded(
    files.filter((filename) => !measured.has(filename)),
    async (filename) => {
      const code = transformSync(readFileSync(filename, "utf8"), {
        loader: filename.endsWith(".tsx") ? "tsx" : filename.endsWith(".ts") ? "ts" : "js",
        format: "esm",
        target: "es2022",
        jsx: "automatic",
        sourcefile: filename,
        sourcemap: "inline",
        sourcesContent: true,
      }).code;
      mergeOwned(await remap({ code, url: pathToFileURL(filename).href, map: sourceMapFor(code, filename) }, []));
      // Type-only files have no executable statements, but still belong to scope.
      if (!map.files().includes(filename)) map.addFileCoverage(createFileCoverage(filename));
    }
  );
  const summary = { total: map.getCoverageSummary().toJSON() };
  for (const filename of map.files().toSorted()) summary[filename] = map.fileCoverageFor(filename).toSummary().toJSON();
  mkdirSync(directory, { recursive: true });
  writeFileSync(path.join(directory, "coverage-final.json"), JSON.stringify(map));
  writeFileSync(path.join(directory, "coverage-summary.json"), JSON.stringify(summary, null, 2) + "\n");
  const workspaceDirectory = path.join(directory, "workspaces");
  mkdirSync(workspaceDirectory, { recursive: true });
  for (const workspace of Object.keys(suites)) {
    const prefix = path.resolve(root, workspace) + path.sep;
    const rows = map.files().filter((filename) => filename.startsWith(prefix));
    const subset = createCoverageMap(
      Object.fromEntries(rows.map((filename) => [filename, map.fileCoverageFor(filename).toJSON()]))
    );
    const workspaceSummary = {
      total: subset.getCoverageSummary().toJSON(),
      ...Object.fromEntries(rows.map((filename) => [filename, summary[filename]])),
    };
    writeFileSync(
      path.join(workspaceDirectory, `${workspace.replaceAll("/", "-")}.json`),
      JSON.stringify(workspaceSummary, null, 2) + "\n"
    );
  }
  return summary;
}

if (process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url) {
  const { values } = parseArgs({
    options: { directory: { type: "string" }, root: { type: "string", default: process.cwd() } },
  });
  if (!values.directory) throw new Error("--directory is required");
  const scope = readJSON(path.join(values.root, ".quality/coverage-scope.json"));
  const suites = Object.fromEntries(
    Object.entries(scope.coveredSuites).filter(([, suite]) => suite.testCommand === "pnpm test:frontend:coverage")
  );
  const summary = await collectFrontendCoverage({
    root: values.root,
    directory: path.resolve(values.directory),
    suites,
  });
  process.stdout.write(
    `Frontend source coverage: ${Object.keys(summary).length - 1} files, ${summary.total.lines.pct}% lines\n`
  );
}
