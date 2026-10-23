import fs from "node:fs";
import path from "node:path";

export function writeCountBaseline(baselinePath, counts, describe) {
  const entries = [...counts.entries()]
    .toSorted(([a], [b]) => a.localeCompare(b))
    .map(([key, count]) => ({ fingerprint: JSON.parse(key), count }));
  fs.mkdirSync(path.dirname(baselinePath), { recursive: true });
  fs.writeFileSync(baselinePath, `${JSON.stringify({ version: 1, entries }, null, 2)}\n`);
  process.stdout.write(`${describe(entries.length)}\n`);
}

export function readCountBaseline(baselinePath, fingerprintLength, label) {
  const baseline = JSON.parse(fs.readFileSync(baselinePath, "utf8"));
  if (baseline.version !== 1 || !Array.isArray(baseline.entries)) {
    console.error(`unsupported ${label} baseline format: ${baselinePath}`);
    process.exit(2);
  }
  const allowed = new Map();
  for (const { fingerprint, count } of baseline.entries) {
    if (
      !Array.isArray(fingerprint) ||
      fingerprint.length !== fingerprintLength ||
      fingerprint.some((part) => typeof part !== "string") ||
      !Number.isSafeInteger(count) ||
      count <= 0
    ) {
      console.error(`invalid ${label} baseline entry: ${JSON.stringify({ fingerprint, count })}`);
      process.exit(2);
    }
    const key = JSON.stringify(fingerprint);
    if (allowed.has(key)) {
      console.error(`duplicate ${label} baseline entry: ${key}`);
      process.exit(2);
    }
    allowed.set(key, count);
  }
  return allowed;
}

export function compareCountMaps(counts, allowed) {
  const failures = [];
  for (const [key, count] of counts) {
    const previous = allowed.get(key) ?? 0;
    if (count > previous) failures.push(`NEW x${count - previous} ${key}`);
  }
  for (const [key, count] of allowed) {
    const current = counts.get(key) ?? 0;
    if (count > current) failures.push(`BASELINE CAN SHRINK x${count - current} ${key}`);
  }
  return failures;
}

export function countTotal(counts) {
  return [...counts.values()].reduce((sum, count) => sum + count, 0);
}
