#!/usr/bin/env node
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { compareCountMaps, countTotal, readCountBaseline, writeCountBaseline } from "./count-baseline.mjs";

const [mode, reportPath, baselinePath, moduleRoot] = process.argv.slice(2);
if (!["check", "write"].includes(mode) || !reportPath || !baselinePath || !moduleRoot) {
  console.error("usage: go-dupl-ratchet.mjs <check|write> <dupl-report> <baseline.json> <module-root>");
  process.exit(2);
}

const counts = new Map();
const records = fs.readFileSync(reportPath, "utf8").split(/\r?\n/).filter(Boolean);
if (records.length === 0 && fs.statSync(reportPath).size > 0 && fs.readFileSync(reportPath, "utf8").trim() !== "") {
  console.error(`dupl report contains no usable records: ${reportPath}`);
  process.exit(2);
}
for (const line of records) {
  const match = line.match(/^(.+):(\d+)-(\d+): duplicate of (.+):(\d+)-(\d+)$/);
  if (!match) {
    console.error(`unrecognized dupl record: ${line}`);
    process.exit(2);
  }
  const [, fileA, startA, endA, fileB, startB, endB] = match;
  const body = (filename, start, end) =>
    fs
      .readFileSync(path.join(moduleRoot, filename), "utf8")
      .split(/\r?\n/)
      .slice(Number(start) - 1, Number(end))
      .join("\n")
      .replace(/\s+/g, " ")
      .trim();
  const bodyA = body(fileA, startA, endA);
  const bodyB = body(fileB, startB, endB);
  if (!bodyA || !bodyB) {
    console.error(`dupl record points to an empty or missing source range: ${line}`);
    process.exit(2);
  }
  const files = [fileA, fileB].sort();
  const hashes = [bodyA, bodyB].map((body) => crypto.createHash("sha256").update(body).digest("hex")).sort();
  const key = JSON.stringify([...files, ...hashes]);
  counts.set(key, (counts.get(key) ?? 0) + 1);
}
if (mode === "write") {
  writeCountBaseline(
    baselinePath,
    counts,
    (length) => `wrote ${length} duplication fingerprints (${records.length} directed reports) to ${baselinePath}`
  );
  process.exit(0);
}
const allowed = readCountBaseline(baselinePath, 4, "duplication");
const failures = compareCountMaps(counts, allowed);
if (failures.length) {
  console.error(failures.join("\n"));
  console.error(
    "Review duplication changes, then refresh the explicit baseline with scripts/quality/go.sh --write-static-baselines."
  );
  process.exit(1);
}
console.log(`dupl ratchet passed (${countTotal(counts)} existing clone pairs)`);
