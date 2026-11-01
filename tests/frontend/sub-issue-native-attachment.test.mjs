import assert from "node:assert/strict";
import test from "node:test";
import { React, act, createPermissionUiFixture } from "./helpers/permission-ui-fixture.mjs";
const { EIssueServiceType } = await import("../../packages/types/dist/index.mjs");
const { useSubIssueOperations } =
  await import("../../apps/web/core/components/issues/issue-detail-widgets/sub-issues/helper.ts");
const { useWorkItemDetailScope } = await import("../../apps/web/core/hooks/use-work-item-detail.ts");
const { IssueService } = await import("../../apps/web/core/services/issue/index.ts");
const { issueDetailKeys } = await import("../../apps/web/core/queries/issue-details.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { itemAccessOptions } = await import("../../apps/web/core/queries/work-item-access.ts");

await test("native nested attachments refresh only their parent and reject denied target access", async (t) => {
  const view = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/P1-1",
    workspaceRole: 15,
    projectRoles: { p1: 20, p2: 20 },
  });
  let current;
  function Probe({ projectId = "p1", parentId = "root" }) {
    const operations = useSubIssueOperations("acme", projectId, parentId, EIssueServiceType.ISSUES);
    const access = useWorkItemDetailScope("acme", projectId, EIssueServiceType.ISSUES, parentId);
    React.useLayoutEffect(() => {
      current = { operations, scope: access.scope };
    }, [operations, access.scope]);
    return null;
  }
  await view.render(React.createElement(Probe));
  const keys = new Map(["root", "nested", "sibling"].map((id) => [id, issueDetailKeys.subItems(current.scope, id)]));
  for (const key of keys.values()) view.client.setQueryData(key, { sub_issues: [] });
  const requests = [];
  const saved = { id: "child", project_id: "p1", parent_id: "nested", name: "Attached child" };
  t.mock.method(IssueService.prototype, "addSubIssues", async (workspace, project, parent, payload) => {
    requests.push({ workspace, project, parent, payload });
    return { sub_issues: [{ ...saved, parent_id: parent }] };
  });
  await act(async () => current.operations.addSubIssue("acme", "p1", "nested", [{ id: "child", project_id: "p1" }]));
  assert.deepEqual(requests, [
    { workspace: "acme", project: "p1", parent: "nested", payload: { sub_issue_ids: ["child"] } },
  ]);
  assert.equal(view.client.getQueryState(keys.get("nested")).isInvalidated, true);
  assert.equal(view.client.getQueryState(keys.get("root")).isInvalidated, false);
  assert.equal(view.client.getQueryState(keys.get("sibling")).isInvalidated, false);
  await assert.rejects(
    current.operations.addSubIssue("other", "p1", "nested", [{ id: "child", project_id: "p1" }]),
    /access changed/
  );
  const denial = itemAccessOptions({
    accountId: "alice",
    generation: 0,
    workspaceSlug: "acme",
    projectId: "p1",
    issueId: "nested",
    serviceType: EIssueServiceType.ISSUES,
  });
  view.client.setQueryData(denial.queryKey, true);
  await assert.rejects(
    current.operations.addSubIssue("acme", "p1", "nested", [{ id: "child", project_id: "p1" }]),
    /access changed/
  );
  assert.equal(requests.length, 1, "denied targets never issue another attachment request");
  await view.render(React.createElement(Probe, { projectId: "p2", parentId: "other-parent" }));
  const crossKey = issueDetailKeys.subItems(current.scope, "other-parent");
  view.client.setQueryData(crossKey, { sub_issues: [] });
  await act(async () =>
    current.operations.addSubIssue("acme", "p2", "other-parent", [{ id: "child", project_id: "p1" }])
  );
  assert.deepEqual(requests.at(-1), {
    workspace: "acme",
    project: "p2",
    parent: "other-parent",
    payload: { sub_issue_ids: ["child"] },
  });
  assert.equal(view.client.getQueryState(crossKey).isInvalidated, true);
  assert.equal(view.client.getQueryState(keys.get("root")).isInvalidated, false);
  const staleOperations = current.operations;
  view.client.setQueryData(
    permissionKeys.workspaceProjectRoles(
      current.scope.accountId,
      "acme",
      current.scope.membershipRole,
      current.scope.membershipRevision
    ),
    { p1: 20 }
  );
  await assert.rejects(
    staleOperations.addSubIssue("acme", "p2", "other-parent", [{ id: "child", project_id: "p1" }]),
    /access changed/
  );
  assert.equal(requests.length, 2, "a captured operation cannot reuse revoked project access");
});
