import { createOxcInstrumenter } from "oxc-coverage-instrument/vitest";
import { createHash } from "node:crypto";
import { appendFileSync, existsSync, mkdirSync, readFileSync, renameSync, rmSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

export function sourceMapFor(code, filename) {
  const reference = /\/\/[#@] sourceMappingURL=(\S+)/g;
  const value = Array.from(code.matchAll(reference)).at(-1)?.[1];
  if (!value) return undefined;
  const inline = value.startsWith("data:");
  const mapPath = inline ? filename : path.resolve(path.dirname(filename), value);
  const map = JSON.parse(
    inline ? Buffer.from(value.slice(value.indexOf(",") + 1), "base64").toString() : readFileSync(mapPath, "utf8")
  );
  return {
    ...map,
    sourceRoot: "",
    sources: map.sources.map((source) =>
      source.startsWith("file:")
        ? fileURLToPath(source)
        : path.resolve(path.dirname(mapPath), map.sourceRoot ?? "", source)
    ),
  };
}

const recorded = new Set();
function instrument(code, filename, map) {
  const instrumenter = createOxcInstrumenter({ compat: "istanbul", stripTypescript: false });
  return instrumenter.instrumentSync(code, filename, map);
}
let flushing = false;
function flushAtExit(directory) {
  if (flushing) return;
  flushing = true;
  process.on("exit", () => {
    mkdirSync(path.join(directory, "raw"), { recursive: true });
    writeFileSync(
      path.join(directory, "raw", `coverage-${process.pid}.json`),
      JSON.stringify(globalThis["__coverage__"] ?? {})
    );
  });
}
export function instrumentCoverageSource(filename, code) {
  const directory = process.env.VECTOR_FRONTEND_COVERAGE;
  if (!directory) return code;
  recordCoverageSource(filename, code);
  flushAtExit(directory);
  return instrument(code, filename, sourceMapFor(code, filename));
}
export function recordCoverageSource(filename, code) {
  const directory = process.env.VECTOR_FRONTEND_COVERAGE;
  if (!directory) return;
  const url = pathToFileURL(filename).href;
  const source = { url, code, map: sourceMapFor(code, filename) };
  const digest = createHash("sha256")
    .update(url)
    .update(code)
    .update(JSON.stringify(source.map ?? null))
    .digest("hex");
  const key = `${directory}/${digest}`;
  if (recorded.has(key)) return;
  const target = path.join(directory, "sources");
  const variants = path.join(directory, "variants");
  mkdirSync(target, { recursive: true });
  mkdirSync(variants, { recursive: true });
  const payload = path.join(variants, `${digest}.json`);
  if (!existsSync(payload)) {
    const temporary = `${payload}.${process.pid}.tmp`;
    try {
      writeFileSync(temporary, JSON.stringify(source), { flag: "wx" });
      renameSync(temporary, payload);
    } finally {
      rmSync(temporary, { force: true });
    }
  }
  appendFileSync(path.join(target, `${process.pid}.jsonl`), `${digest}\n`);
  recorded.add(key);
}
