#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";

const identity = (entry) => JSON.stringify([entry.file, entry.package, entry.function]);

export function parseComplexity(output, files) {
  const lines = output.trim().split(/\r?\n/);
  if (!/^Average: (?:\d+(?:\.\d+)?|NaN)$/.test(lines.pop() ?? "")) {
    throw new Error("gocyclo report is incomplete: missing average footer");
  }
  return lines.map((line) => {
    const match = /^(\d+) (\S+) (\S+) (.+):(\d+):(\d+)$/.exec(line);
    if (!match) throw new Error(`malformed gocyclo record: ${line}`);
    const [, score, packageName, functionName, filename, sourceLine] = match;
    const file = path.normalize(filename).replaceAll(path.sep, "/");
    const complexity = Number(score);
    if (!files.has(file) || !Number.isSafeInteger(complexity) || complexity < 1) {
      throw new Error(`gocyclo reported an unknown file or invalid score: ${line}`);
    }
    return { file, package: packageName, function: functionName, complexity, line: Number(sourceLine) };
  });
}

export function scanComplexity(moduleRoot, files, executable = "gocyclo") {
  if (!files.length || new Set(files).size !== files.length)
    throw new Error("owned-file inventory is empty or duplicated");
  for (const file of files) {
    if (path.isAbsolute(file) || file.split(/[\\/]/).includes("..") || !file.endsWith(".go")) {
      throw new Error(`invalid owned Go file: ${file}`);
    }
    if (!fs.statSync(path.join(moduleRoot, file)).isFile()) throw new Error(`missing owned Go file: ${file}`);
  }
  const entries = [];
  for (let offset = 0; offset < files.length; offset += 128) {
    const batch = files.slice(offset, offset + 128);
    // Unfiltered output exits zero only after successful analysis. The footer
    // detects truncated output; a missing tool or parser error propagates.
    const output = execFileSync(executable, ["-avg", ...batch], {
      cwd: moduleRoot,
      encoding: "utf8",
      timeout: 60_000,
      maxBuffer: 16 * 1024 * 1024,
    });
    entries.push(...parseComplexity(output, new Set(batch)));
  }
  const keys = entries.map(identity);
  if (!entries.length || new Set(keys).size !== keys.length)
    throw new Error("function inventory is empty or ambiguous");
  return entries.toSorted((left, right) => identity(left).localeCompare(identity(right)));
}
