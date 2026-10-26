import { registerHooks } from "node:module";
import { pathToFileURL } from "node:url";
import { webRequire } from "./react.mjs";
import { resolvePackageStub } from "./module-stubs.mjs";
import { registerWebModuleHooks } from "./web-module-hooks.mjs";

export const reactModuleUrl = pathToFileURL(webRequire.resolve("react")).href;

/** Install the existing source loader and only the module fixtures supplied by this test. */
export function registerWebTestModules(stubs, resolve) {
  registerWebModuleHooks();
  registerHooks({
    resolve(specifier, context, nextResolve) {
      const resolveStub = (name, parent) => resolvePackageStub(name, stubs, parent, nextResolve);
      return resolve ? resolve(specifier, context, resolveStub) : resolveStub(specifier, context);
    },
  });
}
