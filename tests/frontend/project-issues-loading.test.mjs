import assert from "node:assert/strict";
import test from "node:test";
import { createRequire, registerHooks } from "node:module";
import { pathToFileURL } from "node:url";
import path from "node:path";
const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { createReactDomTestEnvironment } = await fromRoot("tests/frontend/helpers/react-dom.mjs");
const { React, act, webRequire } = await fromRoot("tests/frontend/helpers/react.mjs");
const { registerWebModuleHooks } = await fromRoot("tests/frontend/helpers/web-module-hooks.mjs");

registerWebModuleHooks();
const requireFromWeb = createRequire(path.resolve("apps/web/package.json"));
const reactQueryModuleUrl = pathToFileURL(requireFromWeb.resolve("@tanstack/react-query")).href;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@tanstack/react-query") return { shortCircuit: true, url: reactQueryModuleUrl };
    return nextResolve(specifier, context);
  },
});
const { QueryClient, QueryClientProvider } = webRequire("@tanstack/react-query");
const { ALL_ISSUES } = webRequire("@plane/constants");
const { IssueService } = await import("../../apps/web/core/services/issue/index.ts");
const { seedIssueQueryScope } = await import("./helpers/issue-query-fixtures.mjs");
const { useIssueListQuery } = await import("../../apps/web/core/hooks/use-issue-list-query.ts");

const scope = {
  accountId: "alice",
  sessionGeneration: 0,
  workspaceSlug: "acme",
  workspaceRole: 15,
  membershipRevision: "membership-1",
  projectId: "p1",
  projectRole: 20,
  projectMembershipRole: 20,
};
const row = (id) => ({ id, project_id: "p1", state_id: "state-1" });
const groupedResponse = {
  grouped_by: "state",
  total_count: 4,
  count: 2,
  total_pages: 2,
  total_results: 3,
  results: {
    "state-1": {
      total_results: 2,
      results: {
        "priority-high": { results: [row("one")], total_results: 1 },
        "priority-low": { results: [row("other")], total_results: 1 },
      },
    },
    "state-2": { results: [row("third")], total_results: 3 },
  },
};

await test("mounted list keeps per-group counts and paginates only the selected group", async (t) => {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(async () => {
    await env.unmount(() => client.clear());
  });
  seedIssueQueryScope(client, scope);
  const read = {
    scope,
    pagination: { canGroup: true, perPageCount: 1, groupedBy: "state_id" },
    request: {
      family: "project",
      input: {
        workspaceSlug: "acme",
        projectId: "p1",
        richFilters: {},
        displayFilters: { layout: "kanban", group_by: "state", sub_group_by: null },
      },
      isReadCurrent: () => true,
    },
  };
  const nextPage = Promise.withResolvers();
  const pendingPage = Promise.withResolvers();
  let requests = [];
  const signals = [];
  t.mock.method(IssueService.prototype, "getIssues", async (_workspace, _project, params, options) => {
    requests.push(params);
    signals.push(options.signal);
    if (requests.length === 1) return groupedResponse;
    if (requests.length === 2) return nextPage.promise;
    return pendingPage.promise;
  });
  let latest;
  let currentRead = read;
  function Probe() {
    const state = useIssueListQuery(currentRead);
    React.useEffect(() => {
      latest = state;
    }, [state]);
    return null;
  }
  await env.render(React.createElement(QueryClientProvider, { client }, React.createElement(Probe)));
  await act(async () => new Promise((resolve) => setTimeout(resolve, 20)));

  assert.deepEqual(latest.groupedIssueIds["state-1"], {
    "priority-high": ["one"],
    "priority-low": ["other"],
  });
  assert.deepEqual(latest.groupedIssueIds["state-2"], ["third"]);
  assert.equal(latest.getGroupIssueCount("state-1", "priority-high"), 1);
  assert.equal(latest.getGroupIssueCount("state-1", "priority-low"), 1);
  assert.equal(latest.getGroupIssueCount("state-1", "priority-high", true), 1);
  assert.equal(latest.getGroupIssueCount("state-2"), 3);
  assert.equal(latest.groupedIssueCount[ALL_ISSUES], 4);
  assert.equal(latest.getGroupIssueCount(), 4, "the ungrouped group path normalizes to ALL_ISSUES");
  const groupQueries = client
    .getQueryCache()
    .findAll()
    .filter((query) => query.queryKey.includes("group-pages"));
  currentRead = { ...read, scope: { ...read.scope }, request: { ...read.request, input: { ...read.request.input } } };
  await act(async () => {
    await env.render(React.createElement(QueryClientProvider, { client }, React.createElement(Probe)));
  });
  assert.deepEqual(
    client
      .getQueryCache()
      .findAll()
      .filter((query) => query.queryKey.includes("group-pages")),
    groupQueries,
    "a structurally equal read keeps the same group Query objects"
  );

  let pagination;
  await act(async () => {
    pagination = latest.fetchNextIssues("state-2");
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  assert.equal(latest.getIssueLoader("state-2"), "pagination");
  assert.equal(latest.getIssueLoader("state-1", "priority-high"), undefined);
  assert.equal(requests.length, 2);
  assert.equal(requests[1].state, "state-2");

  await act(async () => {
    nextPage.resolve({ results: [row("two")], total_count: 1, next_page_results: true });
    await pagination;
  });
  assert.deepEqual(latest.groupedIssueIds["state-1"], {
    "priority-high": ["one"],
    "priority-low": ["other"],
  });
  assert.deepEqual(latest.groupedIssueIds["state-2"], ["third", "two"]);
  assert.equal(latest.getGroupIssueCount("state-2"), 3);
  assert.equal(requests.length, 2);

  await act(async () => {
    latest.fetchNextIssues("state-2");
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  assert.equal(requests.length, 3);
  await act(async () => env.render(null));
  assert.equal(signals[2].aborted, true, "unmount cancels the pending group read");
});
