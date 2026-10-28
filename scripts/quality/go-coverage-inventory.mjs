#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

// Use the same owned-source boundary for instrumentation and the coverage gate.
// Integration tests may exercise a repository from a different Go package.
export function coverageInventory(moduleRoot) {
  const packages = execFileSync("go", ["list", "-json", "-tags=integration", "./..."], {
    cwd: moduleRoot,
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
  })
    .split(/\n(?=\{)/)
    .filter(Boolean)
    .map((chunk) => JSON.parse(chunk));
  const filePackages = new Map();
  const expectedPackages = new Set();
  const instrumentedPackages = new Set();
  for (const pkg of packages) {
    let hasOwnedProductionFiles = false;
    for (const filename of [...(pkg.GoFiles ?? []), ...(pkg.CgoFiles ?? [])]) {
      const fullPath = path.join(pkg.Dir, filename);
      const key = `${pkg.ImportPath}/${filename}`;
      if (fullPath.includes(`${path.sep}gen${path.sep}`)) {
        filePackages.set(key, null);
        continue;
      }
      const header = fs.readFileSync(fullPath, "utf8").split(/\r?\n/).slice(0, 10).join("\n");
      if (/^\/\/ Code generated .* DO NOT EDIT\.$/m.test(header)) {
        filePackages.set(key, null);
        continue;
      }
      hasOwnedProductionFiles = true;
      filePackages.set(key, pkg.ImportPath);
    }
    if (hasOwnedProductionFiles) instrumentedPackages.add(pkg.ImportPath);
    if (hasOwnedProductionFiles && (pkg.TestGoFiles?.length ?? 0) + (pkg.XTestGoFiles?.length ?? 0) > 0) {
      expectedPackages.add(pkg.ImportPath);
    }
  }
  return { filePackages, expectedPackages, instrumentedPackages };
}

if (process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1])) {
  const moduleRoot = process.argv[2];
  if (!moduleRoot) throw new Error("usage: go-coverage-inventory.mjs <module-root>");
  const packages = [...coverageInventory(moduleRoot).instrumentedPackages].toSorted((left, right) =>
    left.localeCompare(right)
  );
  if (!packages.length) throw new Error("no owned Go packages found for coverage");
  process.stdout.write(`${packages.join(",")}\n`);
}
