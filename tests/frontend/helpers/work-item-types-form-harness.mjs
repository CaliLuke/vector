import { registerHooks } from "node:module";
import { observeNativeForms } from "./native-form-observer.mjs";
import { registerWebModuleHooks } from "./web-module-hooks.mjs";
import { resolvePackageStub } from "./module-stubs.mjs";

/** Installs app aliases, narrow UI stubs, and observation of real TanStack FormApi instances. */
export function registerWorkItemTypesFormHarness({ sourceFiles, stubs, reactGlobal, react }) {
  registerWebModuleHooks();
  const nativeForms = observeNativeForms(sourceFiles);
  globalThis[reactGlobal] = react;
  registerHooks({
    resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
  });
  return nativeForms;
}
