import { createHash } from "node:crypto";
import { instrumentCoverageSource } from "./coverage-source.mjs";
import { parse } from "acorn";
import { transformSync } from "esbuild";
import { plugin } from "bun";
import { builtinModules, createRequire as nativeCreateRequire } from "node:module";
import { readFileSync, existsSync } from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

// One chain per isolated test process. New registrations wrap earlier ones so
// test-specific fixtures can delegate to the shared application resolver.
const hooks = [];
const fixtures = new Map();
export function registerFixture(source) {
  const digest = createHash("sha256").update(source).digest("hex");
  fixtures.set(digest, source);
  return `vector-fixture:${digest}`;
}
function fixtureURL(url) {
  return url.startsWith("data:") ? registerFixture(decodeURIComponent(url.slice(url.indexOf(",") + 1))) : url;
}

const builtins = new Set(builtinModules.flatMap((name) => [name, `node:${name}`]));
const packagesRoot = fileURLToPath(new URL("../../../packages/", import.meta.url));
function coverageSource(filename) {
  if (!process.env.VECTOR_FRONTEND_COVERAGE || !filename.startsWith(packagesRoot)) return filename;
  const source = filename.replace(/\/dist\/(.*)\.[cm]?js$/, "/src/$1");
  if (source === filename) return filename;
  return [source + ".ts", source + ".tsx"].find((candidate) => existsSync(candidate)) ?? filename;
}
export function createRequire(filename) {
  const original = nativeCreateRequire(filename);
  if (!process.env.VECTOR_FRONTEND_COVERAGE) return original;
  const require = (specifier) => original(coverageSource(original.resolve(specifier)));
  Object.assign(require, original);
  require.resolve = (specifier, options) => coverageSource(original.resolve(specifier, options));
  Object.assign(require.resolve, original.resolve);
  return require;
}
let resolving = false;
function resolve(specifier, context, index = hooks.length - 1) {
  if (index >= 0) {
    const next = (name, ctx = context) => resolve(name, ctx, index - 1);
    return hooks[index].resolve?.(specifier, context, next) ?? next(specifier, context);
  }
  if (
    specifier.startsWith("data:") ||
    specifier.startsWith("vector-fixture:") ||
    specifier.startsWith("node:") ||
    builtins.has(specifier)
  )
    return { url: specifier };
  resolving = true;
  let filename;
  try {
    filename = Bun.resolveSync(
      specifier.startsWith("file:") ? fileURLToPath(specifier) : specifier,
      context.parentURL?.startsWith("file:") ? path.dirname(fileURLToPath(context.parentURL)) : process.cwd()
    );
  } finally {
    resolving = false;
  }
  // Instrument workspace source before bundling can erase statement mappings.
  // The ordinary behavior suite still exercises the built package exports.
  return { url: pathToFileURL(coverageSource(filename)).href };
}
function load(url, context, index = hooks.length - 1) {
  if (index >= 0) {
    const next = (name, ctx = context) => load(name, ctx, index - 1);
    return hooks[index].load?.(url, context, next) ?? next(url, context);
  }
  return { source: readFileSync(fileURLToPath(url), "utf8") };
}
globalThis[Symbol.for("vector.resolveTestModule")] = (name, parentURL) =>
  fixtureURL(resolve(name, { parentURL, conditions: ["import", "default", "bun"] }).url);
function imports(code, url) {
  const edits = [];
  const visit = (node) => {
    if (!node || typeof node !== "object") return;
    if (["ImportDeclaration", "ExportNamedDeclaration", "ExportAllDeclaration"].includes(node.type) && node.source) {
      const name = node.source.value;
      if (!(name.startsWith("node:") || builtins.has(name))) {
        const target = resolve(name, { parentURL: url, conditions: ["import", "default", "bun"] }).url;
        edits.push({ start: node.source.start, end: node.source.end, value: JSON.stringify(fixtureURL(target)) });
      }
    }
    if (node.type === "ImportExpression") {
      edits.push({
        start: node.source.start,
        end: node.source.end,
        value: `globalThis[Symbol.for("vector.resolveTestModule")](${code.slice(node.source.start, node.source.end)},${JSON.stringify(url)})`,
      });
      return;
    }
    for (const child of Object.values(node)) {
      if (Array.isArray(child)) child.forEach(visit);
      else if (child && typeof child === "object") visit(child);
    }
  };
  visit(parse(code, { ecmaVersion: "latest", sourceType: "module" }));
  for (const edit of edits.toSorted((a, b) => b.start - a.start))
    code = code.slice(0, edit.start) + edit.value + code.slice(edit.end);
  return code;
}
const loadSource = ({ path: filename }) => {
  const result = load(pathToFileURL(filename).href, {});
  const source =
    result.format === "module" || !/\.(?:[cm]?ts|tsx)$/.test(filename)
      ? result.source
      : transformSync(result.source, { loader: filename.endsWith("tsx") ? "tsx" : "ts", format: "esm" }).code;
  return {
    contents: imports(
      process.env.VECTOR_FRONTEND_COVERAGE && /\/(apps|packages)\//.test(filename)
        ? instrumentCoverageSource(filename, source)
        : source,
      pathToFileURL(filename).href
    ),
    loader: "js",
  };
};

plugin({
  name: "vector-test-modules",
  setup(builder) {
    builder.onResolve({ filter: /.*/ }, ({ path: specifier, importer }) => {
      // Leave dependency resolution to Bun, including CommonJS require conditions.
      if (importer?.includes("/node_modules/")) return undefined;
      if (resolving || specifier.startsWith("node:") || builtins.has(specifier)) return undefined;
      if (specifier.startsWith("vector-fixture:")) return { path: specifier.slice(15), namespace: "vector-fixture" };
      const result = resolve(specifier.startsWith("/") ? pathToFileURL(specifier).href : specifier, {
        parentURL: importer ? pathToFileURL(importer).href : undefined,
        conditions: ["import", "default", "bun"],
      });
      const url = fixtureURL(result.url);
      if (url.startsWith("vector-fixture:")) return { path: url.slice(15), namespace: "vector-fixture" };
      if (url.startsWith("node:") || builtins.has(url)) return undefined;
      const filename = fileURLToPath(url);
      return { path: filename, ...(existsSync(filename) ? {} : { namespace: "vector-source" }) };
    });
    builder.onLoad({ filter: /.*/, namespace: "vector-fixture" }, ({ path: url }) => ({
      contents: imports(fixtures.get(url), "vector-fixture:" + url),
      loader: "js",
    }));
    builder.onLoad({ filter: /^(?!.*\/node_modules\/).*\.[cm]?[jt]sx?$/ }, loadSource);
    builder.onLoad({ filter: /.*/, namespace: "vector-source" }, loadSource);
  },
});

export function registerHooks(hook) {
  hooks.push(hook);
  return {
    deregister() {
      const index = hooks.indexOf(hook);
      if (index >= 0) hooks.splice(index, 1);
    },
  };
}
