import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
const state = {
  React,
  route: { workspaceSlug: "workspace", projectId: "project" },
  generation: 0,
  toasts: [],
  update: undefined,
};
globalThis.ganttMutationFailure = state;
const stubs = {
  "@tanstack/react-query": "export const useQueryClient = () => ({});",
  "@tanstack/react-router": "export const useParams = () => globalThis.ganttMutationFailure.route;",
  "@/hooks/use-current-user": "export const useCurrentUserId = () => 'account';",
  "@/hooks/use-identity-generation":
    "export const useIdentityGeneration = () => globalThis.ganttMutationFailure.generation;",
  "@plane/propel/toast":
    "export const TOAST_TYPE = { ERROR: 'error' }; export const setToast = toast => globalThis.ganttMutationFailure.toasts.push(toast);",
  "./chart/root":
    "export function ChartViewRoot(props) { globalThis.ganttMutationFailure.update = props.blockUpdateHandler; return null; }",
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { GanttChartRoot } = await import("../../apps/web/core/components/gantt-chart/root.tsx");
const item = { id: "module" };
const props = {
  title: "Modules",
  loaderTitle: "Modules",
  blockIds: [item.id],
  blockItems: [item],
  blockToRender: () => null,
  sidebarToRender: () => null,
};

void test("Gantt owns synchronous and rejected updates, then allows a successful retry", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  state.toasts = [];
  let calls = 0;
  const update = (_item, payload) => {
    assert.equal(_item, item);
    assert.equal(payload.sort_order.newSortOrder, 123);
    calls++;
    if (calls === 1) throw new Error("sync");
    if (calls === 2) return Promise.reject(new Error("transport"));
    return Promise.resolve();
  };
  await env.render(React.createElement(GanttChartRoot, { ...props, blockUpdateHandler: update }));
  await act(async () => state.update(item, { sort_order: { newSortOrder: 123 } }));
  await act(async () => state.update(item, { sort_order: { newSortOrder: 123 } }));
  await act(async () => state.update(item, { sort_order: { newSortOrder: 123 } }));
  assert.equal(calls, 3);
  assert.equal(state.toasts.length, 2);
  assert.match(state.toasts[0].message, /try again/);
});

for (const change of ["project", "session", "unmount"])
  void test(`late Gantt failure is ignored after ${change}`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    state.toasts = [];
    const pending = Promise.withResolvers();
    const element = React.createElement(GanttChartRoot, { ...props, blockUpdateHandler: () => pending.promise });
    await env.render(element);
    await act(async () => state.update(item, {}));
    if (change === "project") state.route = { ...state.route, projectId: "replacement" };
    if (change === "session") state.generation++;
    await env.render(change === "unmount" ? null : React.cloneElement(element));
    await act(async () => pending.reject(new Error("late")));
    assert.deepEqual(state.toasts, []);
  });
