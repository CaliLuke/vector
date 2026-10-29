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
const { ALL_ISSUES } = webRequire("@plane/constants");
globalThis.React = React;
globalThis.nestedToasts = [];
const dropdown = (name, exported) =>
  `export const ${exported}=props=>globalThis.renderNestedDropdown(${JSON.stringify(name)},props);`;
globalThis.renderNestedDropdown = (name, props) =>
  React.createElement(
    "button",
    {
      type: "button",
      disabled: props.disabled,
      "data-property": name,
      "data-value": JSON.stringify(props.value),
      onClick: () => props.onChange?.("urgent"),
    },
    name
  );
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/toast":
    "export const TOAST_TYPE={ERROR:'error',SUCCESS:'success'};export const setToast=value=>globalThis.nestedToasts.push(value);",
  "@plane/propel/tooltip": "export const Tooltip=({children})=>children;",
  "@/hooks/use-project-states": "export const useProjectStates=()=>({getStateById:()=>({group:'started'})});",
  "@/hooks/use-platform-os": "export const usePlatformOS=()=>({isMobile:false});",
  "@/hooks/use-issue-peek-overview-redirection": "export default ()=>({handleRedirection:()=>{}});",
  "@/components/issues/issue-detail/issue-identifier": "export const IssueIdentifier=()=>null;",
  "@/providers/issue-detail-ui-state":
    "export const useIssueDetailUiState=()=>({toggleCreateIssueModal:()=>{},toggleDeleteIssueModal:()=>{},toggleParentIssueModal:()=>{}});",
  "@/components/dropdowns/priority": dropdown("priority", "PriorityDropdown"),
  "@/components/dropdowns/state/dropdown": dropdown("state", "StateDropdown"),
  "@/components/dropdowns/member/dropdown": dropdown("assignee", "MemberDropdown"),
  "@/components/dropdowns/date": dropdown("date", "DateDropdown"),
  "@/components/dropdowns/date-range": dropdown("range", "DateRangeDropdown"),
  "../../issue-modal/modal":
    "export const CreateUpdateIssueModal=props=>{globalThis.nestedEditModal=props;return null;};",
});
const ui = `const React=globalThis.React;
export const Collapsible=({children})=>React.createElement("section",null,children);
export const ControlLink=({children,...props})=>React.createElement("a",props,children);
export const CustomMenu=({children})=>React.createElement("div",null,children);
CustomMenu.MenuItem=({children,...props})=>React.createElement("button",{type:"button",...props},children);
export const AlertModalCore=props=>React.createElement("button",{type:"button","data-delete-confirm":true,disabled:props.isSubmitting,onClick:props.handleSubmit},"Confirm delete");`;
registerPermissionUiStubs("/sub-issues/issues-list/list-group.tsx", { "@plane/ui": ui });
const catalogue =
  "export const useWorkspaceProjectDetails=()=>({projects:[{id:'p1',identifier:'P1'},{id:'p2',identifier:'P2'},{id:'p3',identifier:'P3'}]});";
registerPermissionUiStubs("/sub-issues/issues-list/list-item.tsx", {
  "@plane/ui": ui,
  "@/hooks/use-projects": catalogue,
});
registerPermissionUiStubs("/components/issues/delete-issue-modal.tsx", {
  "@plane/ui": ui,
  "@/hooks/use-projects": catalogue,
});
registerPermissionUiStubs("/sub-issues/issues-list/root.tsx", {
  "@/components/issues/issue-layouts/use-group-columns": `export const useGroupColumns=()=>[{id:${JSON.stringify(ALL_ISSUES)},name:'All'}];`,
  "@/components/issues/issue-layouts/utils": "export const isWorkspaceLevel=()=>false;",
});
const { SubIssuesCollapsibleContent } =
  await import("../core/components/issues/issue-detail-widgets/sub-issues/content.tsx");
