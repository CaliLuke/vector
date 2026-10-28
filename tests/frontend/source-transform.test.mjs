import assert from "node:assert/strict";
import { execFile } from "node:child_process";
import { mkdtempSync, readdirSync, rmSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { promisify } from "node:util";
import test from "node:test";
import { transformFrontendSource } from "./helpers/source-transform.mjs";

const run = promisify(execFile);
const options = { loader: "ts", format: "esm", target: "es2022" };

function cacheFixture(t) {
  const directory = mkdtempSync(path.join(os.tmpdir(), "vector-transforms-"));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  return directory;
}

void test("cached transforms preserve source and compiler option changes", (t) => {
  const directory = cacheFixture(t);
  const source = "export const value: number = 1;";
  const first = transformFrontendSource(source, options, directory);
  assert.equal(transformFrontendSource(source, options, directory), first);
  assert.equal(readdirSync(directory).length, 1);
  const changed = transformFrontendSource("export const value: number = 2;", options, directory);
  assert.match(changed, /value = 2/);
  const minified = transformFrontendSource(source, { ...options, minify: true }, directory);
  assert.notEqual(minified, first);
  assert.equal(readdirSync(directory).length, 3);
  assert.throws(() => transformFrontendSource("export const = ;", options, directory));
  assert.equal(readdirSync(directory).length, 3, "failed compilation must not publish a cache entry");
});

void test("independent workers publish complete interchangeable transforms", async (t) => {
  const directory = cacheFixture(t);
  const source = "export const value: number = 42;";
  const script = `import { transformFrontendSource } from ${JSON.stringify(new URL("./helpers/source-transform.mjs", import.meta.url).href)};
    process.stdout.write(transformFrontendSource(${JSON.stringify(source)}, ${JSON.stringify(options)}, ${JSON.stringify(directory)}));`;
  const results = await Promise.all([
    run(process.execPath, ["--input-type=module", "-e", script]),
    run(process.execPath, ["--input-type=module", "-e", script]),
  ]);
  assert.equal(results[0].stdout, results[1].stdout);
  assert.match(results[0].stdout, /value = 42/);
  assert.equal(readdirSync(directory).length, 1);
  assert.equal(transformFrontendSource(source, options, directory), results[0].stdout);
});
