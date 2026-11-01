/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// Usage:
//   tsx packages/i18n/scripts/sync-check.ts          # Report only
//   tsx packages/i18n/scripts/sync-check.ts --ci     # Exit 1 if issues found

import type { LocaleData } from "./lib/locale-io.js";
import { LOCALES_DIR, listLocales, loadLocale } from "./lib/locale-io.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Format a number with commas (e.g. 7712 -> "7,712"). */
function fmt(n: number): string {
  return n.toLocaleString("en-US");
}

// ---------------------------------------------------------------------------
// Checks
// ---------------------------------------------------------------------------

interface CollisionEntry {
  key: string;
  files: string[];
}

/** Cross-namespace collision check: same flattened key in multiple namespace files. */
function findCollisions(localeData: LocaleData): CollisionEntry[] {
  const keyToFiles = new Map<string, string[]>();
  for (const ns of localeData.namespaces) {
    for (const key of ns.keys) {
      const existing = keyToFiles.get(key);
      if (existing) {
        existing.push(`${ns.name}.json`);
      } else {
        keyToFiles.set(key, [`${ns.name}.json`]);
      }
    }
  }

  const collisions: CollisionEntry[] = [];
  for (const [key, files] of keyToFiles) {
    if (files.length > 1) {
      collisions.push({ key, files });
    }
  }
  return collisions.toSorted((a, b) => a.key.localeCompare(b.key));
}

interface PathConflict {
  leaf: string;
  branch: string;
}

/** Path conflict check: a key is both a leaf AND a prefix of another key. */
function findPathConflicts(localeData: LocaleData): PathConflict[] {
  const allKeysArray = [...localeData.allKeys].toSorted();
  const conflicts: PathConflict[] = [];

  // Build a set of all prefixes used in the keys
  const prefixes = new Set<string>();
  for (const key of allKeysArray) {
    const parts = key.split(".");
    for (let i = 1; i < parts.length; i++) {
      prefixes.add(parts.slice(0, i).join("."));
    }
  }

  // A conflict exists when a leaf key is also a prefix
  for (const key of allKeysArray) {
    if (prefixes.has(key)) {
      // Find one example of a key that extends this prefix
      const extending = allKeysArray.find((k) => k.startsWith(key + "."));
      if (extending) {
        conflicts.push({ leaf: key, branch: extending });
      }
    }
  }

  return conflicts;
}

interface LocaleComparison {
  locale: string;
  totalKeys: number;
  missingKeys: string[];
  staleKeys: string[];
  coverage: number; // 0-100
}

function compareToEnglish(enKeys: Set<string>, other: LocaleData): LocaleComparison {
  const missingKeys: string[] = [];
  const staleKeys: string[] = [];

  for (const key of enKeys) {
    if (!other.allKeys.has(key)) {
      missingKeys.push(key);
    }
  }

  for (const key of other.allKeys) {
    if (!enKeys.has(key)) {
      staleKeys.push(key);
    }
  }

  const coverage = enKeys.size > 0 ? ((enKeys.size - missingKeys.length) / enKeys.size) * 100 : 100;

  return {
    locale: other.locale,
    totalKeys: other.allKeys.size,
    missingKeys: missingKeys.toSorted(),
    staleKeys: staleKeys.toSorted(),
    coverage,
  };
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

function main() {
  const ciMode = process.argv.includes("--ci");

  // Discover all locale directories
  const localeDirs = listLocales();

  if (!localeDirs.includes("en")) {
    console.error("ERROR: English locale (en) not found in", LOCALES_DIR);
    process.exit(1);
  }

  // Load all locales
  const localeDataMap = new Map<string, LocaleData>();
  for (const locale of localeDirs) {
    localeDataMap.set(locale, loadLocale(locale));
  }

  const enData = localeDataMap.get("en")!;

  // Run checks
  const collisions = findCollisions(enData);
  const pathConflicts = findPathConflicts(enData);

  const comparisons: LocaleComparison[] = [];
  for (const locale of localeDirs) {
    if (locale === "en") continue;
    comparisons.push(compareToEnglish(enData.allKeys, localeDataMap.get(locale)!));
  }

  // -------------------------------------------------------------------------
  // Print report
  // -------------------------------------------------------------------------

  let hasFailure = false;

  process.stdout.write("\n=== Sync Check Results ===\n\n");
  process.stdout.write(`  en:    ${fmt(enData.allKeys.size)} keys (source)\n\n`);

  printLocaleSummary(comparisons);
  hasFailure = comparisons.some((comparison) => comparison.missingKeys.length > 0);

  // Cross-namespace collisions
  if (collisions.length > 0) {
    hasFailure = true;
    process.stdout.write("\nCROSS-NAMESPACE COLLISIONS:\n");
    for (const c of collisions) {
      process.stdout.write(`  ✗ "${c.key}" exists in: ${c.files.join(", ")}\n`);
    }
  }

  // Path conflicts
  if (pathConflicts.length > 0) {
    hasFailure = true;
    process.stdout.write("\nPATH CONFLICTS:\n");
    for (const pc of pathConflicts) {
      process.stdout.write(`  ✗ "${pc.leaf}" is a leaf but "${pc.branch}" extends it\n`);
    }
  }

  printKeyDetails(comparisons, "missingKeys", "missing", "\n--- Missing Keys Detail ---\n\n");
  printKeyDetails(comparisons, "staleKeys", "stale", "--- Stale Keys Detail ---\n\n");

  // CI exit code
  if (ciMode && hasFailure) {
    process.stdout.write("CI mode: exiting with code 1 due to missing keys, collisions, or path conflicts.\n");
    process.exit(1);
  }

  if (!hasFailure) {
    process.stdout.write("\nAll locales are in sync with English. No issues found.\n");
  }
}

function printLocaleSummary(comparisons: LocaleComparison[]) {
  for (const comp of comparisons) {
    const status = comp.missingKeys.length === 0 ? "✓" : "✗";
    const missingStr = comp.missingKeys.length > 0 ? ` — ${fmt(comp.missingKeys.length)} missing` : "";
    const staleStr = comp.staleKeys.length > 0 ? `, ${fmt(comp.staleKeys.length)} stale` : "";
    process.stdout.write(
      `  ${status} ${comp.locale.padEnd(10)} ${fmt(comp.totalKeys)} keys (${comp.coverage.toFixed(1)}%)${missingStr}${staleStr}\n`
    );
  }
}

function printKeyDetails(
  comparisons: LocaleComparison[],
  field: "missingKeys" | "staleKeys",
  label: string,
  heading: string
) {
  const affected = comparisons.filter((comparison) => comparison[field].length > 0);
  if (!affected.length) return;
  process.stdout.write(heading);
  for (const comparison of affected) {
    const keys = comparison[field];
    process.stdout.write(`${comparison.locale} (${fmt(keys.length)} ${label}):\n`);
    for (const key of keys.slice(0, 20)) process.stdout.write(`  - ${key}\n`);
    if (keys.length > 20) process.stdout.write(`  ... and ${fmt(keys.length - 20)} more\n`);
    process.stdout.write("\n");
  }
}

try {
  main();
} catch (err) {
  console.error("Sync check failed:", err);
  process.exit(1);
}
