import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerWebModuleHooks();
globalThis.typeTableReact = React;
const stubs = {
  "@/hooks/use-intersection-observer": `export const useIntersectionObserver=()=>{};`,
  "@/hooks/use-table-keyboard-navigation": `export const useTableKeyboardNavigation=()=>()=>{};`,
  "@/helpers/issue-filter.helper": `export const shouldRenderColumn=key=>key!=="estimate";`,
  "./issue-row": `export const SpreadsheetIssueRow=()=>null;`,
  "./spreadsheet-header": `const R=globalThis.typeTableReact;export const SpreadsheetHeader=()=>R.createElement("thead",null,R.createElement("tr",null,R.createElement("th",null,"Work items"),R.createElement("th",null,"State")));`,
  "@plane/ui": `export const Row=props=>globalThis.typeTableReact.createElement("div",props);`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const bootstrap = await createReactDomTestEnvironment();
const { SpreadsheetTable } =
  await import("../../apps/web/core/components/issues/issue-layouts/spreadsheet/spreadsheet-table.tsx");
const { IssueListViewProvider } = await import("../../apps/web/core/providers/issue-list-view.tsx");
await bootstrap.unmount();
await test("pagination counts real columns, excluding the type badge and unavailable columns", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const props = {
    customColumns: [],
    displayProperties: { key: true, issue_type: true, state: true, priority: false, modules: true, estimate: true },
    displayFilters: {},
    issueIds: [],
    isEstimateEnabled: false,
    onEditProperties: () => {},
    handleDisplayFilterUpdate: () => {},
    quickActions: () => null,
    updateIssue: undefined,
    canEditProperties: () => false,
    portalElement: { current: null },
    selectionHelpers: {},
    canLoadMoreIssues: true,
    loadMoreIssues: () => {},
    containerRef: { current: null },
    spreadsheetColumnsList: ["state", "priority", "estimate"],
  };
  const listView = {
    issuesById: {},
    groupedIssueIds: {},
    getGroupIssueCount: () => undefined,
    getPaginationData: () => undefined,
    getIssueLoader: () => undefined,
    fetchNextIssues: async () => {},
  };
  const render = (tableProps) =>
    env.render(
      React.createElement(
        IssueListViewProvider,
        { readState: listView },
        React.createElement(SpreadsheetTable, tableProps)
      )
    );
  await render(props);
  assert.equal(env.container.querySelectorAll("thead th").length, 2);
  assert.equal(env.container.querySelectorAll("tfoot tr").length, 3);
  assert.deepEqual(
    [...env.container.querySelectorAll("tfoot tr")].map((row) => row.querySelectorAll("td").length),
    [2, 2, 2]
  );
  await render({
    ...props,
    displayProperties: { ...props.displayProperties, issue_type: false },
  });
  assert.equal(env.container.querySelector("tfoot tr").querySelectorAll("td").length, 2);
});
