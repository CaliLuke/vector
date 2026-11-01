import assert from "node:assert/strict";
import { execFile } from "node:child_process";
import { createRequire } from "node:module";
import { fileURLToPath, pathToFileURL } from "node:url";
import { promisify } from "node:util";
import test from "node:test";

const run = promisify(execFile);
const loggerURL = new URL("../../packages/logger/dist/index.mjs", import.meta.url).href;
const requireLive = createRequire(new URL("../../apps/live/package.json", import.meta.url));
const coverageOptions = process.env.VECTOR_FRONTEND_COVERAGE
  ? ["--import", fileURLToPath(new URL("./helpers/coverage-preload.mjs", import.meta.url))]
  : [];

void test("shared logging emits structured levels and timestamps and honors LOG_LEVEL", async () => {
  const program = `import {logger} from ${JSON.stringify(loggerURL)};
logger.info('hidden informational message');
logger.error('visible error', {operation:'fixture'});`;
  const result = await run(process.execPath, [...coverageOptions, "--input-type=module", "-e", program], {
    env: { ...process.env, LOG_LEVEL: "error" },
  });
  const rows = result.stdout
    .trim()
    .split("\n")
    .map((line) => JSON.parse(line));
  assert.equal(rows.length, 1);
  assert.equal(rows[0].level, "error");
  assert.equal(rows[0].message, "visible error");
  assert.equal(rows[0].operation, "fixture");
  assert.match(rows[0].timestamp, /^\d{4}-\d{2}-\d{2} /);
});
void test("HTTP logging records the completed request's method, URL and response status", async () => {
  const program = `import express from ${JSON.stringify(pathToFileURL(requireLive.resolve("express")).href)};
import {loggerMiddleware} from ${JSON.stringify(loggerURL)};
const app=express(); app.use(loggerMiddleware); app.get('/fixture', (_req,res)=>res.status(201).send('created'));
const server=app.listen(0,'127.0.0.1'); await new Promise(resolve=>server.once('listening',resolve));
try { const response=await fetch('http://127.0.0.1:'+server.address().port+'/fixture'); await response.text(); }
finally { await new Promise((resolve,reject)=>server.close(error=>error?reject(error):resolve())); }`;
  const result = await run(process.execPath, [...coverageOptions, "--input-type=module", "-e", program], {
    env: { ...process.env, LOG_LEVEL: "" },
  });
  const rows = result.stdout
    .trim()
    .split("\n")
    .map((line) => JSON.parse(line));
  assert.equal(rows.length, 1);
  assert.equal(rows[0].level, "info");
  assert.match(rows[0].message, /^GET \/fixture 201 /);
  assert.equal(rows[0].meta.req.method, "GET");
  assert.equal(rows[0].meta.res.statusCode, 201);
});
void test("shared PostCSS configuration compiles Tailwind utilities without a browser", async () => {
  const configURL = new URL("../../packages/tailwind-config/postcss.config.js", import.meta.url);
  const requireConfig = createRequire(configURL);
  const requirePlugin = createRequire(requireConfig.resolve("@tailwindcss/postcss"));
  const postcss = requirePlugin("postcss");
  const { default: config } = await import(configURL.href);
  const result = await postcss(config.plugins).process(
    '@import "tailwindcss" source(none); @source inline("flex text-center");',
    { from: fileURLToPath(new URL("fixture.css", configURL)) }
  );
  assert.match(result.css, /\.flex\s*\{\s*display: flex;/);
  assert.match(result.css, /\.text-center\s*\{\s*text-align: center;/);
  assert.deepEqual(result.warnings(), []);
});
