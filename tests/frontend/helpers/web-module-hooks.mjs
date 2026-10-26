import { createRequire, registerHooks } from "node:module";
import { existsSync, readFileSync, statSync } from "node:fs";
import { transformSync } from "esbuild";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const repositoryRoot = fileURLToPath(new URL("../../..", import.meta.url));
const webRoot = path.join(repositoryRoot, "apps/web");
const adminRoot = path.join(repositoryRoot, "apps/admin");
const spaceRoot = path.join(repositoryRoot, "apps/space");
const uiRoot = path.join(repositoryRoot, "packages/ui");
const propelRoot = path.join(repositoryRoot, "packages/propel");
const i18nRoot = path.join(repositoryRoot, "packages/i18n");
const hooksRoot = path.join(repositoryRoot, "packages/hooks");
const requireFromWeb = createRequire(path.join(webRoot, "package.json"));
const appSourceRoots = [webRoot, adminRoot, spaceRoot, uiRoot, propelRoot, hooksRoot, i18nRoot].map(
  (root) => pathToFileURL(root).href
);
const relativeSpecifiers = new Set([".", ".."]);
const atlaskitPublicSubpaths = new Map(
  [
    "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter",
    "@atlaskit/pragmatic-drag-and-drop/combine",
    "@atlaskit/pragmatic-drag-and-drop/element/pointer-outside-of-preview",
    "@atlaskit/pragmatic-drag-and-drop/element/set-custom-native-drag-preview",
    "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge",
    "@atlaskit/pragmatic-drag-and-drop-hitbox/tree-item",
  ].map((specifier) => [specifier, requireFromWeb.resolve(specifier)])
);

function candidates(target) {
  return [`${target}.ts`, `${target}.tsx`, path.join(target, "index.ts"), path.join(target, "index.tsx"), target];
}

function isRelativeSpecifier(specifier) {
  return relativeSpecifiers.has(specifier) || specifier.startsWith("./") || specifier.startsWith("../");
}

function isAppSourceUrl(url) {
  return appSourceRoots.some((root) => url?.startsWith(root));
}

export function registerWebModuleHooks(options = {}) {
  registerAppModuleHooks(webRoot, "core", options);
}

export function registerAdminModuleHooks(options = {}) {
  registerAppModuleHooks(adminRoot, "", options);
}

export function registerSpaceModuleHooks(options = {}) {
  registerAppModuleHooks(spaceRoot, "", options);
}

function registerAppModuleHooks(appRoot, aliasRoot, { stubEmojiRoot = false } = {}) {
  registerHooks({
    resolve(specifier, context, nextResolve) {
      let target;
      if (
        stubEmojiRoot &&
        specifier === "../emoji-icon-picker/emoji/emoji" &&
        context.parentURL?.startsWith(pathToFileURL(propelRoot).href)
      ) {
        target = path.join(repositoryRoot, "tests/frontend/helpers/emoji-root-stub.mjs");
      } else if (appRoot === webRoot && /^@\/(app|helpers|styles)\//.test(specifier)) {
        target = path.resolve(appRoot, specifier.slice(2));
      } else if (specifier.startsWith("@/")) {
        target = path.resolve(appRoot, aliasRoot, specifier.slice(2));
      } else if (isRelativeSpecifier(specifier) && isAppSourceUrl(context.parentURL)) {
        target = fileURLToPath(new URL(specifier, context.parentURL));
      }

      target ??= atlaskitPublicSubpaths.get(specifier);
      if (target) {
        const file = candidates(target).find((candidate) => existsSync(candidate) && statSync(candidate).isFile());
        if (file) return nextResolve(pathToFileURL(file).href, context);
      }

      return nextResolve(specifier, context);
    },
    load(url, context, nextLoad) {
      if (
        (url.startsWith(pathToFileURL(adminRoot).href) ||
          url.startsWith(pathToFileURL(webRoot).href) ||
          url.startsWith(pathToFileURL(spaceRoot).href) ||
          url.startsWith(pathToFileURL(uiRoot).href) ||
          url.startsWith(pathToFileURL(propelRoot).href) ||
          url.startsWith(pathToFileURL(hooksRoot).href) ||
          url.startsWith(pathToFileURL(i18nRoot).href)) &&
        /\.tsx?$/.test(url)
      ) {
        const source = readFileSync(fileURLToPath(url), "utf8");
        const output = transformSync(source, {
          format: "esm",
          loader: url.endsWith(".tsx") ? "tsx" : "ts",
          sourcefile: fileURLToPath(url),
          target: "es2022",
          tsconfigRaw: { compilerOptions: { useDefineForClassFields: true } },
          ...(url.endsWith(".tsx") ? { jsx: "automatic", jsxImportSource: "react" } : {}),
        }).code;
        return { format: "module", shortCircuit: true, source: output };
      }

      return nextLoad(url, context);
    },
  });
}
