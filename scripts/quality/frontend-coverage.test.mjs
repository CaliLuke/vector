import assert from "node:assert/strict";
import { execFile } from "node:child_process";
import { realpathSync, mkdtempSync, mkdirSync, readFileSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { pathToFileURL } from "node:url";
import { promisify } from "node:util";
import test from "node:test";
import { transformSync } from "esbuild";
import { collectFrontendCoverage } from "./frontend-coverage.mjs";

const run = promisify(execFile);
const sourceHook = new URL("../../tests/frontend/helpers/source-transform.mjs", import.meta.url).href;
const recordHook = new URL("../../tests/frontend/helpers/coverage-source.mjs", import.meta.url).href;
const suites = { "apps/web": { sourcePatterns: ["**/*.tsx", "**/*.ts"], exclusions: [] } };
const source = `type Choice = { enabled: boolean };
const element = (tag: string, _props: unknown, text: string) => ({tag, text});
export function render(choice: Choice) {
  if (choice.enabled) return <span>on</span>;
  return <span>off</span>;
}
`;
function fixture(t) {
  const root = realpathSync(mkdtempSync(path.join(os.tmpdir(), "vector-source-coverage-")));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const app = path.join(root, "apps/web");
  mkdirSync(app, { recursive: true });
  const filename = path.join(app, "component.tsx");
  const untouched = path.join(app, "untouched.ts");
  const types = path.join(app, "types.ts");
  writeFileSync(filename, source);
  writeFileSync(untouched, "export function untested() { return 'not executed'; }\n");
  writeFileSync(types, "export type OnlyTypes = { value: string };\n");
  const directory = path.join(root, "coverage");
  return { root, directory, filename, untouched, types, files: [filename, untouched, types], suites };
}
async function execute(f, enabled = true, { built = false, emitMap = true, facade = false } = {}) {
  const target = built ? path.join(f.root, "apps/web/dist/component.mjs") : f.filename;
  if (built) {
    mkdirSync(path.dirname(target), { recursive: true });
    const output = transformSync(source, {
      loader: "tsx",
      format: "esm",
      jsxFactory: "element",
      sourcefile: f.filename,
      sourcemap: "external",
    });
    writeFileSync(target, `${output.code}${emitMap ? "\n//# sourceMappingURL=component.mjs.map\n" : ""}`);
    writeFileSync(`${target}.map`, output.map);
  }
  const entry = facade ? path.join(path.dirname(target), "facade.mjs") : target;
  if (facade) writeFileSync(entry, 'export {render} from "./component.mjs";');
  const program = `import {readFileSync} from 'node:fs';
import {registerHooks} from 'node:module';
import {fileURLToPath} from 'node:url';
import {transformFrontendSource} from ${JSON.stringify(sourceHook)};
import {recordCoverageSource} from ${JSON.stringify(recordHook)};
registerHooks({load(url,context,next) {
  if(!${JSON.stringify([pathToFileURL(target).href, pathToFileURL(entry).href])}.includes(url))return next(url,context);
  const filename=fileURLToPath(url), source=readFileSync(filename,'utf8');
  const code=${built} ? source : transformFrontendSource(source,{loader:'tsx',format:'esm',jsxFactory:'element',sourcefile:filename});
  recordCoverageSource(filename,code);
  return {format:'module',shortCircuit:true,source:code};
}});
const {render}=await import(${JSON.stringify(pathToFileURL(entry).href)});
if(render({enabled:${enabled}}).text!==${JSON.stringify(enabled ? "on" : "off")})throw new Error('wrong branch');`;
  // This fixture exercises the existing V8 coverage and loader-hook contract.
  await run("node", ["--input-type=module", "-e", program], {
    env: {
      ...process.env,
      NODE_DISABLE_COMPILE_CACHE: "1",
      NODE_V8_COVERAGE: path.join(f.directory, "raw"),
      VECTOR_FRONTEND_COVERAGE: f.directory,
    },
  });
}
function lineHits(coverage, line) {
  return Object.entries(coverage.statementMap)
    .filter(([, location]) => location.start.line === line)
    .map(([id]) => coverage.s[id]);
}
void test("source coverage counts the skipped TSX return as uncovered, includes unloaded files, and omits type-only statements", async (t) => {
  const f = fixture(t);
  await execute(f);
  const summary = await collectFrontendCoverage(f);
  const coverage = JSON.parse(readFileSync(path.join(f.directory, "coverage-final.json"), "utf8"));
  assert.ok(lineHits(coverage[f.filename], 4).some((count) => count > 0));
  assert.deepEqual(lineHits(coverage[f.filename], 5), [0]);
  assert.deepEqual(lineHits(coverage[f.filename], 1), []);
  assert.equal(summary[f.untouched].functions.pct, 0);
  assert.equal(summary[f.untouched].lines.pct, 0);
  assert.equal(summary[f.types].statements.total, 0);
  assert.equal(summary[f.types].lines.pct, 100);
});
void test("source coverage combines worker runs and separately mapped build output", async (t) => {
  const f = fixture(t);
  await Promise.all([execute(f, true), execute(f, false), execute(f, false, { built: true, facade: true })]);
  const summary = await collectFrontendCoverage(f);
  assert.equal(readdirSync(path.join(f.directory, "variants")).length, 3);
  assert.equal(readdirSync(path.join(f.directory, "sources")).length, 3);
  assert.equal(summary[f.filename].branches.pct, 100);
  assert.equal(summary[f.filename].lines.pct, 100);
});
void test("source coverage rejects missing executed source and stale build maps", async (t) => {
  const f = fixture(t);
  await execute(f, true, { built: true });
  writeFileSync(f.filename, `${source}\nexport const changed = true;`);
  await assert.rejects(collectFrontendCoverage(f), /source is stale/);
  writeFileSync(f.filename, source);
  const directory = path.join(f.directory, "sources");
  for (const filename of readdirSync(directory)) rmSync(path.join(directory, filename));
  await assert.rejects(collectFrontendCoverage(f), /No executed source captured/);
});

void test("source coverage rejects executable build output without a map", async (t) => {
  const f = fixture(t);
  await execute(f, true, { built: true, emitMap: false });
  await assert.rejects(collectFrontendCoverage(f), /Missing build source map/);
});

void test("source coverage rejects corrupted shared source payloads", async (t) => {
  const f = fixture(t);
  await execute(f);
  const directory = path.join(f.directory, "variants");
  const filename = path.join(directory, readdirSync(directory)[0]);
  const captured = JSON.parse(readFileSync(filename, "utf8"));
  captured.code += "\n// modified after execution";
  writeFileSync(filename, JSON.stringify(captured));
  await assert.rejects(collectFrontendCoverage(f), /source digest mismatch/);
});
