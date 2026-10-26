import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient } from "@tanstack/react-query";
import { pathToFileURL } from "node:url";
import path from "node:path";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { WorkspaceService } = await fromRoot("apps/web/core/services/workspace.service.ts");
const { identityKeys } = await fromRoot("apps/web/core/queries/identity.ts");
const { permissionKeys } = await fromRoot("apps/web/core/queries/permissions.ts");
const { issueListOptions } = await import("../../apps/web/core/queries/issue-lists.ts");
const { getWorkspaceIssueRequestParams } = await fromRoot("apps/web/helpers/issue-list/workspace-request.ts");

const scopeFor = (accountId) => ({
  accountId,
  sessionGeneration: 0,
  workspaceSlug: "acme",
  workspaceRole: 15,
  membershipRevision: "membership-1",
  projectRoles: [],
});
const pagination = { canGroup: false, perPageCount: 20 };
const response = { results: [] };
function client(accountId) {
  const scope = scopeFor(accountId);
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  queryClient.setQueryData(identityKeys.currentUser(), { id: scope.accountId });
  queryClient.setQueryData(
    permissionKeys.workspaceProjectRoles(
      scope.accountId,
      scope.workspaceSlug,
      scope.workspaceRole,
      scope.membershipRevision
    ),
    {}
  );
  queryClient.setQueryData(permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug), {
    is_active: true,
    role: scope.workspaceRole,
    updated_at: scope.membershipRevision,
  });
  return queryClient;
}
function read(viewId, accountId) {
  const scope = scopeFor(accountId);
  const input = {
    kind: "workspace",
    workspaceSlug: scope.workspaceSlug,
    viewId,
    accountId,
    richFilters: {},
    displayFilters: { order_by: "sort_order" },
    isScopeCurrent: () => true,
    isReadScopeCurrent: () => true,
  };
  return {
    scope,
    pagination,
    request: { family: "workspace", input, isReadCurrent: () => true },
  };
}
const workspaceLayoutInput = (displayFilters) => ({
  kind: "workspace",
  workspaceSlug: "acme",
  viewId: "all-issues",
  accountId: "alice",
  richFilters: {},
  displayFilters,
  isScopeCurrent: () => true,
  isReadScopeCurrent: () => true,
});

await test("workspace issue Query builds personal view params from its immutable account snapshot", async (t) => {
  const clients = new Map(["alice", "bob"].map((accountId) => [accountId, client(accountId)]));
  t.after(() => clients.forEach((queryClient) => queryClient.clear()));
  const calls = [];
  t.mock.method(WorkspaceService.prototype, "getViewIssues", async (_workspace, params) => {
    calls.push(params);
    return response;
  });

  await Promise.all(
    [
      ["assigned", "alice"],
      ["assigned", "bob"],
      ["created", "bob"],
      ["subscribed", "bob"],
      ["all-issues", "bob"],
    ].map(([viewId, accountId]) => clients.get(accountId).fetchQuery(issueListOptions(read(viewId, accountId))))
  );

  assert.equal(calls[0].assignees, "alice");
  assert.equal(calls[1].assignees, "bob");
  assert.equal(calls[2].created_by, "bob");
  assert.equal(calls[3].subscriber, "bob");
  assert.equal(calls[4].assignees, undefined);
});

await test("workspace list Query does not start when the captured read scope is stale", async (t) => {
  const queryClient = client("alice");
  t.after(() => queryClient.clear());
  const getViewIssues = t.mock.method(WorkspaceService.prototype, "getViewIssues", async () => response);
  const input = read("all-issues", "alice");
  input.request = { ...input.request, isReadCurrent: () => false };
  await assert.rejects(queryClient.fetchQuery(issueListOptions(input)), /access or list preferences changed/);
  assert.equal(getViewIssues.mock.callCount(), 0);
});

await test("workspace layout request params preserve the table projection defaults", () => {
  assert.equal(getWorkspaceIssueRequestParams(workspaceLayoutInput({}), pagination).include_properties, true);
  assert.equal(
    getWorkspaceIssueRequestParams(workspaceLayoutInput({ layout: "spreadsheet" }), pagination, "next")
      .include_properties,
    true
  );
  assert.equal(
    getWorkspaceIssueRequestParams(workspaceLayoutInput({ layout: "list" }), pagination).include_properties,
    undefined
  );
});
