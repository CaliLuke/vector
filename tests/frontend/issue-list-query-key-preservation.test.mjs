import assert from "node:assert/strict";
import test from "node:test";
import { createRequire } from "node:module";
import path from "node:path";
import { pathToFileURL } from "node:url";

const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { createReactDomTestEnvironment } = await fromRoot("tests/frontend/helpers/react-dom.mjs");
const { React, act, webRequire } = await fromRoot("tests/frontend/helpers/react.mjs");
const { registerWebModuleHooks } = await fromRoot("tests/frontend/helpers/web-module-hooks.mjs");
registerWebModuleHooks();
const requireFromWeb = createRequire(path.resolve("apps/web/package.json"));
const reactQueryModuleUrl = pathToFileURL(requireFromWeb.resolve("@tanstack/react-query")).href;
const { registerHooks } = await import("node:module");
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@tanstack/react-query") return { shortCircuit: true, url: reactQueryModuleUrl };
    return nextResolve(specifier, context);
  },
});
const { QueryClient, QueryClientProvider } = webRequire("@tanstack/react-query");
const { seedIssueQueryScope } = await import("./helpers/issue-query-fixtures.mjs");
const { WorkspaceService } = await fromRoot("apps/web/core/services/workspace.service.ts");
const { issueListGroupOptions, issueListKeys } = await import("../../apps/web/core/queries/issue-lists.ts");
const { captureIssueListMutationOwner, currentIssueListMutationOwner } =
  await import("../../apps/web/core/queries/issue-list-mutations.ts");
const { useIssueListQuery } = await import("../../apps/web/core/hooks/use-issue-list-query.ts");

const scope = {
  accountId: "alice",
  sessionGeneration: 0,
  workspaceSlug: "acme",
  workspaceRole: 15,
  membershipRevision: "membership-1",
  projectRoles: [["p1", 20]],
};
const row = { id: "i1", project_id: "p1", state_id: "state-1" };
const response = {
  grouped_by: "state",
  total_count: 2,
  count: 1,
  total_pages: 2,
  total_results: 2,
  next_page_results: true,
  results: { "state-1": { results: [row], total_results: 2 } },
};

await test("workspace group observers preserve undefined key slots for native cancellation", async (t) => {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity, staleTime: Infinity } },
  });
  t.after(async () => {
    await env.unmount(() => client.clear());
  });
  seedIssueQueryScope(client, { ...scope, projectRoles: { p1: 20 } });

  const read = {
    scope,
    pagination: { canGroup: true, perPageCount: 1, groupedBy: "state_id" },
    request: {
      family: "workspace",
      input: {
        workspaceSlug: "acme",
        viewId: "my_issues",
        accountId: "alice",
        richFilters: {},
        displayFilters: { layout: "kanban", group_by: "state", sub_group_by: null },
        isReadScopeCurrent: () => true,
      },
      isReadCurrent: () => true,
    },
  };
  client.setQueryData(issueListKeys.list(read), { generation: "full-read-1", response });
  const page = Promise.withResolvers();
  const started = Promise.withResolvers();
  let requestSignal;
  t.mock.method(WorkspaceService.prototype, "getViewIssues", async (_workspace, _params, options) => {
    requestSignal = options.signal;
    started.resolve();
    return page.promise;
  });

  let latest;
  function Probe() {
    const state = useIssueListQuery(read);
    React.useEffect(() => {
      latest = state;
    }, [state]);
    return null;
  }
  await act(async () => {
    await env.render(React.createElement(QueryClientProvider, { client }, React.createElement(Probe)));
  });

  const fullRead = client.getQueryData(issueListKeys.list(read));
  assert.ok(fullRead);
  const groupOptions = issueListGroupOptions(read, fullRead, { groupId: "state-1" });
  const groupQuery = client.getQueryCache().find({ queryKey: groupOptions.queryKey, exact: true });
  assert.ok(groupQuery);
  assert.equal(groupQuery.queryKey[6], undefined, "the optional catalogue revision remains undefined");

  const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true);
  const captured = captureIssueListMutationOwner(client, owner);
  assert.equal(captured.queries.has(groupQuery), true, "the accepted-write owner captures the native group query");

  let pagination;
  await act(async () => {
    pagination = latest.fetchNextIssues("state-1");
    await started.promise;
  });
  assert.ok(requestSignal instanceof AbortSignal);
  await act(async () => {
    await client.cancelQueries({ queryKey: issueListKeys.list(read) });
    await pagination;
  });
  assert.equal(requestSignal.aborted, true, "native prefix cancellation reaches the page request");
  page.resolve(response);
});
