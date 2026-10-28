import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient } from "@tanstack/react-query";
import { pathToFileURL } from "node:url";
import path from "node:path";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { IssueService } = await fromRoot("apps/web/core/services/issue/index.ts");
const { identityKeys } = await fromRoot("apps/web/core/queries/identity.ts");
const { permissionKeys } = await fromRoot("apps/web/core/queries/permissions.ts");
const { labelKeys } = await import("../../apps/web/core/queries/labels.ts");
const { issueListOptions } = await import("../../apps/web/core/queries/issue-lists.ts");
const { processIssueResponse } = await import("../../apps/web/helpers/issue-list/response.ts");
const { projectGroupedIssueIdsByDisplayOrder, projectIssueIdsByDisplayOrder } =
  await import("../../apps/web/core/components/issues/issue-layouts/issue-display-ordering.ts");

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
const labelA = { id: "label-a", name: "Alpha", project_id: "p1", workspace_id: "w1", sort_order: 20 };
const labelB = { id: "label-b", name: "Zulu", project_id: "p1", workspace_id: "w1", sort_order: 10 };
const issueA = { id: "issue-a", project_id: "p1", label_ids: [labelA.id], priority: "low" };
const issueB = { id: "issue-b", project_id: "p1", label_ids: [labelB.id], priority: "low" };
function client() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  queryClient.setQueryData(identityKeys.currentUser(), { id: scope.accountId });
  queryClient.setQueryData(permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug), {
    is_active: true,
    role: scope.workspaceRole,
    updated_at: scope.membershipRevision,
  });
  queryClient.setQueryData(
    permissionKeys.workspaceProjectRoles(
      scope.accountId,
      scope.workspaceSlug,
      scope.workspaceRole,
      scope.membershipRevision
    ),
    { p1: 20 }
  );
  return queryClient;
}
const labelsScope = {
  accountId: scope.accountId,
  workspaceSlug: scope.workspaceSlug,
  projectId: scope.projectId,
  workspaceRole: scope.workspaceRole,
  membershipRevision: scope.membershipRevision,
  projectRole: scope.projectRole,
  projectMembershipRole: scope.projectMembershipRole,
  sessionGeneration: scope.sessionGeneration,
};
function read() {
  const input = {
    workspaceSlug: scope.workspaceSlug,
    projectId: scope.projectId,
    richFilters: {},
    displayFilters: { layout: "spreadsheet", group_by: "priority", order_by: "labels__name" },
  };
  return {
    scope,
    pagination: { canGroup: true, perPageCount: 20, groupedBy: "priority" },
    request: { family: "project", input, isReadCurrent: () => true },
  };
}
function labelsById(queryClient) {
  return new Map((queryClient.getQueryData(labelKeys.project(labelsScope)) ?? []).map((label) => [label.id, label]));
}

await test("native list rows reproject when the scoped label Query is renamed or cleared", async (t) => {
  const queryClient = client();
  t.after(() => queryClient.clear());
  queryClient.setQueryData(labelKeys.project(labelsScope), [labelA, labelB]);
  let issueListReads = 0;
  t.mock.method(IssueService.prototype, "getIssues", async () => {
    issueListReads += 1;
    return {
      grouped_by: "priority",
      total_count: 2,
      count: 2,
      total_pages: 1,
      total_results: 2,
      results: { low: { results: [issueB, issueA], total_results: 2 } },
    };
  });
  const readInput = read();
  const result = await queryClient.fetchQuery(issueListOptions(readInput));
  const processed = processIssueResponse(result.response);
  const select = () =>
    projectGroupedIssueIdsByDisplayOrder(
      processed.groupedIssues,
      (id) => Object.fromEntries(processed.issueList.map((issue) => [issue.id, issue]))[id],
      undefined,
      "labels__name",
      labelsById(queryClient)
    );

  assert.deepEqual(select().low, ["issue-a", "issue-b"]);
  queryClient.setQueryData(labelKeys.project(labelsScope), [
    { ...labelA, name: "Zulu", sort_order: 1 },
    { ...labelB, name: "Alpha", sort_order: 2 },
  ]);
  assert.deepEqual(select().low, ["issue-b", "issue-a"]);
  queryClient.setQueryData(labelKeys.project(labelsScope), []);
  assert.deepEqual(select().low, ["issue-b", "issue-a"]);
  assert.equal(issueListReads, 1, "catalogue replacement reprojects the captured Query rows without another list read");
});

await test("native workspace rows sort by current label names and preserve Query order when labels disappear", async () => {
  const rows = [issueB, issueA];
  const byId = Object.fromEntries(rows.map((issue) => [issue.id, issue]));
  let labels = new Map([
    [labelA.id, labelA],
    [labelB.id, labelB],
  ]);
  assert.deepEqual(
    projectIssueIdsByDisplayOrder(
      rows.map((issue) => issue.id),
      (id) => byId[id],
      undefined,
      "labels__name",
      labels
    ),
    ["issue-a", "issue-b"]
  );
  labels = new Map([
    [labelA.id, { ...labelA, name: "Zulu" }],
    [labelB.id, { ...labelB, name: "Alpha" }],
  ]);
  assert.deepEqual(
    projectIssueIdsByDisplayOrder(
      rows.map((issue) => issue.id),
      (id) => byId[id],
      undefined,
      "labels__name",
      labels
    ),
    ["issue-b", "issue-a"]
  );
  labels = new Map();
  assert.deepEqual(
    projectIssueIdsByDisplayOrder(
      rows.map((issue) => issue.id),
      (id) => byId[id],
      undefined,
      "labels__name",
      labels
    ),
    ["issue-b", "issue-a"]
  );
});
