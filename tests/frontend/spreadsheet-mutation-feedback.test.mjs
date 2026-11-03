import assert from "node:assert/strict";
import { registerHooks } from "./helpers/module-hooks.mjs";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
const state = { React, route: { workspaceSlug: "workspace" }, generation: 0, toasts: [], actions: {} };
globalThis.spreadsheetMutation = state;
const dropdown = (name, value) =>
  `export function ${name}(props) { return globalThis.spreadsheetMutation.React.createElement('button', { disabled: props.disabled, onClick: () => props.onChange(${value}) }, 'Change'); }`;
const stubs = {
  "@tanstack/react-router": "export const useParams = () => globalThis.spreadsheetMutation.route;",
  "@tanstack/react-query": "export const useQueryClient = () => ({});",
  "@/hooks/use-current-user": "export const useCurrentUserId = () => 'account';",
  "@/hooks/use-identity-generation":
    "export const useIdentityGeneration = () => globalThis.spreadsheetMutation.generation;",
  "@/hooks/use-issues-actions": "export const useIssuesActions = () => globalThis.spreadsheetMutation.actions;",
  "@/hooks/use-issue-list-type": "export const useIssueListType = () => 'project';",
  "@plane/propel/toast":
    "export const TOAST_TYPE = { ERROR: 'error' }; export const setToast = toast => globalThis.spreadsheetMutation.toasts.push(toast);",
  "@/components/dropdowns/cycle": dropdown("CycleDropdown", "globalThis.spreadsheetMutation.cycleId"),
  "@/components/dropdowns/module/dropdown": dropdown("ModuleDropdown", "['retained', 'added']"),
  "@/helpers/issue-filter.helper": "export const shouldRenderColumn = () => true;",
  "../utils":
    "export const SPREADSHEET_COLUMNS = { priority: props => globalThis.spreadsheetMutation.React.createElement('button', { onClick: () => props.onChange(props.issue, { priority: 'high' }) }, 'Change') };",
  "../properties/with-display-properties-HOC": "export const WithDisplayPropertiesHOC = ({children}) => children;",
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { SpreadsheetCycleColumn } =
  await import("../../apps/web/core/components/issues/issue-layouts/spreadsheet/columns/cycle-column.tsx");
const { SpreadsheetModuleColumn } =
  await import("../../apps/web/core/components/issues/issue-layouts/spreadsheet/columns/module-column.tsx");
const { IssueColumn } =
  await import("../../apps/web/core/components/issues/issue-layouts/spreadsheet/issue-column.tsx");

const examples = [
  {
    name: "cycle assignment",
    Component: SpreadsheetCycleColumn,
    action: "addCycleToIssue",
    cycleId: "new-cycle",
    args: ["project", "new-cycle", "issue"],
  },
  {
    name: "cycle removal",
    Component: SpreadsheetCycleColumn,
    action: "removeCycleFromIssue",
    cycleId: null,
    args: ["project", "old-cycle", "issue"],
  },
  {
    name: "module selection",
    Component: SpreadsheetModuleColumn,
    action: "changeModulesInIssue",
    args: ["project", "issue", ["added"], ["removed"]],
  },
  {
    name: "property update",
    Component: IssueColumn,
    action: "updateIssue",
    args: ["project", "issue", { priority: "high" }],
  },
];

for (const example of examples) {
  void test(`spreadsheet ${example.name} reports failure, allows retry, and suppresses stale feedback`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    state.toasts = [];
    state.generation = 0;
    state.route = { workspaceSlug: "workspace" };
    state.cycleId = example.cycleId;
    const requests = [];
    const update = (...args) => {
      const request = { ...Promise.withResolvers(), args };
      requests.push(request);
      return request.promise;
    };
    state.actions = { [example.action]: update };
    const issue = { id: "issue", project_id: "project", cycle_id: "old-cycle", module_ids: ["retained", "removed"] };
    const render = () => {
      const child = React.createElement(example.Component, {
        issue,
        issueDetail: issue,
        onClose() {},
        disabled: false,
        displayProperties: {},
        disableUserActions: false,
        property: "priority",
        updateIssue: update,
        isEstimateEnabled: true,
      });
      return env.render(
        example.Component === IssueColumn
          ? React.createElement(
              "table",
              null,
              React.createElement("tbody", null, React.createElement("tr", null, child))
            )
          : child
      );
    };
    const click = () => act(async () => env.container.querySelector("button").click());
    await render();
    await click();
    assert.deepEqual(requests[0].args, example.args);
    await act(async () => requests[0].reject(new Error("private failure")));
    assert.equal(state.toasts[0].title, "Could not update work item");
    assert.doesNotMatch(JSON.stringify(state.toasts), /private failure/);
    await click();
    await act(async () => requests[1].resolve());
    assert.equal(state.toasts.length, 1);
    await click();
    state.generation++;
    await render();
    await act(async () => requests[2].reject(new Error("old session")));
    assert.equal(state.toasts.length, 1);
    await click();
    state.route = { workspaceSlug: "replacement" };
    await render();
    state.route = { workspaceSlug: "workspace" };
    await render();
    await act(async () => requests[3].reject(new Error("old route")));
    assert.equal(state.toasts.length, 1);
  });
}
