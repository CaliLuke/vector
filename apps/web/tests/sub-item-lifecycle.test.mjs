import assert from "node:assert/strict";
import test from "node:test";
import {
  React,
  act,
  createPermissionUiFixture,
  settleQuery,
  registerPermissionUiStubs,
} from "../../../tests/frontend/helpers/permission-ui-fixture.mjs";
import { registerWebTestModules } from "../../../tests/frontend/helpers/web-test-modules.mjs";

globalThis.React = React;
globalThis.lifecycleToasts = [];
globalThis.renderDeleteConfirmation = (props) =>
  React.createElement(
    "button",
    { type: "button", "data-confirm-delete": true, disabled: props.isSubmitting, onClick: props.handleSubmit },
    "Delete child"
  );
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/toast":
    "export const TOAST_TYPE={ERROR:'error',SUCCESS:'success'};export const setToast=value=>globalThis.lifecycleToasts.push(value);",
  "@/providers/issue-detail-ui-state": "export const useIssueDetailUiState=()=>globalThis.lifecycleUi;",
  "@/hooks/use-issue-link-operations": "export const useLinkOperations=()=>({});",
  "@/hooks/use-issue-resources": "export const useIssueRelations=()=>({create:async()=>{}});",
  "../issue-detail/links/create-update-link-modal": "export const IssueLinkCreateUpdateModal=()=>null;",
  "../issue-modal/modal":
    "export const CreateUpdateIssueModal=props=>{globalThis.createdChildModal=props;return null;};",
  "../../issue-modal/modal": "export const CreateUpdateIssueModal=()=>null;",
  "@/components/core/modals/existing-issues-list-modal":
    "export const ExistingIssuesListModal=props=>{if(props.searchParams.sub_issue)globalThis.existingChildModal=props;return null;};",
  "./issues-list/root": "export const SubIssuesListRoot=props=>{globalThis.lifecycleList=props;return null;};",
});
registerPermissionUiStubs("/components/issues/delete-issue-modal.tsx", {
  "@plane/ui": "export const AlertModalCore=props=>globalThis.renderDeleteConfirmation(props);",
  "@/hooks/use-projects":
    "export const useWorkspaceProjectDetails=()=>({projects:[{id:'parent-project',identifier:'P'},{id:'child-project',identifier:'C'}]});",
});
const { IssueDetailWidgetModals } =
  await import("../core/components/issues/issue-detail-widgets/issue-detail-widget-modals.tsx");
const { SubIssuesCollapsibleContent } =
  await import("../core/components/issues/issue-detail-widgets/sub-issues/content.tsx");
