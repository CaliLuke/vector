import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { QueryClient, QueryClientProvider, useMutation } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
const state = { accountId: "account", generation: 0, editable: true, toasts: [], requests: [] };
state.useUpdate = () =>
  useMutation({
    mutationFn: (value) => {
      const request = { ...Promise.withResolvers(), value };
      state.requests.push(request);
      return request.promise;
    },
  });
globalThis.pageLogoMutation = state;
const stubs = {
  "@/hooks/use-current-user": "export const useCurrentUserId = () => globalThis.pageLogoMutation.accountId;",
  "@/hooks/use-identity-generation":
    "export const useIdentityGeneration = () => globalThis.pageLogoMutation.generation;",
  "@/hooks/use-page-permissions":
    "export const usePagePermissions = () => ({ isContentEditable: globalThis.pageLogoMutation.editable });",
  "@/hooks/use-project-pages": "export const useUpdateProjectPage = () => globalThis.pageLogoMutation.useUpdate();",
  "@plane/propel/toast":
    "export const TOAST_TYPE = { ERROR: 'error' }; export const setToast = toast => globalThis.pageLogoMutation.toasts.push(toast);",
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { usePageLogoPicker } =
  await import("../../apps/web/core/components/pages/editor/header/use-page-logo-picker.ts");

async function fixture(t) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  t.after(() => env.unmount(() => client.clear()));
  state.requests = [];
  state.toasts = [];
  state.editable = true;
  state.generation = 0;
  let controls;
  function Picker({ pageId }) {
    const result = usePageLogoPicker({ id: pageId, project_ids: ["project"] }, "workspace");
    React.useLayoutEffect(() => {
      controls = result.pickerProps;
    });
    return null;
  }
  const render = (pageId = "page") =>
    env.render(React.createElement(QueryClientProvider, { client }, React.createElement(Picker, { pageId })));
  await render();
  const select = () => act(async () => controls.onChange({ type: "icon", value: { name: "House", color: "#123456" } }));
  return { render, select };
}

void test("page icon failures have feedback and the same selection can be retried", async (t) => {
  const f = await fixture(t);
  await f.select();
  assert.deepEqual(state.requests[0].value.logo_props, { in_use: "icon", icon: { name: "House", color: "#123456" } });
  await act(async () => state.requests[0].reject(new Error("private backend detail")));
  assert.equal(state.toasts[0].title, "Could not update page icon");
  assert.doesNotMatch(JSON.stringify(state.toasts), /private backend/);
  await f.select();
  await act(async () => state.requests[1].resolve({}));
  assert.equal(state.toasts.length, 1);
  state.editable = false;
  await f.render();
  await f.select();
  assert.equal(state.requests.length, 2);
});

void test("page icon failure feedback belongs to its original page and session", async (t) => {
  const f = await fixture(t);
  await f.select();
  await f.render("replacement");
  await f.render("page");
  await act(async () => state.requests[0].reject(new Error("old page")));
  assert.deepEqual(state.toasts, []);
  await f.select();
  state.generation++;
  await f.render();
  await act(async () => state.requests[1].reject(new Error("old session")));
  assert.deepEqual(state.toasts, []);
});
