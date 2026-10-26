import { registerHooks } from "node:module";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";
import { createModuleStub } from "./module-stubs.mjs";

const observers = new Map();
let nextObserverId = 0;
const sourceUrl = (file) => pathToFileURL(resolve(file)).href;

// Called by the instrumented module; the actual TanStack hook and FormApi are unchanged.
export function recordNativeForm(observerId, source, form) {
  observers.get(observerId)?.get(source)?.add(form);
}

/**
 * Observe real FormApi instances created by the specified repository source files.
 * Register before importing components. With sequential top-level awaited tests,
 * restore after the last test, rather than in a root after hook.
 */
export function observeNativeForms(sourceFiles, hookExports = { "@tanstack/react-form": ["useForm"] }) {
  const observerId = nextObserverId++;
  const forms = new Map(sourceFiles.map((file) => [sourceUrl(file), new Set()]));
  observers.set(observerId, forms);
  const hook = registerHooks({
    resolve(specifier, context, nextResolve) {
      const source = context.parentURL;
      const exportsToObserve = hookExports[specifier];
      if (!exportsToObserve || !forms.has(source)) {
        return nextResolve(specifier, context);
      }
      const native = nextResolve(specifier, context);
      return createModuleStub(`
        export * from ${JSON.stringify(native.url)};
        import * as actualHooks from ${JSON.stringify(native.url)};
        import { recordNativeForm } from ${JSON.stringify(import.meta.url)};
        ${exportsToObserve
          .map(
            (name) => `export function ${name}(...args) {
          const form = actualHooks.${name}(...args);
          recordNativeForm(${observerId}, ${JSON.stringify(source)}, form);
          return form;
        }`
          )
          .join("\n")}
      `);
    },
  });

  return {
    getForms(file) {
      return [...(forms.get(sourceUrl(file)) ?? [])];
    },
    clear() {
      for (const instances of forms.values()) instances.clear();
    },
    restore() {
      hook.deregister();
      observers.delete(observerId);
    },
  };
}