const { useWorkItemSubItems } = await import("../core/hooks/use-work-item-detail.ts");
const { IssueService } = await import("../core/services/issue/index.ts");
const identity = await import("../core/queries/identity.ts");
const { EIssueServiceType } = await import("@plane/types");
const child = {
  id: "child",
  parent_id: "parent",
  project_id: "child-project",
  created_by: "alice",
  name: "Child",
  sequence_id: 3,
};
function deferred() {
  let resolve, reject;
  const promise = new Promise((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}
function WidgetProbe() {
  useWorkItemSubItems("acme", "parent-project", "parent");
  return React.createElement(IssueDetailWidgetModals, {
    workspaceSlug: "acme",
    projectId: "parent-project",
    issueId: "parent",
    issueServiceType: EIssueServiceType.ISSUES,
  });
}
async function openDelete() {
  await act(() => globalThis.lifecycleList.handleIssueCrudState("delete", "parent", child));
  await settleQuery();
}

async function setup(t, { mode = "create", remove = async () => {} } = {}) {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/parent-project/issues/P-1",
    workspaceRole: 15,
    projectRoles: { "parent-project": 20, "child-project": 20 },
  });
  const writes = [],
    reads = [];
  globalThis.lifecycleToasts = [];
  globalThis.createdChildModal = undefined;
  globalThis.existingChildModal = undefined;
  globalThis.lifecycleUi = {
    isCreateIssueModalOpen: mode === "create",
    isSubIssuesModalOpen: mode === "existing",
    issueCrudOperationState: {
      create: { toggle: mode === "create", parentIssueId: "parent" },
      existing: { toggle: mode === "existing", parentIssueId: "parent" },
    },
    toggleCreateIssueModal: () => {},
    toggleSubIssuesModal: () => {},
    setIssueCrudOperationState: () => {},
    toggleDeleteIssueModal: () => {},
    toggleIssueLinkModal: () => {},
    setIssueLinkData: () => {},
    setLastWidgetAction: () => {},
    setRelationKey: () => {},
    toggleRelationModal: () => {},
  };
  t.mock.method(IssueService.prototype, "subIssues", async (_workspace, project, parent) => {
    reads.push([project, parent]);
    return { sub_issues: [child] };
  });
  t.mock.method(IssueService.prototype, "addSubIssues", async (...args) => {
    writes.push(["add", ...args]);
    return { sub_issues: [child] };
  });
  t.mock.method(IssueService.prototype, "deleteIssue", async (...args) => {
    writes.push(["delete", ...args]);
    await remove(...args);
  });
  await fixture.render(
    mode === "delete"
      ? React.createElement(SubIssuesCollapsibleContent, {
          workspaceSlug: "acme",
          projectId: "parent-project",
          parentIssueId: "parent",
          disabled: false,
        })
      : React.createElement(WidgetProbe)
  );
  await settleQuery();
  const deleteButton = () => fixture.environment.container.querySelector("[data-confirm-delete]");
  const confirmDelete = async () => {
    await act(() => deleteButton().click());
    await settleQuery();
  };
  return { ...fixture, writes, reads, openDelete, deleteButton, confirmDelete };
}
await test("acknowledging a created child refreshes its parent without another relationship write", async (t) => {
  const view = await setup(t);
  assert.equal(globalThis.createdChildModal.data.parent_id, "parent");
  await act(() => globalThis.createdChildModal.onSubmit(child));
  await settleQuery();
  assert.deepEqual(view.writes, []);
  assert.deepEqual(view.reads, [
    ["parent-project", "parent"],
    ["parent-project", "parent"],
  ]);
  assert.deepEqual(globalThis.lifecycleToasts, []);
});
await test("a created child remains acknowledged when its old session has retired", async (t) => {
  const view = await setup(t);
  const acknowledge = globalThis.createdChildModal.onSubmit;
  await act(() => identity.advanceIdentitySession(view.client));
  await act(() => acknowledge(child));
  await settleQuery();
  assert.deepEqual(view.writes, []);
  assert.deepEqual(globalThis.lifecycleToasts, []);
});
await test("adding an existing child still writes the relationship", async (t) => {
  const view = await setup(t, { mode: "existing" });
  await act(() => globalThis.existingChildModal.handleOnSubmit([child]));
  await settleQuery();
  assert.deepEqual(view.writes, [["add", "acme", "parent-project", "parent", { sub_issue_ids: ["child"] }]]);
  assert.equal(globalThis.lifecycleToasts.length, 1);
  assert.equal(globalThis.lifecycleToasts[0].type, "success");
});
await test("deleting a cross-project child uses its own project and refreshes its parent", async (t) => {
  const view = await setup(t, { mode: "delete" });
  await view.openDelete();
  await view.confirmDelete();
  assert.deepEqual(view.writes, [["delete", "acme", "child-project", "child"]]);
  assert.deepEqual(view.reads, [
    ["parent-project", "parent"],
    ["parent-project", "parent"],
  ]);
  assert.equal(globalThis.lifecycleToasts.length, 1);
  assert.equal(globalThis.lifecycleToasts[0].type, "success");
  assert.equal(view.deleteButton(), null);
});
await test("a failed child deletion shows one error and keeps the confirmation open for retry", async (t) => {
  let attempt = 0;
  const view = await setup(t, {
    mode: "delete",
    remove: async () => {
      if (++attempt === 1) throw new Error("Rejected");
    },
  });
  await view.openDelete();
  await view.confirmDelete();
  assert.equal(globalThis.lifecycleToasts.length, 1);
  assert.equal(globalThis.lifecycleToasts[0].type, "error");
  assert.equal(view.deleteButton().disabled, false);
  await view.confirmDelete();
  assert.equal(view.writes.length, 2);
  assert.deepEqual(
    globalThis.lifecycleToasts.map((toast) => toast.type),
    ["error", "success"]
  );
  assert.equal(view.deleteButton(), null);
});
await test("a late deletion failure cannot notify a replacement session", async (t) => {
  const pending = deferred();
  const view = await setup(t, { mode: "delete", remove: () => pending.promise });
  await view.openDelete();
  await view.confirmDelete();
  await act(() => identity.advanceIdentitySession(view.client));
  await settleQuery();
  await act(() => pending.reject(new Error("Retired request")));
  await settleQuery();
  assert.deepEqual(globalThis.lifecycleToasts, []);
});
