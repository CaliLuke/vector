import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import { existsSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { React, act, createReactDomTestEnvironment, webRequire } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const adminRoot = path.join(repositoryRoot, "apps/admin");
const reactURL = pathToFileURL(webRequire.resolve("react")).href;
const mobxURL = pathToFileURL(webRequire.resolve("mobx")).href;
const typescript = webRequire("typescript");
const inputURL = pathToFileURL(path.join(repositoryRoot, "packages/ui/src/form-fields/input.tsx")).href;
const serviceStub = `
  export class AuthService { requestCSRFToken() { return Promise.resolve({ csrf_token: "test-csrf-token" }); } }
  export class InstanceWorkspaceService { list() { return Promise.resolve({ results: [], next_cursor: null, next_page_results: false }); } }
`;
const buttonStub = `import React from "react"; export const Button = (props) => React.createElement("button", props, props.children);`;
const uiStub = `import React from "react"; export { Input } from ${JSON.stringify(inputURL)}; export const Spinner = () => React.createElement("span");`;
const substitutes = {
  "next/navigation": "export const useSearchParams = () => new URLSearchParams();",
  "lucide-react": "export const Eye = () => null; export const EyeOff = () => null;",
  "@plane/constants": 'export const API_BASE_URL = "http://localhost";',
  "@plane/propel/button": buttonStub,
  "@plane/services": serviceStub,
  "@plane/ui": uiStub,
  "@/components/common/banner": "export const Banner = () => null;",
  "@/components/instance/form-header": "export const FormHeader = () => null;",
  "./auth-banner": "export const AuthBanner = () => null;",
  "./auth-header": "export const AuthHeader = () => null;",
  "./auth-helpers": "export const authErrorHandler = () => null;",
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "react") return { shortCircuit: true, url: reactURL };
    if (specifier === "mobx") return { shortCircuit: true, url: mobxURL };
    if (Object.hasOwn(substitutes, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(substitutes[specifier])}` };
    }
    if (context.parentURL?.startsWith(pathToFileURL(adminRoot).href) && specifier.startsWith("@/")) {
      const base = path.resolve(adminRoot, specifier.slice(2));
      const candidates = [`${base}.ts`, `${base}.tsx`, path.join(base, "index.ts"), path.join(base, "index.tsx"), base];
      const target = candidates.find((candidate) => existsSync(candidate) && statSync(candidate).isFile());
      if (target) return { shortCircuit: true, url: pathToFileURL(target).href };
    }
    if (specifier === "../utils" && context.parentURL?.endsWith("/form-fields/input.tsx")) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');")}`,
      };
    }
    return nextResolve(specifier, context);
  },
  load(url, context, nextLoad) {
    if (url.startsWith(pathToFileURL(adminRoot).href) && /\.tsx?$/.test(url)) {
      const compilerOptions = { module: typescript.ModuleKind.ESNext, target: typescript.ScriptTarget.ES2022 };
      if (url.endsWith(".tsx")) compilerOptions.jsx = typescript.JsxEmit.ReactJSX;
      const sourceOverride =
        (url.endsWith("/app/(all)/(home)/sign-in-form.tsx") && process.env.ADMIN_SIGNIN_SOURCE_OVERRIDE) ||
        (url.endsWith("/store/workspace.store.ts") && process.env.ADMIN_WORKSPACE_STORE_SOURCE_OVERRIDE);
      const source = readFileSync(sourceOverride || fileURLToPath(url), "utf8");
      const output = typescript.transpileModule(source, { compilerOptions }).outputText;
      return { format: "module", shortCircuit: true, source: output };
    }
    return nextLoad(url, context);
  },
});

test("InstanceSignInForm keeps the CSRF input controlled before and after token loading", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const warnings = [];
  const originalError = console.error;
  console.error = (...args) => warnings.push(args.join(" "));
  t.after(() => {
    console.error = originalError;
  });

  const { InstanceSignInForm } = await import("../../apps/admin/app/(all)/(home)/sign-in-form.tsx");
  await environment.render(React.createElement(InstanceSignInForm));
  await act(async () => Promise.resolve());

  const csrfInput = environment.container.querySelector('input[name="csrfmiddlewaretoken"]');
  assert.ok(csrfInput);
  assert.equal(csrfInput.value, "test-csrf-token");
  assert.equal(
    warnings.some((warning) => warning.includes("changing an uncontrolled input to be controlled")),
    false,
    "the hidden CSRF input must have a string value on its first render"
  );
});

test("WorkspaceStore finishes fetchWorkspaces inside a MobX action", async (t) => {
  const { configure, autorun } = await import(mobxURL);
  configure({ enforceActions: "always" });
  t.after(() => configure({ enforceActions: "observed" }));
  const { WorkspaceStore } = await import("../../apps/admin/store/workspace.store.ts");
  const store = new WorkspaceStore({});
  const warnings = [];
  const originalWarn = console.warn;
  const originalError = console.error;
  console.warn = (...args) => warnings.push(args.join(" "));
  console.error = (...args) => warnings.push(args.join(" "));
  t.after(() => {
    console.warn = originalWarn;
    console.error = originalError;
  });
  const dispose = autorun(() => store.loader);
  t.after(dispose);

  await store.fetchWorkspaces();

  assert.equal(store.loader, "loaded");
  assert.equal(
    warnings.some((warning) => warning.includes("changing (observed) observable values without using an action")),
    false,
    `the asynchronous loader reset must run inside a MobX action; MobX reported: ${warnings.join(" | ")}`
  );
});
