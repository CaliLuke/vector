import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { pathToFileURL } from "node:url";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { createReactDomTestEnvironment } = await import("./helpers/react-dom.mjs");
const { React, act, webRequire } = await import("./helpers/react.mjs");
const { createRouterFixture, routerModuleUrl } = await import("./helpers/router.mjs");
const reactQueryModuleUrl = pathToFileURL(webRequire.resolve("@tanstack/react-query")).href;
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@tanstack/react-router") return nextResolve(routerModuleUrl, context);
    if (specifier === "@tanstack/react-query") return { shortCircuit: true, url: reactQueryModuleUrl };
    if (context.parentURL?.endsWith("/use-group-dragndrop.ts") && specifier === "./use-issues-actions")
      return {
        shortCircuit: true,
        url: "data:text/javascript,export const useIssuesActions=()=>globalThis.dndHarness.actions;",
      };
    if (context.parentURL?.endsWith("/use-group-dragndrop.ts") && specifier === "./use-issue-filter-preferences")
      return {
        shortCircuit: true,
        url: "data:text/javascript,export const useIssueFilterPreferences=()=>({requestInput:undefined});",
      };
    return nextResolve(specifier, context);
  },
});
const { EIssuesStoreType } = await import("../../packages/types/dist/index.mjs");
const { ALL_ISSUES } = await import("../../packages/constants/dist/index.mjs");
const { useGroupIssuesDragNDrop } = await import("../../apps/web/core/hooks/use-group-dragndrop.ts");

async function createDragFixture(t) {
  const environment = await createReactDomTestEnvironment();
  const { QueryClient, QueryClientProvider } = webRequire("@tanstack/react-query");
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const router = createRouterFixture("/acme/projects/p1/issues/");
  t.after(async () => {
    await environment.render(null);
    await client.cancelQueries();
    client.clear();
    await environment.unmount();
  });
  const view = {
    client,
    render: (element) => router.render(environment, React.createElement(QueryClientProvider, { client }, element)),
  };
  view.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  return view;
}

function Probe({ orderBy, projectedIssueGroups, issuesById, canonicalIssueGroups, setDrop }) {
  const handleDrop = useGroupIssuesDragNDrop(
    EIssuesStoreType.PROJECT,
    orderBy,
    undefined,
    undefined,
    projectedIssueGroups,
    issuesById,
    canonicalIssueGroups
  );
  setDrop(handleDrop);
  return null;
}

async function renderDropHandler(view, actions, issuesById, projectedIssueGroups, canonicalIssueGroups, orderBy) {
  globalThis.dndHarness = { actions };
  let drop;
  await view.render(
    React.createElement(Probe, {
      orderBy,
      projectedIssueGroups,
      issuesById,
      canonicalIssueGroups,
      setDrop: (handler) => {
        drop = handler;
      },
    })
  );
  return drop;
}

await test("DND ranks against visible native name order and keeps canonical IDs for other sorts", async (t) => {
  const view = await createDragFixture(t);
  t.after(() => {
    delete globalThis.dndHarness;
  });
  const updates = [];
  const issues = {
    moving: { id: "moving", project_id: "p1", sort_order: 1 },
    first: { id: "first", project_id: "p1", sort_order: 20 },
    target: { id: "target", project_id: "p1", sort_order: 30 },
    last: { id: "last", project_id: "p1", sort_order: 40 },
  };
  const actions = { updateIssue: async (projectId, issueId, data) => updates.push({ projectId, issueId, data }) };
  const projectedIssueGroups = { [ALL_ISSUES]: ["first", "target", "last"] };
  const canonicalIssueGroups = { [ALL_ISSUES]: ["target", "first", "last"] };
  let drop = await renderDropHandler(
    view,
    actions,
    issues,
    projectedIssueGroups,
    canonicalIssueGroups,
    "assignees__first_name"
  );
  await act(async () => drop({ id: "moving" }, { id: "target" }));
  assert.equal(updates.at(-1).data.sort_order, 25);

  drop = await renderDropHandler(view, actions, issues, projectedIssueGroups, canonicalIssueGroups, "labels__name");
  await act(async () => drop({ id: "moving" }, { id: "target" }));
  assert.equal(updates.at(-1).data.sort_order, 25);

  drop = await renderDropHandler(view, actions, issues, projectedIssueGroups, canonicalIssueGroups, "priority");
  await act(async () => drop({ id: "moving" }, { id: "target" }));
  assert.equal(updates.at(-1).data.sort_order, 30 - 65535);
});

await test("assignee-name drag to the visible end skips unresolved trailing IDs", async (t) => {
  const view = await createDragFixture(t);
  t.after(() => {
    delete globalThis.dndHarness;
  });
  const updates = [];
  const issues = {
    moving: { id: "moving", project_id: "p1", sort_order: 1 },
    first: { id: "first", project_id: "p1", sort_order: 20 },
    last: { id: "last", project_id: "p1", sort_order: 40 },
  };
  const actions = { updateIssue: async (projectId, issueId, data) => updates.push({ projectId, issueId, data }) };
  const projectedIssueGroups = { [ALL_ISSUES]: ["first", "last", "not-loaded"] };
  const canonicalIssueGroups = { [ALL_ISSUES]: ["first", "last", "not-loaded"] };
  const drop = await renderDropHandler(
    view,
    actions,
    issues,
    projectedIssueGroups,
    canonicalIssueGroups,
    "assignees__first_name"
  );
  await act(async () => drop({ id: "moving" }, { id: "last", canAddIssueBelow: true }));

  assert.equal(updates.length, 1);
  assert.equal(updates[0].data.sort_order, 40 + 65535);
});
