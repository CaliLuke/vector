import assert from "node:assert/strict";
import test from "node:test";
import {
  React,
  act,
  createPermissionUiFixture,
  settleQuery,
} from "../../../tests/frontend/helpers/permission-ui-fixture.mjs";
import { registerWebTestModules } from "../../../tests/frontend/helpers/web-test-modules.mjs";
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/toast": "export const TOAST_TYPE={};export const setToast=()=>{};",
});
const { useSubIssueOperations } = await import("../core/components/issues/issue-detail-widgets/sub-issues/helper.ts");
const { useWorkItemDetail, useWorkItemSubItems } = await import("../core/hooks/use-work-item-detail.ts");
const { IssueService } = await import("../core/services/issue/index.ts");
const { EIssueServiceType } = await import("@plane/types");
function Probe({ onOperations, parentService }) {
  onOperations(useSubIssueOperations("acme", "p1", "parent", parentService));
  const detail = useWorkItemDetail("acme", "p2", "child");
  const children = useWorkItemSubItems("acme", "p1", "parent", parentService);
  return React.createElement(
    "output",
    null,
    JSON.stringify({
      priority: detail.issue?.priority,
      parent: detail.issue?.parent_id,
      children: children.subItems?.map((item) => item.id),
    })
  );
}

async function verifyReconciliation(t, change, expected, parentService = EIssueServiceType.ISSUES, patchStatus = 200) {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/P-1",
    workspaceRole: 15,
    projectRoles: { p1: 20, p2: 20 },
  });
  let row = { id: "child", project_id: "p2", parent_id: "parent", priority: "low" };
  let operations;
  const writes = [];
  const childPath = "/api/workspaces/acme/projects/p2/issues/child/";
  const parentPath = `/api/workspaces/acme/projects/p1/${parentService}/parent/${parentService === EIssueServiceType.EPICS ? "issues" : "sub-issues"}/`;
  t.mock.method(IssueService.prototype, "get", async (path) => {
    if (path === childPath && row) return { status: 200, data: row };
    if (path === parentPath) return { status: 200, data: { sub_issues: row?.parent_id === "parent" ? [row] : [] } };
    throw { response: { data: { status: 404 } } };
  });
  t.mock.method(IssueService.prototype, "patch", async (path, data) => {
    writes.push(path);
    row = { ...row, ...data };
    return patchStatus === 204 ? { status: 204 } : { status: 200, data: row };
  });
  t.mock.method(IssueService.prototype, "delete", async (path) => {
    writes.push(path);
    row = undefined;
    return { status: 204 };
  });
  await fixture.render(
    React.createElement(Probe, {
      parentService,
      onOperations: (value) => {
        operations = value;
      },
    })
  );
  await settleQuery();
  const state = () => JSON.parse(fixture.environment.container.querySelector("output").textContent);
  assert.deepEqual(state(), { priority: "low", parent: "parent", children: ["child"] });
  await act(() => change(operations));
  await settleQuery();
  assert.deepEqual(writes, [childPath]);
  assert.deepEqual(state(), expected);
}

await test("cross-project child detail and parent list agree after a property edit", (t) =>
  verifyReconciliation(
    t,
    (operations) => operations.updateSubIssue("acme", "p2", "parent", "child", { priority: "urgent" }),
    { priority: "urgent", parent: "parent", children: ["child"] }
  ));
await test("cross-project child detail and parent list agree after parent removal", (t) =>
  verifyReconciliation(t, (operations) => operations.removeSubIssue("acme", "p2", "parent", "child"), {
    priority: "low",
    parent: null,
    children: [],
  }));

await test("ordinary child edits under an epic use the child endpoint and reconcile its detail", (t) =>
  verifyReconciliation(
    t,
    (operations) => operations.updateSubIssue("acme", "p2", "parent", "child", { priority: "urgent" }),
    { priority: "urgent", parent: "parent", children: ["child"] },
    EIssueServiceType.EPICS
  ));
await test("ordinary child removal from an epic uses the child endpoint and refreshes its epic parent", (t) =>
  verifyReconciliation(
    t,
    (operations) => operations.removeSubIssue("acme", "p2", "parent", "child"),
    { priority: "low", parent: null, children: [] },
    EIssueServiceType.EPICS
  ));
await test("ordinary child deletion under an epic uses the child endpoint", (t) =>
  verifyReconciliation(
    t,
    (operations) => operations.deleteSubIssue("acme", "p2", "parent", "child"),
    { children: [] },
    EIssueServiceType.EPICS
  ));
await test("ordinary parent child deletion clears the child's detail and parent list", (t) =>
  verifyReconciliation(t, (operations) => operations.deleteSubIssue("acme", "p2", "parent", "child"), {
    children: [],
  }));

for (const parentService of [EIssueServiceType.ISSUES, EIssueServiceType.EPICS])
  void test(`an accepted child edit without a response body refreshes detail under ${parentService}`, (t) =>
    verifyReconciliation(
      t,
      (operations) => operations.updateSubIssue("acme", "p2", "parent", "child", { priority: "urgent" }),
      { priority: "urgent", parent: "parent", children: ["child"] },
      parentService,
      204
    ));
