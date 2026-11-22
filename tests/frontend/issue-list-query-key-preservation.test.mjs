import * as ContractEnums from "../../packages/types/dist/index.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { createRequire } from "./helpers/module-hooks.mjs";
import path from "node:path";
import { pathToFileURL } from "node:url";

const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { createReactDomTestEnvironment } = await fromRoot("tests/frontend/helpers/react-dom.mjs");
const { React, act, webRequire } = await fromRoot("tests/frontend/helpers/react.mjs");
const { registerWebModuleHooks } = await fromRoot("tests/frontend/helpers/web-module-hooks.mjs");
registerWebModuleHooks();
const requireFromWeb = createRequire(path.resolve("apps/web/package.json"));
const reactQueryModuleUrl = pathToFileURL(requireFromWeb.resolve("@tanstack/react-query")).href;
const { registerHooks } = await import("./helpers/module-hooks.mjs");
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@tanstack/react-query") return { shortCircuit: true, url: reactQueryModuleUrl };
    return nextResolve(specifier, context);
  },
});
const { QueryClient, QueryClientProvider } = webRequire("@tanstack/react-query");
const { projectIssueScope, seedIssueQueryScope } = await import("./helpers/issue-query-fixtures.mjs");
const { IssueService } = await fromRoot("apps/web/core/services/issue/index.ts");
const { WorkspaceService } = await fromRoot("apps/web/core/services/workspace.service.ts");
const { issueListGroupOptions, issueListKeys } = await import("../../apps/web/core/queries/issue-lists.ts");
const { captureIssueListMutationOwner, currentIssueListMutationOwner } =
  await import("../../apps/web/core/queries/issue-list-mutations.ts");
const { useIssueListQuery } = await import("../../apps/web/core/hooks/use-issue-list-query.ts");

const scope = {
  accountId: "alice",
  sessionGeneration: 0,
  workspaceSlug: "acme",
  workspaceRole: ContractEnums.EUserPermissions.MEMBER,
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
        displayFilters: { layout: ContractEnums.EIssueLayoutTypes.KANBAN, group_by: "state", sub_group_by: null },
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

  const owner = currentIssueListMutationOwner(client, "acme", "p1", ContractEnums.EIssueServiceType.ISSUES, () => true);
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

function FilterTransitionProbe({ read }) {
  const state = useIssueListQuery(read);
  const status = state.isUpdatingResults ? "updating" : state.isPending ? "loading" : "ready";
  return React.createElement("output", { "data-state": status }, Object.keys(state.issuesById).join(","));
}

await test("filter transitions retain only a settled view in the same verified scope", async (t) => {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity, staleTime: Infinity } },
  });
  t.after(() => env.unmount(() => client.clear()));
  const project = projectIssueScope();
  const { rolesKey } = seedIssueQueryScope(client, { ...project, projectRoles: { p1: 20, p2: 20 } });
  const makeRead = (filter, projectId = "p1", isReadCurrent = () => true) => ({
    scope: { ...project, projectId },
    pagination: { canGroup: true, perPageCount: 1, groupedBy: "state_id" },
    request: {
      family: "project",
      input: {
        workspaceSlug: "acme",
        projectId,
        richFilters: { state_id__in: filter },
        displayFilters: { layout: ContractEnums.EIssueLayoutTypes.LIST, group_by: "state", sub_group_by: null },
      },
      isReadCurrent,
    },
    isPresentationCurrent: () => true,
  });
  const original = makeRead("first", "p1", () => false);
  client.setQueryData(issueListKeys.list(original), { generation: "original", response });
  const requests = [];
  t.mock.method(IssueService.prototype, "getIssues", async (_workspace, projectId, _params, { signal }) => {
    const pending = Promise.withResolvers();
    requests.push({ projectId, signal, pending });
    return pending.promise;
  });
  const render = (read) =>
    env.render(
      React.createElement(QueryClientProvider, { client }, React.createElement(FilterTransitionProbe, { read }))
    );
  const view = () => [
    env.container.querySelector("output").dataset.state,
    env.container.querySelector("output").textContent,
  ];

  await render(original);
  assert.deepEqual(view(), ["ready", "i1"], "cached rows can render before the new transport owner commits");
  await render(makeRead("second"));
  assert.deepEqual(view(), ["updating", "i1"]);
  await render(makeRead("third"));
  assert.deepEqual(view(), ["updating", "i1"]);
  assert.equal(requests[0].signal.aborted, true);
  await act(async () => {
    requests[0].pending.resolve({
      ...response,
      results: { "state-1": { results: [{ ...row, id: "superseded" }], total_results: 1 } },
    });
    await new Promise((resolve) => setTimeout(resolve, 10));
  });
  assert.deepEqual(view(), ["updating", "i1"]);
  await act(async () => {
    requests[1].pending.resolve({
      ...response,
      results: { "state-1": { results: [{ ...row, id: "latest" }], total_results: 1 } },
    });
    await new Promise((resolve) => setTimeout(resolve, 10));
  });
  assert.deepEqual(view(), ["ready", "latest"]);

  await render(makeRead("fourth"));
  assert.deepEqual(view(), ["updating", "latest"]);
  client.removeQueries({ queryKey: rolesKey, exact: true });
  client.setQueryData(rolesKey, { p1: 20, p2: 20 });
  await render(makeRead("fourth"));
  assert.deepEqual(view(), ["loading", ""], "replaced permission owners cannot retain earlier rows");
  await render(makeRead("fourth", "p2"));
  assert.deepEqual(view(), ["loading", ""], "a project switch starts with its own presentation");
  assert.equal(requests.at(-1).projectId, "p2");
});
