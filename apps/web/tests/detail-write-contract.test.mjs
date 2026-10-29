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
import { webRequire } from "../../../tests/frontend/helpers/react.mjs";
const { EIssuesStoreType } = webRequire("@plane/types");
globalThis.React = React;
globalThis.detailWriteToasts = [];
globalThis.detailWriteControls = new Map();
const dropdown = (kind, name) =>
  `export const ${name}=props=>globalThis.renderDetailWriteControl(${JSON.stringify(kind)},props);`;
globalThis.renderDetailWriteControl = (kind, props) => {
  globalThis.detailWriteControls.set(kind, props);
  return React.createElement(
    "button",
    {
      type: "button",
      "data-property": kind,
      "data-value": JSON.stringify(props.value),
      disabled: props.disabled,
      onClick: () =>
        props.onChange(kind === "assignee" ? ["bob"] : kind === "date" ? new Date("2026-10-30") : "urgent"),
    },
    kind
  );
};
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/toast":
    "export const TOAST_TYPE={ERROR:'error',SUCCESS:'success'};export const setToast=t=>globalThis.detailWriteToasts.push(t);export const setPromiseToast=()=>{};",
  "@/app/assets/empty-state/issue.svg?url": "export default '';",
  "@/components/common/empty-state": "export const EmptyState=()=>null;",
  "@/providers/sidebar-state": "export const useIssueDetailSidebarCollapsed=()=>false;",
  "@/providers/issue-detail-ui-state":
    "export const useIssueDetailUiState=()=>({peekIssue:globalThis.detailWritePeek,setPeekIssue:()=>{},isParentIssueModalOpen:'item',toggleParentIssueModal:()=>{}});export const useWorkItemPeek=()=>({});",
  "@/hooks/use-issue-list-type": `export const useIssueListType=()=>${JSON.stringify(EIssuesStoreType.PROJECT)};`,
  "@/hooks/use-issue-properties": "export const useWorkItemProperties=()=>{};",
  "@/hooks/use-projects":
    "export const useWorkspaceProjectDetails=()=>({projects:[{id:'p1',identifier:'P1',estimate:'estimate'}]});",
  "@/hooks/use-project-states":
    "export const useProjectStates=()=>({getStateById:()=>({group:'started'})});export const useWorkspaceStates=()=>({getStateById:()=>({group:'started'}),getProjectStates:()=>[]});",
  "@/hooks/use-member-rosters": "export const useWorkspaceMemberRoster=()=>({memberById:()=>undefined});",
  "@/hooks/use-estimates":
    "export const useProjectEstimates=()=>({isEnabled:true});export const useWorkspaceEstimates=()=>({});",
  "@/hooks/use-platform-os": "export const usePlatformOS=()=>({isMobile:false});",
  "@/components/work-item-types/item-custom-properties": "export const ItemCustomProperties=()=>null;",
  "@/components/dropdowns/state/dropdown": dropdown("state", "StateDropdown"),
  "@/components/dropdowns/member/dropdown": dropdown("assignee", "MemberDropdown"),
  "@/components/dropdowns/priority": dropdown("priority", "PriorityDropdown"),
  "@/components/dropdowns/date": dropdown("date", "DateDropdown"),
  "@/components/dropdowns/estimate": dropdown("estimate", "EstimateDropdown"),
  "@/components/common/layout/sidebar/property-list-item":
    "export const SidebarPropertyListItem=({children})=>children;",
});
registerPermissionUiStubs("/components/issues/issue-detail/root.tsx", {
  "./main-content":
    "export const IssueMainContent=props=>{globalThis.detailWriteOperations=props.issueOperations;return globalThis.renderDetailWriteMain(props);};",
  "../peek-overview": "export const IssuePeekOverview=()=>null;",
});
registerPermissionUiStubs("/components/issues/peek-overview/root.tsx", {
  "./view":
    "export const IssueView=props=>{globalThis.detailWriteOperations=props.issueOperations;return globalThis.renderDetailWritePeek(props);};",
});
registerPermissionUiStubs("/components/issues/issue-detail/sidebar.tsx", {
  "./label": "export const IssueLabel=()=>null;",
});
registerPermissionUiStubs("/components/issues/peek-overview/properties.tsx", {
  "../issue-detail/label": "export const IssueLabel=()=>null;",
});
registerPermissionUiStubs("/components/issues/issue-detail/project-properties.tsx", {
  "./cycle-select": "export const IssueCycleSelect=()=>null;",
  "./module-select": "export const IssueModuleSelect=()=>null;",
});
registerPermissionUiStubs("/components/issues/issue-detail/parent-select.tsx", {
  "../work-item-selection-modal":
    "export const WorkItemSelectionModal=props=>{globalThis.detailWriteSelection=props;return null;};",
  "@/components/issues/issue-detail/issue-identifier": "export const IssueIdentifier=()=>null;",
});
for (const parent of ["use-cycles", "use-modules", "use-workspace-issue-properties"])
  registerPermissionUiStubs(`/hooks/${parent}.ts`, {
    "./use-estimates":
      "export const useProjectEstimates=()=>({isEnabled:true});export const useWorkspaceEstimates=()=>({});",
  });
