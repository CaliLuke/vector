import { observeNativeForms } from "./native-form-observer.mjs";
import { registerWebModuleHooks } from "./web-module-hooks.mjs";

export function setupWebNativeFormTest({ react, sourceFiles, globalName, hookExports }) {
  registerWebModuleHooks();
  globalThis[globalName] = react;
  return observeNativeForms(sourceFiles, hookExports);
}
