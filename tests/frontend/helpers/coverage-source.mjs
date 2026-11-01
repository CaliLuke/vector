import { createHash } from "node:crypto";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
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
export function recordCoverageSource(filename, code) {
  const directory = process.env.VECTOR_FRONTEND_COVERAGE;
  if (!directory) return;
  const url = pathToFileURL(filename).href;
  const digest = createHash("sha256").update(url).update(code).digest("hex");
  const key = `${process.pid}-${digest}`;
  if (recorded.has(`${directory}/${key}`)) return;
  const target = path.join(directory, "sources");
  mkdirSync(target, { recursive: true });
  writeFileSync(
    path.join(target, `${key}.json`),
    JSON.stringify({ pid: process.pid, url, code, map: sourceMapFor(code, filename) })
  );
  recorded.add(`${directory}/${key}`);
}
