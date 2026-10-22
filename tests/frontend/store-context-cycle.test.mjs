import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";
import { readdirSync, readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const webRoot = path.join(repositoryRoot, "apps/web");
const storeRoot = path.join(webRoot, "core/store");
const storeContextPath = path.join(webRoot, "core/lib/store-context");
const requireFromWeb = createRequire(path.join(webRoot, "package.json"));
const typescript = requireFromWeb("typescript");

function sourceFiles(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const entryPath = path.join(directory, entry.name);
    if (entry.isDirectory()) return sourceFiles(entryPath);
    if (entry.isFile() && /\.[cm]?[jt]sx?$/.test(entry.name)) return [entryPath];
    return [];
  });
}

function resolveModulePath(sourcePath, specifier) {
  if (specifier.startsWith("@/")) return path.resolve(webRoot, "core", specifier.slice(2));
  if (specifier.startsWith(".")) return path.resolve(path.dirname(sourcePath), specifier);
  return undefined;
}

test("store modules do not create a runtime import cycle through the global store context", () => {
  const cycleEdges = [];

  for (const sourcePath of sourceFiles(storeRoot)) {
    const source = readFileSync(sourcePath, "utf8");
    const sourceFile = typescript.createSourceFile(sourcePath, source, typescript.ScriptTarget.Latest, true);

    for (const statement of sourceFile.statements) {
      const isRuntimeImport =
        (typescript.isImportDeclaration(statement) && !statement.importClause?.isTypeOnly) ||
        (typescript.isExportDeclaration(statement) && !statement.isTypeOnly);
      if (!isRuntimeImport || !statement.moduleSpecifier || !typescript.isStringLiteral(statement.moduleSpecifier))
        continue;

      const specifier = statement.moduleSpecifier.text;
      if (resolveModulePath(sourcePath, specifier) === storeContextPath) {
        cycleEdges.push(`${path.relative(repositoryRoot, sourcePath)} -> ${specifier}`);
      }
    }
  }

  assert.deepEqual(cycleEdges, []);
});
