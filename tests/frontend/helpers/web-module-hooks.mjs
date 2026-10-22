import { createRequire, registerHooks } from "node:module";
import { existsSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const repositoryRoot = fileURLToPath(new URL("../../..", import.meta.url));
const webRoot = path.join(repositoryRoot, "apps/web");
const spaceRoot = path.join(repositoryRoot, "apps/space");
const uiRoot = path.join(repositoryRoot, "packages/ui");
const propelRoot = path.join(repositoryRoot, "packages/propel");
const i18nRoot = path.join(repositoryRoot, "packages/i18n");
const hooksRoot = path.join(repositoryRoot, "packages/hooks");
const requireFromWeb = createRequire(path.join(webRoot, "package.json"));
const typescript = requireFromWeb("typescript");

function candidates(target) {
  return [`${target}.ts`, `${target}.tsx`, path.join(target, "index.ts"), path.join(target, "index.tsx"), target];
}

export function registerWebModuleHooks(options = {}) {
  registerAppModuleHooks(webRoot, "core", options);
}

export function registerSpaceModuleHooks(options = {}) {
  registerAppModuleHooks(spaceRoot, "", options);
}

function registerAppModuleHooks(appRoot, aliasRoot, { stubStoreContext = false, stubEmojiRoot = false } = {}) {
  registerHooks({
    resolve(specifier, context, nextResolve) {
      let target;
      if (stubStoreContext && specifier === "@/lib/store-context") {
        target = path.join(repositoryRoot, "tests/frontend/helpers/work-item-store-context.mjs");
      } else if (
        stubEmojiRoot &&
        specifier === "../emoji-icon-picker/emoji/emoji" &&
        context.parentURL?.startsWith(pathToFileURL(propelRoot).href)
      ) {
        target = path.join(repositoryRoot, "tests/frontend/helpers/emoji-root-stub.mjs");
      } else if (specifier.startsWith("@/")) {
        target = path.resolve(appRoot, aliasRoot, specifier.slice(2));
      } else if (
        (specifier === "." || specifier === ".." || specifier.startsWith("./") || specifier.startsWith("../")) &&
        (context.parentURL?.startsWith(pathToFileURL(webRoot).href) ||
          context.parentURL?.startsWith(pathToFileURL(spaceRoot).href) ||
          context.parentURL?.startsWith(pathToFileURL(uiRoot).href) ||
          context.parentURL?.startsWith(pathToFileURL(propelRoot).href) ||
          context.parentURL?.startsWith(pathToFileURL(hooksRoot).href) ||
          context.parentURL?.startsWith(pathToFileURL(i18nRoot).href))
      ) {
        target = fileURLToPath(new URL(specifier, context.parentURL));
      }

      if (target) {
        const file = candidates(target).find((candidate) => existsSync(candidate) && statSync(candidate).isFile());
        if (file) return nextResolve(pathToFileURL(file).href, context);
      }

      return nextResolve(specifier, context);
    },
    load(url, context, nextLoad) {
      if (
        (url.startsWith(pathToFileURL(webRoot).href) ||
          url.startsWith(pathToFileURL(spaceRoot).href) ||
          url.startsWith(pathToFileURL(uiRoot).href) ||
          url.startsWith(pathToFileURL(propelRoot).href) ||
          url.startsWith(pathToFileURL(hooksRoot).href) ||
          url.startsWith(pathToFileURL(i18nRoot).href)) &&
        /\.tsx?$/.test(url)
      ) {
        const compilerOptions = {
          module: typescript.ModuleKind.ESNext,
          target: typescript.ScriptTarget.ES2022,
        };
        if (url.endsWith(".tsx")) compilerOptions.jsx = typescript.JsxEmit.ReactJSX;
        const source = readFileSync(fileURLToPath(url), "utf8");
        const output = typescript.transpileModule(source, { compilerOptions }).outputText;
        return { format: "module", shortCircuit: true, source: output };
      }

      return nextLoad(url, context);
    },
  });
}
