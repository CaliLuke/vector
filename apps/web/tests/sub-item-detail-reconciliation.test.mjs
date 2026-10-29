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
function Probe({ onOperations }) {
  onOperations(useSubIssueOperations("acme", "p1", "parent", EIssueServiceType.ISSUES));
  const detail = useWorkItemDetail("acme", "p2", "child");
  const children = useWorkItemSubItems("acme", "p1", "parent");
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

async function verifyReconciliation(t, change, expected) {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/P-1",
    workspaceRole: 15,
    projectRoles: { p1: 20, p2: 20 },
  });
  let row = { id: "child", project_id: "p2", parent_id: "parent", priority: "low" };
  let operations;
  t.mock.method(IssueService.prototype, "retrieve", async () => row);
  t.mock.method(IssueService.prototype, "subIssues", async () => ({
    sub_issues: row.parent_id === "parent" ? [row] : [],
  }));
  t.mock.method(
    IssueService.prototype,
    "patchIssue",
    async (_workspace, _project, _id, data) => (row = { ...row, ...data })
  );
  await fixture.render(
    React.createElement(Probe, {
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