registerPermissionUiStubs("/components/issues/issue-detail/parent/root.tsx", {
  "@plane/ui":
    "const React=globalThis.React;export const ControlLink=({children})=>React.createElement('span',null,children);export const CustomMenu=Object.assign(({children})=>React.createElement('div',null,children),{MenuItem:({children,...props})=>React.createElement('button',props,children)});",
  "./siblings": "export const IssueParentSiblings=()=>null;",
  "@/components/issues/issue-detail/issue-identifier": "export const IssueIdentifier=()=>null;",
});
const { IssueDetailRoot } = await import("../core/components/issues/issue-detail/root.tsx");
const { IssuePeekOverview } = await import("../core/components/issues/peek-overview/root.tsx");
const { PeekOverviewProperties } = await import("../core/components/issues/peek-overview/properties.tsx");
const { IssueParentDetail } = await import("../core/components/issues/issue-detail/parent/root.tsx");
const { useWorkItemDetailRecord } = await import("../core/providers/work-item-detail-record.tsx");
function ParentSummary(props) {
  const record = useWorkItemDetailRecord();
  return record?.issue
    ? React.createElement(IssueParentDetail, {
        ...props,
        issue: record.issue,
        disabled: !!props.disabled || props.isEditable === false || !!props.isArchived || !!props.is_archived,
      })
    : null;
}
const { IssueService } = await import("../core/services/issue/index.ts");
const identity = await import("../core/queries/identity.ts");
globalThis.renderDetailWriteMain = () => null;
globalThis.renderDetailWritePeek = (props) => React.createElement(PeekOverviewProperties, props);
async function click(button) {
  assert.ok(button, "expected visible property");
  await act(async () => {
    button.click();
    await Promise.resolve();
  });
  await settleQuery();
}
async function setup(t, mode = "detail", options = {}) {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/P1-1",
    workspaceRole: 15,
    projectRoles: { p1: options.role ?? 20, p2: 20 },
  });
  let row = {
    id: "item",
    project_id: "p1",
    name: "Original",
    created_by: "alice",
    priority: "low",
    state_id: "started",
    assignee_ids: [],
    label_ids: [],
    module_ids: [],
    start_date: null,
    target_date: null,
    estimate_point: "small",
    parent_id: null,
  };
  if (options.parent)
    row = { ...row, parent_id: "parent", parent: { id: "parent", project_id: "p1", sequence_id: 2, name: "Parent" } };
  globalThis.renderDetailWriteMain = options.parent ? (props) => React.createElement(ParentSummary, props) : () => null;
  globalThis.renderDetailWritePeek = (props) =>
    React.createElement(
      React.Fragment,
      null,
      options.parent ? React.createElement(ParentSummary, props) : null,
      React.createElement(PeekOverviewProperties, { ...props, disabled: props.disabled || props.is_archived })
    );
  const transport = { patch: async () => {} };
  const writes = [];
  globalThis.detailWriteToasts = [];
  globalThis.detailWriteControls.clear();
  globalThis.detailWriteOperations = undefined;
  globalThis.detailWritePeek = {
    workspaceSlug: "acme",
    projectId: "p1",
    issueId: "item",
    isArchived: !!options.archived,
  };
  t.mock.method(IssueService.prototype, "retrieve", async () => row);
  t.mock.method(IssueService.prototype, "patchIssue", async (workspace, project, id, data) => {
    writes.push([workspace, project, id, data]);
    await transport.patch();
    row = { ...row, ...data };
    return row;
  });
  const Owner = mode === "detail" ? IssueDetailRoot : IssuePeekOverview;
  const render = () =>
    fixture.render(
      React.createElement(Owner, {
        workspaceSlug: "acme",
        projectId: "p1",
        issueId: "item",
        storeType: EIssuesStoreType.PROJECT,
        is_archived: !!options.archived,
      })
    );
  await render();
  await settleQuery();
  assert.ok(globalThis.detailWriteOperations);
  return {
    ...fixture,
    transport,
    writes,
    render,
    click,
    row: () => row,
    property: (name) => fixture.environment.container.querySelector(`[data-property="${name}"]`),
  };
}
async function rejectsWrite(t, mode) {
  const view = await setup(t, mode);
  view.transport.patch = async () => {
    throw new Error("write rejected");
  };
  await act(async () => {
    await assert.rejects(
      globalThis.detailWriteOperations.update("acme", "p1", "item", { priority: "urgent" }),
      /write rejected/
    );
  });
  assert.equal(view.writes.length, 1);
  assert.equal(view.row().priority, "low");
  assert.deepEqual(globalThis.detailWriteToasts, []);
}
await test("detail writes reject failures so the submitting form can retain its draft", (t) =>
  rejectsWrite(t, "detail"));
