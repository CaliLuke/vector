import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerPermissionUiStubs } from "./helpers/permission-ui-fixture.mjs";

const { ALL_ISSUES } = webRequire("@plane/constants");

registerPermissionUiStubs("/sub-issues/issues-list/root.tsx", {
  "@/components/issues/issue-layouts/use-group-columns": `export const useGroupColumns = () => [{id:${JSON.stringify(ALL_ISSUES)}}, {id:"high"}, {id:"low"}];`,
  "@/components/issues/issue-layouts/utils": "export const isWorkspaceLevel = () => false;",
  "@/hooks/use-work-item-detail":
    "export const useWorkItemSubItems = (_workspace,_project,parentId) => ({subItems:parentId === globalThis.subWorkItemFixture.rootId ? Object.values(globalThis.subWorkItemFixture.rows) : []});",
  "@plane/i18n": "export const useTranslation = () => ({t:key=>key});",
  "./list-group": `import { React } from "${new URL("./helpers/react.mjs", import.meta.url).href}";
    export const SubIssuesListGroup = ({workItems,group}) => React.createElement("output", {"data-group":group.id}, workItems.map(item=>item.id).join(","));`,
  "@/components/empty-state/section-empty-state-root": `import { React } from "${new URL("./helpers/react.mjs", import.meta.url).href}";
    export const SectionEmptyState = ({actionElement}) => React.createElement("section", {"data-empty":true}, actionElement);`,
});
const { SubWorkItemPreferencesProvider, useSubWorkItemPreferences } =
  await import("../../apps/web/core/providers/sub-work-item-preferences.tsx");
const { SubIssuesListRoot } =
  await import("../../apps/web/core/components/issues/issue-detail-widgets/sub-issues/issues-list/root.tsx");
const { EIssueServiceType, EIssuesStoreType } = webRequire("@plane/types");

const scope = { workspaceSlug: "acme", projectId: "p1", parentId: "parent", serviceType: EIssueServiceType.ISSUES };
function Controls({ current = scope }) {
  const { preferences, updatePreferences, resetPreferences } = useSubWorkItemPreferences(current);
  return React.createElement(
    React.Fragment,
    null,
    React.createElement("output", { "data-preferences": true }, JSON.stringify(preferences)),
    React.createElement(
      "button",
      {
        "data-action": "filter",
        onClick: () => updatePreferences({ filters: { priority: ["high"] }, displayFilters: { group_by: "priority" } }),
      },
      "Filter"
    ),
    React.createElement(
      "button",
      { "data-action": "display", onClick: () => updatePreferences({ displayProperties: { key: false } }) },
      "Hide identifier"
    ),
    React.createElement("button", { "data-action": "reset", onClick: resetPreferences }, "Reset")
  );
}
async function setup(t) {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  return environment;
}
async function click(environment, action) {
  await act(async () => environment.container.querySelector(`[data-action="${action}"]`).click());
}
function readPreferences(environment) {
  return JSON.parse(environment.container.querySelector("[data-preferences]").textContent);
}

await test("sub-item preferences retain sibling fields and isolate workspace/project/item/service scopes", async (t) => {
  const environment = await setup(t);
  const otherScopes = [
    { workspaceSlug: "other" },
    { projectId: "p2" },
    { parentId: "other" },
    { serviceType: EIssueServiceType.EPICS },
  ];
  function ReadOtherScope({ change, index }) {
    const { preferences } = useSubWorkItemPreferences({ ...scope, ...change });
    return React.createElement("output", { "data-other-scope": index }, JSON.stringify(preferences));
  }
  const render = (current) =>
    environment.render(
      React.createElement(
        SubWorkItemPreferencesProvider,
        null,
        React.createElement(Controls, { current }),
        ...otherScopes.map((change, index) =>
          React.createElement(ReadOtherScope, { key: JSON.stringify(change), change, index })
        )
      )
    );
  await render(scope);
  await click(environment, "filter");
  await click(environment, "display");
  const saved = readPreferences(environment);
  assert.equal(saved.displayProperties.key, false);
  assert.equal(saved.displayProperties.state, true);
  assert.deepEqual(saved.filters.priority, ["high"]);
  assert.equal(saved.displayFilters.group_by, "priority");
  environment.container.querySelectorAll("[data-other-scope]").forEach((output) => {
    const preferences = JSON.parse(output.textContent);
    assert.deepEqual(preferences.filters, {});
    assert.equal(preferences.displayProperties.key, true);
  });
  await render({ ...scope, parentId: "other" });
  assert.deepEqual(readPreferences(environment).filters, {});
  await render(scope);
  assert.deepEqual(readPreferences(environment), saved);
  await click(environment, "reset");
  assert.deepEqual(readPreferences(environment).filters, {});
  assert.equal(readPreferences(environment).displayProperties.key, true);
});

await test("session-keyed preference remount drops previous actor drafts", async (t) => {
  const environment = await setup(t);
  const render = (session) =>
    environment.render(
      React.createElement(SubWorkItemPreferencesProvider, { key: session }, React.createElement(Controls))
    );
  await render("session-a");
  await click(environment, "filter");
  await render("session-b");
  assert.deepEqual(readPreferences(environment).filters, {});
});

await test("sub-item list reacts to native preferences and current unarchived item rows", async (t) => {
  const environment = await setup(t);
  const rows = {
    high: { id: "high", priority: "high", created_at: "2026-10-01", archived_at: null },
    low: { id: "low", priority: "low", created_at: "2026-10-02", archived_at: null },
    archived: { id: "archived", priority: "high", archived_at: "2026-10-03" },
  };
  globalThis.subWorkItemFixture = {
    rows,
    rootId: scope.parentId,
  };
  t.after(() => delete globalThis.subWorkItemFixture);
  const render = () =>
    environment.render(
      React.createElement(
        SubWorkItemPreferencesProvider,
        null,
        React.createElement(Controls),
        React.createElement(SubIssuesListRoot, {
          workspaceSlug: scope.workspaceSlug,
          projectId: scope.projectId,
          parentIssueId: scope.parentId,
          rootIssueId: scope.parentId,
          canEdit: true,
          storeType: EIssuesStoreType.PROJECT,
        })
      )
    );
  await render();
  assert.equal(environment.container.querySelector(`[data-group="${ALL_ISSUES}"]`).textContent, "low,high");
  await click(environment, "filter");
  assert.equal(environment.container.querySelector('[data-group="high"]').textContent, "high");
  assert.equal(environment.container.querySelector('[data-group="low"]').textContent, "");
  globalThis.subWorkItemFixture.rows = {
    ...rows,
    high: { ...rows.high, priority: "low" },
  };
  await render();
  assert.ok(environment.container.querySelector("[data-empty]"), "a row update immediately changes the filtered list");
  await click(environment, "reset");
  assert.equal(environment.container.querySelector(`[data-group="${ALL_ISSUES}"]`).textContent, "low,high");
});