const { SubWorkItemPreferencesProvider } = await import("../core/providers/sub-work-item-preferences.tsx");
const { IssueService } = await import("../core/services/issue/index.ts");
registerPermissionUiStubs("/components/issues/issue-detail/parent-select.tsx", {
  "@/hooks/use-projects": catalogue,
  "../work-item-selection-modal": "export const WorkItemSelectionModal=()=>null;",
});
const { IssueParentSelectRoot } = await import("../core/components/issues/parent-select-root.tsx");
const { WorkItemDetailRecordProvider } = await import("../core/providers/work-item-detail-record.tsx");
const { useWorkItemDetail } = await import("../core/hooks/use-work-item-detail.ts");
function ParentDetail({ disabled }) {
  const { issue } = useWorkItemDetail("acme", "p3", "leaf");
  if (!issue) return null;
  return React.createElement(
    WorkItemDetailRecordProvider,
    {
      value: { workspaceSlug: "acme", projectId: "p3", issueId: "leaf", serviceType: "issues", issue },
    },
    React.createElement(IssueParentSelectRoot, {
      workspaceSlug: "acme",
      projectId: "p3",
      issueId: "leaf",
      disabled,
      issueOperations: {},
    })
  );
}
const { forgetItemReads } = await import("../core/queries/work-item-access.ts");
const identity = await import("../core/queries/identity.ts");
async function click(button) {
  assert.ok(button, "expected a visible action");
  await act(async () => {
    button.click();
    await Promise.resolve();
  });
  await settleQuery();
}

async function setup(t, { parentPicker = false } = {}) {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/P1-1",
    workspaceRole: 15,
    projectRoles: { p1: 20, p2: 20, p3: 20 },
  });
  const base = {
    created_by: "alice",
    priority: "low",
    state_id: "started",
    assignee_ids: [],
    label_ids: [],
    start_date: null,
    target_date: null,
    archived_at: null,
  };
  const rows = new Map([
    [
      "root",
      { ...base, id: "root", project_id: "p1", parent_id: null, name: "Root", sequence_id: 1, sub_issues_count: 1 },
    ],
    [
      "nested",
      {
        ...base,
        id: "nested",
        project_id: "p2",
        parent_id: "root",
        name: "Nested",
        sequence_id: 2,
        sub_issues_count: 1,
      },
    ],
    [
      "leaf",
      { ...base, id: "leaf", project_id: "p3", parent_id: "nested", name: "Leaf", sequence_id: 3, sub_issues_count: 0 },
    ],
  ]);
  const reads = [],
    writes = [];
  const transport = { patch: async () => {} };
  globalThis.nestedToasts = [];
  globalThis.nestedEditModal = undefined;
  t.mock.method(IssueService.prototype, "subIssues", async (_workspace, project, id) => {
    reads.push([project, id]);
    const parent = rows.get(id);
    if (!parent || parent.project_id !== project) throw { status: 404 };
    return { sub_issues: [...rows.values()].filter((row) => row.parent_id === id) };
  });
  t.mock.method(IssueService.prototype, "retrieve", async (_workspace, project, id) => {
    const row = rows.get(id);
    assert.equal(row.project_id, project);
    return { ...row, parent: rows.get(row.parent_id) };
  });
  t.mock.method(IssueService.prototype, "patchIssue", async (workspace, project, id, data) => {
    writes.push(["patch", workspace, project, id, data]);
    const row = rows.get(id);
    assert.equal(row.project_id, project);
    await transport.patch();
    const next = { ...row, ...data, parent: rows.get(data.parent_id ?? row.parent_id) };
    if (data.parent_id === null) next.parent = undefined;
    rows.set(id, next);
    return next;
  });
  t.mock.method(IssueService.prototype, "deleteIssue", async (workspace, project, id) => {
    writes.push(["delete", workspace, project, id]);
    assert.equal(rows.get(id).project_id, project);
    rows.delete(id);
  });
  const render = (disabled = false) =>
    fixture.render(
      React.createElement(
        SubWorkItemPreferencesProvider,
        null,
        parentPicker
          ? React.createElement(ParentDetail, { disabled })
          : React.createElement(SubIssuesCollapsibleContent, {
              workspaceSlug: "acme",
              projectId: "p1",
              parentIssueId: "root",
              disabled,
            })
      )
    );
  await render();
  await settleQuery();
  const row = (id) => fixture.environment.container.querySelector(`#issue-${id}`)?.parentElement;
  const expand = () => click(row("nested")?.querySelector('[aria-label="Expand sub-items"]'));
  const action = (id, text) =>
    [...row(id).querySelectorAll("button")].find((button) => button.textContent.includes(text));
  return {
    ...fixture,
    render,
    transport,
    rows,
    reads,
    writes,
    row,
    click,
    expand,
    action,
    removeParent: () => fixture.environment.container.querySelector('[aria-label="common.remove"]'),
  };
}
await test("expanding a cross-project child reads only that child's own parent scope", async (t) => {
  const view = await setup(t);
  await view.expand();
  assert.ok(view.row("leaf"));
  assert.deepEqual(view.reads, [
    ["p1", "root"],
    ["p2", "nested"],
  ]);
});
await test("inline edits in a nested project refresh the visible parent list", async (t) => {
  const view = await setup(t);
  await view.expand();
  await view.click(view.row("leaf").querySelector('[data-property="priority"]'));
  assert.deepEqual(view.writes, [["patch", "acme", "p3", "leaf", { priority: "urgent" }]]);
  assert.equal(view.row("leaf").querySelector('[data-property="priority"]').getAttribute("data-value"), '"urgent"');
});
await test("an accepted nested modal edit refreshes its parent in the correct project", async (t) => {
  const view = await setup(t);
  await view.expand();
  await view.click(view.action("leaf", "issue.edit"));
  const accepted = { ...view.rows.get("leaf"), priority: "urgent" };
  view.rows.set("leaf", accepted);
  await act(() => globalThis.nestedEditModal.onSubmit(accepted));
  await settleQuery();
  assert.deepEqual(view.writes, []);
  assert.equal(view.row("leaf").querySelector('[data-property="priority"]').getAttribute("data-value"), '"urgent"');
  assert.deepEqual(globalThis.nestedToasts, []);
});
await test("unlinking a nested child removes it from its actual parent's list", async (t) => {
  const view = await setup(t);
  await view.expand();
  await view.click(view.action("leaf", "issue.remove.parent.label"));
  assert.deepEqual(view.writes, [["patch", "acme", "p3", "leaf", { parent_id: null }]]);
  assert.equal(Boolean(view.row("leaf")), false);
});
await test("deleting a nested child removes it from its actual parent's list", async (t) => {
  const view = await setup(t);
  await view.expand();
  await view.click(view.action("leaf", "issue.delete.label"));
  await view.click(view.environment.container.querySelector("[data-delete-confirm]"));
  assert.deepEqual(view.writes, [["delete", "acme", "p3", "leaf"]]);
  assert.equal(Boolean(view.row("leaf")), false);
  assert.equal(globalThis.nestedToasts.length, 1);
});
await test("a retired session cannot refresh a nested modal's readers", async (t) => {
  const view = await setup(t);
  await view.expand();
  await view.click(view.action("leaf", "issue.edit"));
  const acknowledge = globalThis.nestedEditModal.onSubmit;
  await act(() => identity.advanceIdentitySession(view.client));
  await settleQuery();
  view.reads.length = 0;
  await act(() => acknowledge(view.rows.get("leaf")));
  await settleQuery();
  assert.deepEqual(view.reads, []);
  assert.deepEqual(view.writes, []);
  assert.deepEqual(globalThis.nestedToasts, []);
});