await test("preview writes reject failures so the submitting form can retain its draft", (t) =>
  rejectsWrite(t, "peek"));
async function rejectsParentSelection(t, mode) {
  const view = await setup(t, mode);
  view.transport.patch = async () => {
    throw new Error("write rejected");
  };
  await act(async () => {
    await assert.rejects(
      globalThis.detailWriteSelection.onSelect({ id: "parent", project_id: "p2" }),
      /write rejected/
    );
  });
  assert.equal(view.row().parent_id, null);
  assert.equal(view.writes.length, 1);
  assert.deepEqual(globalThis.detailWriteToasts, []);
}
await test("detail parent selection receives the rejected write", (t) => rejectsParentSelection(t, "detail"));
await test("preview parent selection receives the rejected write", (t) => rejectsParentSelection(t, "peek"));
async function pendingProperty(t, mode) {
  const view = await setup(t, mode);
  const pending = Promise.withResolvers();
  view.transport.patch = () => pending.promise;
  const button = view.property("priority");
  await act(async () => {
    button.click();
    button.click();
    await Promise.resolve();
  });
  await settleQuery();
  const writes = view.writes.length;
  const disabled = view.property("priority").disabled;
  await act(() => pending.reject(new Error("write rejected")));
  await settleQuery();
  assert.equal(writes, 1);
  assert.equal(disabled, true);
  assert.equal(view.property("priority").disabled, false);
  assert.equal(globalThis.detailWriteToasts.length, 1);
  assert.equal(globalThis.detailWriteToasts[0].type, "error");
  view.transport.patch = async () => {};
  await view.click(view.property("priority"));
  assert.equal(view.writes.length, 2);
  assert.equal(view.property("priority").getAttribute("data-value"), '"urgent"');
}
await test("detail property edits prevent duplicate writes and recover after failure", (t) =>
  pendingProperty(t, "detail"));
await test("preview property edits prevent duplicate writes and recover after failure", (t) =>
  pendingProperty(t, "peek"));
async function lateFailure(t, mode) {
  const view = await setup(t, mode);
  const pending = Promise.withResolvers();
  view.transport.patch = () => pending.promise;
  await view.click(view.property("estimate"));
  await act(() => identity.advanceIdentitySession(view.client));
  await settleQuery();
  await act(() => pending.reject(new Error("old failure")));
  await settleQuery();
  assert.equal(view.writes.length, 1);
  assert.deepEqual(globalThis.detailWriteToasts, []);
}
await test("late detail estimate failures cannot notify a replacement session", (t) => lateFailure(t, "detail"));
await test("late preview estimate failures cannot notify a replacement session", (t) => lateFailure(t, "peek"));
await test("a detail write cannot silently target the route workspace instead of its requested workspace", async (t) => {
  const view = await setup(t);
  await act(async () => {
    await assert.rejects(
      globalThis.detailWriteOperations.update("other-workspace", "p1", "item", { priority: "urgent" })
    );
  });
  assert.deepEqual(view.writes, []);
});

