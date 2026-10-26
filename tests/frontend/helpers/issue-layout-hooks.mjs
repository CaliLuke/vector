import { registerHooks } from "node:module";
import { registerWebModuleHooks } from "./web-module-hooks.mjs";

export function registerIssueLayoutHooks() {
  registerWebModuleHooks();
  registerHooks({
    resolve(specifier, context, nextResolve) {
      if (specifier === "@/lib/store-context" && context.parentURL?.endsWith("/issues/issue-layouts/utils.tsx")) {
        return { shortCircuit: true, url: "data:text/javascript,export const store = {};" };
      }
      return nextResolve(specifier, context);
    },
  });
}