await test("a nested delete is rejected after access to its parent is revoked", async (t) => {
  const view = await setup(t);
  await view.expand();
  await view.click(view.action("leaf", "issue.delete.label"));
  await act(() =>
    forgetItemReads(view.client, {
      accountId: "alice",
      generation: identity.getIdentitySessionGeneration(view.client),
      workspaceSlug: "acme",
      projectId: "p2",
      issueId: "nested",
    })
  );
  await settleQuery();
  await view.click(view.environment.container.querySelector("[data-delete-confirm]"));
  assert.deepEqual(view.writes, []);
  assert.equal(globalThis.nestedToasts.length, 1);
  assert.equal(globalThis.nestedToasts[0].type, "error");
});

await test("a rejected parent removal shows one error and allows retry", async (t) => {
  const view = await setup(t, { parentPicker: true });
  view.transport.patch = async () => {
    throw new Error("save failed");
  };
  await view.click(view.removeParent());
  assert.equal(view.writes.length, 1);
  assert.equal(globalThis.nestedToasts.length, 1);
  assert.equal(globalThis.nestedToasts[0].type, "error");
  assert.ok(view.removeParent());
  view.transport.patch = async () => {};
  await view.click(view.removeParent());
  assert.equal(view.writes.length, 2);
  assert.equal(view.rows.get("leaf").parent_id, null);
  assert.equal(Boolean(view.removeParent()), false);
});
await test("a rejected row removal is handled once and leaves its row for retry", async (t) => {
  const view = await setup(t);
  await view.expand();
  view.transport.patch = async () => {
    throw new Error("save failed");
  };
  await view.click(view.action("leaf", "issue.remove.parent.label"));
  assert.equal(view.writes.length, 1);
  assert.equal(globalThis.nestedToasts.length, 1);
  assert.equal(globalThis.nestedToasts[0].type, "error");
  assert.ok(view.row("leaf"));
  view.transport.patch = async () => {};
  await view.click(view.action("leaf", "issue.remove.parent.label"));
  assert.equal(view.writes.length, 2);
  assert.equal(Boolean(view.row("leaf")), false);
});
await test("a pending row removal cannot be submitted twice", async (t) => {
  const view = await setup(t);
  await view.expand();
  const pending = Promise.withResolvers();
  view.transport.patch = () => pending.promise;
  const remove = view.action("leaf", "issue.remove.parent.label");
  await act(async () => {
    remove.click();
    remove.click();
    await Promise.resolve();
  });
  await settleQuery();
  const writesBeforeSettlement = view.writes.length;
  const disabledBeforeSettlement = view.action("leaf", "issue.remove.parent.label").disabled;
  const propertiesDisabled = view.row("leaf").querySelector('[data-property="priority"]').disabled;
  await act(() => pending.resolve());
  await settleQuery();
  assert.equal(writesBeforeSettlement, 1);
  assert.equal(disabledBeforeSettlement, true);
  assert.equal(propertiesDisabled, true);
  assert.equal(Boolean(view.row("leaf")), false);
});
await test("a pending parent removal cannot be submitted twice", async (t) => {
  const view = await setup(t, { parentPicker: true });
  const pending = Promise.withResolvers();
  view.transport.patch = () => pending.promise;
  const remove = view.removeParent();
  await act(async () => {
    remove.click();
    remove.click();
    await Promise.resolve();
  });
  await settleQuery();
  const writesBeforeSettlement = view.writes.length;
  const disabledBeforeSettlement = view.removeParent().disabled;
  await act(() => pending.resolve());
  await settleQuery();
  assert.equal(writesBeforeSettlement, 1);
  assert.equal(disabledBeforeSettlement, true);
  assert.equal(Boolean(view.removeParent()), false);
});
await test("a late failed parent removal cannot notify a replacement session", async (t) => {
  const view = await setup(t, { parentPicker: true });
  const pending = Promise.withResolvers();
  view.transport.patch = () => pending.promise;
  await view.click(view.removeParent());
  await act(() => identity.advanceIdentitySession(view.client));
  await settleQuery();
  await act(() => pending.reject(new Error("old failure")));
  await settleQuery();
  assert.deepEqual(globalThis.nestedToasts, []);
});
await test("a late successful row removal cannot notify a replacement session", async (t) => {
  const view = await setup(t);
  await view.expand();
  const pending = Promise.withResolvers();
  view.transport.patch = () => pending.promise;
  await view.click(view.action("leaf", "issue.remove.parent.label"));
  await act(() => identity.advanceIdentitySession(view.client));
  await settleQuery();
  await act(() => pending.resolve());
  await settleQuery();
  assert.equal(view.writes.length, 1);
  assert.deepEqual(globalThis.nestedToasts, []);
});

await test("a successful row removal reports one success", async (t) => {
  const view = await setup(t);
  await view.expand();
  await view.click(view.action("leaf", "issue.remove.parent.label"));
  assert.equal(Boolean(view.row("leaf")), false);
  assert.equal(globalThis.nestedToasts.length, 1);
  assert.equal(globalThis.nestedToasts[0].type, "success");
});
await test("a successful parent removal reports one success", async (t) => {
  const view = await setup(t, { parentPicker: true });
  await view.click(view.removeParent());
  assert.equal(Boolean(view.removeParent()), false);
  assert.equal(globalThis.nestedToasts.length, 1);
  assert.equal(globalThis.nestedToasts[0].type, "success");
});