for (const mode of ["detail", "peek"]) {
  void test(`${mode} inline state, assignee, date and estimate failures stay handled`, async (t) => {
    const view = await setup(t, mode);
    view.transport.patch = async () => {
      throw new Error("write rejected");
    };
    await view.click(view.property("state"));
    await view.click(view.property("assignee"));
    await view.click(view.property("date"));
    await view.click(view.property("estimate"));
    assert.equal(view.writes.length, 4);
    assert.equal(globalThis.detailWriteToasts.length, 4);
    assert.deepEqual(view.row().assignee_ids, []);
    assert.equal(view.row().start_date, null);
    assert.equal(view.row().estimate_point, "small");
  });
  void test(`${mode} estimate edits share the pending guard with basic properties`, async (t) => {
    const view = await setup(t, mode);
    const pending = Promise.withResolvers();
    view.transport.patch = () => pending.promise;
    const estimate = globalThis.detailWriteControls.get("estimate").onChange;
    await view.click(view.property("priority"));
    const estimateDisabled = view.property("estimate").disabled;
    await act(() => estimate("large"));
    await act(() => pending.resolve());
    await settleQuery();
    assert.equal(estimateDisabled, true);
    assert.equal(view.writes.length, 1);
    assert.equal(view.row().priority, "urgent");
    assert.equal(view.row().estimate_point, "small");
    assert.deepEqual(globalThis.detailWriteToasts, []);
  });
  void test(`${mode} captured writes reject after the account session changes`, async (t) => {
    const view = await setup(t, mode);
    const update = globalThis.detailWriteOperations.update;
    await act(() => identity.advanceIdentitySession(view.client));
    await settleQuery();
    await assert.rejects(update("acme", "p1", "item", { priority: "urgent" }));
    assert.deepEqual(view.writes, []);
  });
  void test(`${mode} archived items reject direct edits and disable property controls`, async (t) => {
    const view = await setup(t, mode, { archived: true });
    assert.equal(view.property("priority").disabled, true);
    await assert.rejects(globalThis.detailWriteOperations.update("acme", "p1", "item", { priority: "urgent" }));
    assert.deepEqual(view.writes, []);
  });
  void test(`${mode} parent-summary unlink handles rejection and prevents repeat submissions`, async (t) => {
    const view = await setup(t, mode, { parent: true });
    const pending = Promise.withResolvers();
    view.transport.patch = () => pending.promise;
    const button = [...view.environment.container.querySelectorAll("button")].find(
      (b) => b.textContent === "issue.remove.parent.label"
    );
    assert.ok(button);
    await act(() => {
      button.click();
      button.click();
    });
    await settleQuery();
    const writes = view.writes.length;
    const disabled = button.disabled;
    await act(() => pending.reject(new Error("write rejected")));
    await settleQuery();
    assert.equal(writes, 1);
    assert.equal(disabled, true);
    assert.equal(button.disabled, false);
    assert.equal(view.row().parent_id, "parent");
    assert.equal(globalThis.detailWriteToasts.length, 1);
  });
  void test(`${mode} readonly parent-summary unlink cannot write`, async (t) => {
    const view = await setup(t, mode, { parent: true, role: 5 });
    const button = [...view.environment.container.querySelectorAll("button")].find(
      (b) => b.textContent === "issue.remove.parent.label"
    );
    assert.ok(button);
    assert.equal(button.disabled, true);
    await view.click(button);
    assert.deepEqual(view.writes, []);
  });
}

for (const mode of ["detail", "peek"])
  void test(`${mode} a late accepted write cannot invalidate a replacement session's schema`, async (t) => {
    const view = await setup(t, mode);
    const pending = Promise.withResolvers();
    view.transport.patch = () => pending.promise;
    let write;
    await act(() => {
      write = globalThis.detailWriteOperations.update("acme", "p1", "item", { priority: "urgent" });
    });
    await act(() => identity.advanceIdentitySession(view.client));
    await settleQuery();
    const key = ["workspace", "acme", "work-item-schema", "replacement", { issue_id: "item" }, "detail"];
    view.client.setQueryData(key, { properties: [] });
    await act(async () => {
      pending.resolve();
      await write;
    });
    await settleQuery();
    assert.equal(view.client.getQueryState(key).isInvalidated, false);
    assert.deepEqual(globalThis.detailWriteToasts, []);
  });
