import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import test from "node:test";
import { QueryClient, QueryClientProvider, InfiniteQueryObserver } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerSpaceModuleHooks } from "./helpers/web-module-hooks.mjs";

registerSpaceModuleHooks();
globalThis.spaceScopeReact = React;
const require = createRequire(new URL("../../apps/space/package.json", import.meta.url));
const { MemoryRouter, useSearchParams, useNavigate } = createRequire(
  new URL("../../apps/space/package.json", import.meta.url)
)("react-router");
const { SitesIssueService, SitesStateService } = require("@plane/services");
const stubs = {
  "@plane/i18n": "export const useTranslation = () => ({t: key => key});",
  "@plane/propel/tooltip": "export const Tooltip = ({children}) => children;",
  "@/components/issues/filters/applied-filters/root": "export const IssueAppliedFilters = () => null;",
  "@/components/issues/peek-overview": "export const IssuePeekOverview = () => null;",
  "@/components/common/logo-spinner":
    "export const LogoSpinner = () => globalThis.spaceScopeReact.createElement('span', null, 'Loading');",
  "./state-group":
    "export const IssueStateGroup = ({firstPage}) => globalThis.spaceScopeReact.createElement('output', null, firstPage.issues.map(issue => issue.id).join(','));",
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { IssuesLayoutsRoot } = await import("../../apps/space/components/issues/issue-layouts/root.tsx");
const { IssuesLayoutSelection } = await import("../../apps/space/components/issues/navbar/layout-selection.tsx");
const { usePublicIssueView } = await import("../../apps/space/hooks/use-public-issue-view.ts");
const { publishSettingsOptions } = await import("../../apps/space/queries/publishing.ts");
const { publicIssuesOptions, stateIssuesOptions, firstStatePage } = await import("../../apps/space/queries/issues.ts");
const filters = { state: [], priority: [], labels: [] };
const response = (id) => ({ results: { state: { results: [{ id }], total_results: 1 } }, total_count: 1 });
const settle = () => act(async () => new Promise((resolve) => setTimeout(resolve, 20)));
async function setup(t) {
  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  for (const anchor of ["A", "B"])
    client.setQueryData(publishSettingsOptions(anchor).queryKey, {
      anchor,
      view_props: { list: true, kanban: anchor === "A" },
    });
  t.mock.method(SitesStateService.prototype, "list", async () => [
    { id: "state", group: "started", name: "State", sequence: 1 },
  ]);
  t.after(async () => {
    await environment.unmount();
    client.clear();
  });
  return {
    environment,
    client,
    render: (element) =>
      environment.render(
        React.createElement(QueryClientProvider, { client }, React.createElement(MemoryRouter, null, element))
      ),
  };
}
await test("public pages keep cached A rows when returning from B without a parallel store", async (t) => {
  const { environment, render } = await setup(t);
  t.mock.method(SitesIssueService.prototype, "list", async (anchor) => response(`${anchor}-issue`));
  await render(React.createElement(IssuesLayoutsRoot, { anchor: "A" }));
  await settle();
  assert.match(environment.container.textContent, /A-issue/);
  await render(React.createElement(IssuesLayoutsRoot, { anchor: "B" }));
  await settle();
  assert.match(environment.container.textContent, /B-issue/);
  await render(React.createElement(IssuesLayoutsRoot, { anchor: "A" }));
  await settle();
  assert.match(environment.container.textContent, /A-issue/);
  assert.doesNotMatch(environment.container.textContent, /B-issue/);
});
await test("a late A response cannot replace the currently displayed B issues", async (t) => {
  const { environment, render } = await setup(t);
  let finishA;
  t.mock.method(SitesIssueService.prototype, "list", (anchor) =>
    anchor === "A"
      ? new Promise((resolve) => {
          finishA = resolve;
        })
      : Promise.resolve(response("B-issue"))
  );
  await render(React.createElement(IssuesLayoutsRoot, { anchor: "A" }));
  await render(React.createElement(IssuesLayoutsRoot, { anchor: "B" }));
  await settle();
  await act(async () => finishA(response("A-late")));
  await settle();
  assert.match(environment.container.textContent, /B-issue/);
  assert.doesNotMatch(environment.container.textContent, /A-late/);
});
await test("publishing settings determine the layout options for each anchor", async (t) => {
  const { environment, render } = await setup(t);
  await render(React.createElement(IssuesLayoutSelection, { anchor: "A" }));
  assert.equal(environment.container.querySelectorAll("button").length, 2);
  await render(React.createElement(IssuesLayoutSelection, { anchor: "B" }));
  assert.equal(environment.container.querySelectorAll("button").length, 1);
});
function Harness() {
  const view = usePublicIssueView("A");
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(
      "output",
      null,
      JSON.stringify({ filters: view.filters, layout: view.layout, peek: params.get("peekId") })
    ),
    React.createElement("button", { onClick: () => setParams({ peekId: "issue", labels: "label" }) }, "Peek"),
    React.createElement("button", { onClick: () => view.setFilter("priority", ["high"]) }, "Filter"),
    React.createElement("button", { onClick: () => view.setLayout("kanban") }, "Layout"),
    React.createElement("button", { onClick: () => navigate(-1) }, "Back")
  );
}
await test("URL filters, peek selection and browser back share one state", async (t) => {
  const { environment, render } = await setup(t);
  await render(React.createElement(Harness));
  const click = async (text) =>
    act(async () =>
      [...environment.container.querySelectorAll("button")].find((button) => button.textContent === text).click()
    );
  await click("Peek");
  await click("Filter");
  await click("Layout");
  const read = () => JSON.parse(environment.container.querySelector("output").textContent);
  assert.deepEqual(read(), {
    filters: { ...filters, priority: ["high"], labels: ["label"] },
    layout: "kanban",
    peek: "issue",
  });
  await click("Back");
  assert.equal(read().layout, "list");
  assert.deepEqual(read().filters.priority, ["high"]);
  await click("Back");
  assert.deepEqual(read().filters.priority, []);
});
await test("state pagination uses the returned cursor, preserves earlier pages, and isolates filters", async (t) => {
  const { client } = await setup(t);
  const calls = [];
  t.mock.method(SitesIssueService.prototype, "list", async (anchor, params) => {
    calls.push({ anchor, ...params });
    return {
      results: [{ id: params.cursor === "50:1:0" ? "second" : "third" }],
      total_count: 3,
      next_page_results: params.cursor === "50:1:0",
      next_cursor: "server-next",
    };
  });
  const first = firstStatePage(
    { ...response("first"), results: { state: { results: [{ id: "first" }], total_results: 3 } } },
    "state"
  );
  const options = stateIssuesOptions("A", { ...filters, priority: ["high"] }, "state");
  client.setQueryData(options.queryKey, { pages: [first], pageParams: ["50:0:0"] });
  const observer = new InfiniteQueryObserver(client, options);
  const unsubscribe = observer.subscribe(() => {});
  t.after(unsubscribe);
  await observer.fetchNextPage();
  await observer.fetchNextPage();
  assert.deepEqual(
    observer.getCurrentResult().data.pages.flatMap((page) => page.issues.map((issue) => issue.id)),
    ["first", "second", "third"]
  );
  assert.deepEqual(
    calls.map((call) => [call.anchor, call.state, call.priority, call.cursor]),
    [
      ["A", "state", "high", "50:1:0"],
      ["A", "state", "high", "server-next"],
    ]
  );
  assert.equal(observer.getCurrentResult().hasNextPage, false);
  assert.equal(client.getQueryData(stateIssuesOptions("B", filters, "state").queryKey), undefined);
  assert.equal(client.getQueryData(publicIssuesOptions("A", filters).queryKey), undefined);
});
