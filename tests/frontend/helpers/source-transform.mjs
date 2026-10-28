import { createHash, randomUUID } from "node:crypto";
import { mkdirSync, readFileSync, renameSync, rmSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { transformSync, version } from "esbuild";

const defaultDirectory = fileURLToPath(new URL("../../../node_modules/.cache/frontend-transforms/", import.meta.url));

// Workers have independent module registries, but identical compiler inputs can
// reuse output. Atomic publication prevents another worker reading partial code.
export function transformFrontendSource(source, options, directory = defaultDirectory) {
  const key = createHash("sha256").update(JSON.stringify({ source, options, version })).digest("hex");
  const target = path.join(directory, `${key}.mjs`);
  try {
    return readFileSync(target, "utf8");
  } catch (error) {
    if (error.code !== "ENOENT") throw error;
  }
  const output = transformSync(source, options).code;
  mkdirSync(directory, { recursive: true });
  const temporary = `${target}.${process.pid}.${randomUUID()}.tmp`;
  try {
    writeFileSync(temporary, output, { flag: "wx" });
    renameSync(temporary, target);
  } finally {
    rmSync(temporary, { force: true });
  }
  return output;
}
