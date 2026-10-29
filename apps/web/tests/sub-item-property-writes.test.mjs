import assert from "node:assert/strict";
import test from "node:test";
import {
  React,
  act,
  createPermissionUiFixture,
  settleQuery,
} from "../../../tests/frontend/helpers/permission-ui-fixture.mjs";
import { registerWebTestModules } from "../../../tests/frontend/helpers/web-test-modules.mjs";

globalThis.React = React;
globalThis.subItemControls = {};
globalThis.subItemToasts = [];
const control = (name, exported) =>
  `export const ${exported} = props => globalThis.renderSubItemControl(${JSON.stringify(name)}, props);`;
globalThis.renderSubItemControl = (name, props) => {
  globalThis.subItemControls[name] = props;
  return React.createElement(
    "button",
    {
      type: "button",
      disabled: props.disabled,
      "data-property": name,
      "data-value": JSON.stringify(props.value),
      onClick: () => (props.onChange ?? props.onSelect)(globalThis.subItemChoices[name]),
    },
    name
  );
};
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/icons": "export const StartDatePropertyIcon=()=>null;export const DueDatePropertyIcon=()=>null;",
  "@plane/propel/toast":
    "export const TOAST_TYPE={ERROR:'error',SUCCESS:'success'};export const setToast=value=>globalThis.subItemToasts.push(value);",
  "@/components/dropdowns/date": control("date", "DateDropdown"),
  "@/components/dropdowns/date-range": control("range", "DateRangeDropdown"),
  "@/components/dropdowns/member/dropdown": control("assignee", "MemberDropdown"),
  "@/components/dropdowns/priority": control("priority", "PriorityDropdown"),
  "@/components/dropdowns/state/dropdown": control("state", "StateDropdown"),
  "@/hooks/use-project-states": "export const useProjectStates=()=>({getStateById:()=>({group:'started'})});",
  "@/components/issues/delete-issue-modal": "export const DeleteIssueModal=()=>null;",
  "@/providers/issue-detail-ui-state":
    "export const useIssueDetailUiState=()=>({toggleCreateIssueModal:()=>{},toggleDeleteIssueModal:()=>{}});",
  "../../issue-modal/modal": "export const CreateUpdateIssueModal=props=>{globalThis.subItemModal=props;return null;};",
  "./issues-list/root": "export const SubIssuesListRoot=props=>globalThis.renderSubItemList(props);",
});
const { SubIssuesListItemProperties } =
  await import("../core/components/issues/issue-detail-widgets/sub-issues/issues-list/properties.tsx");
const { SubIssuesCollapsibleContent } =
  await import("../core/components/issues/issue-detail-widgets/sub-issues/content.tsx");
const { useSubIssueOperations } = await import("../core/components/issues/issue-detail-widgets/sub-issues/helper.ts");
const { useWorkItemSubItems } = await import("../core/hooks/use-work-item-detail.ts");
const { IssueService } = await import("../core/services/issue/index.ts");
const identity = await import("../core/queries/identity.ts");
const { EIssueServiceType } = await import("@plane/types");
const row = {
  id: "child",
  parent_id: "parent",
  project_id: "p1",
  name: "Child",
  state_id: "started",
  priority: "low",
  assignee_ids: [],
  start_date: "2026-10-10",
  target_date: "2026-10-15",
  sequence_id: 2,
};
const displayProperties = { state: true, priority: true, assignee: true, start_date: true, due_date: true };
function deferred() {
  let resolve, reject;
  const promise = new Promise((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}
function NestedReader() {
  useWorkItemSubItems("acme", "p1", "nested");
  return null;
}

async function setup(t, save, modal = false) {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/P1-1",
    workspaceRole: 15,
    projectRoles: { p1: 20 },
  });
  globalThis.subItemControls = {};
  globalThis.subItemToasts = [];
  globalThis.subItemModal = undefined;
  globalThis.subItemList = undefined;
  globalThis.subItemChoices = { priority: "urgent", range: { from: new Date(2026, 9, 20), to: new Date(2026, 9, 25) } };
  globalThis.renderSubItemList = (props) => {
    globalThis.subItemList = props;
    return modal === "nested" ? React.createElement(NestedReader) : null;
  };
  let saved = { ...row };
  let currentRow = row;
  let canEdit = true;
  const writes = [];
  const reads = [];
  t.mock.method(IssueService.prototype, "subIssues", async (_workspace, _project, parentId) => {
    reads.push(parentId);
    return { sub_issues: [saved] };
  });
  t.mock.method(IssueService.prototype, "patchIssue", async (...args) => {
    writes.push(args);
    if (save) await save(...args);
    saved = { ...saved, ...args[3] };
    return saved;
  });
  function Probe() {
    const operations = useSubIssueOperations("acme", "p1", "parent", EIssueServiceType.ISSUES);
    useWorkItemSubItems("acme", "p1", "parent");
    return React.createElement(SubIssuesListItemProperties, {
      workspaceSlug: "acme",
      parentIssueId: "parent",
      issueId: currentRow.id,
      issue: currentRow,
      canEdit,
      updateSubIssue: operations.updateSubIssue,
      displayProperties,
    });
  }
  const render = async (nextRow = row, editable = true) => {
    currentRow = nextRow;
    canEdit = editable;
    await fixture.render(
      modal
        ? React.createElement(SubIssuesCollapsibleContent, {
            workspaceSlug: "acme",
            projectId: "p1",
            parentIssueId: "parent",
            disabled: false,
          })
        : React.createElement(Probe)
    );
    await settleQuery();
  };
  await render();
  const button = (name) => fixture.environment.container.querySelector(`[data-property="${name}"]`);
  const click = async (name) => {
    await act(() => button(name).click());
    await settleQuery();
  };
  return { ...fixture, writes, reads, render, button, click };
}
await test("sub-item date selection writes both endpoints atomically", async (t) => {
  const view = await setup(t);
  await view.click("range");
  assert.deepEqual(view.writes, [["acme", "p1", "child", { start_date: "2026-10-20", target_date: "2026-10-25" }]]);
  assert.equal(globalThis.subItemToasts.length, 1);
  assert.equal(globalThis.subItemToasts[0].type, "success");
});
await test("clearing a sub-item date range sends one patch", async (t) => {
  const view = await setup(t);
  globalThis.subItemChoices.range = undefined;
  await view.click("range");
  assert.deepEqual(view.writes, [["acme", "p1", "child", { start_date: null, target_date: null }]]);
});
await test("a rejected sub-item save is handled once and allows retry", async (t) => {
  const pending = deferred();
  let attempt = 0;
  const view = await setup(t, () => (++attempt === 1 ? pending.promise : Promise.resolve()));
  await act(() => {
    globalThis.subItemControls.priority.onChange("urgent");
    globalThis.subItemControls.priority.onChange("high");
  });
  await settleQuery();
  assert.equal(view.writes.length, 1);
  assert.equal(view.button("priority").disabled, true);
  assert.equal(view.button("state").disabled, true);
  await act(() => pending.reject(new Error("Rejected")));
  await settleQuery();
  assert.equal(globalThis.subItemToasts.length, 1);
  assert.equal(globalThis.subItemToasts[0].type, "error");
  assert.equal(view.button("priority").disabled, false);
  assert.equal(view.button("priority").getAttribute("data-value"), '"low"');
  await view.click("priority");
  assert.equal(view.writes.length, 2);
  assert.deepEqual(
    globalThis.subItemToasts.map((toast) => toast.type),
    ["error", "success"]
  );
});
await test("a late accepted save does not notify a replacement sub-item", async (t) => {
  const pending = deferred();
  const view = await setup(t, () => pending.promise);
  await view.click("priority");
  await view.render({ ...row, id: "replacement" });
  await act(() => pending.resolve());
  await settleQuery();
  assert.equal(globalThis.subItemToasts.length, 0);
  assert.equal(view.button("priority").disabled, false);
});
await test("a read-only sub-item rejects retained change callbacks", async (t) => {
  const view = await setup(t);
  const change = globalThis.subItemControls.priority.onChange;
  await view.render(row, false);
  await act(() => change("urgent"));
  await settleQuery();
  assert.deepEqual(view.writes, []);
  assert.equal(view.button("priority").disabled, true);
});
await test("modal acknowledgement only refreshes readers and leaves notification to the modal", async (t) => {
  const view = await setup(t, undefined, true);
  await act(() =>
    globalThis.subItemList.onAction({
      kind: "update",
      issue: row,
      parent: { projectId: "p1", issueId: "parent", serviceType: EIssueServiceType.ISSUES },
    })
  );
  await act(() => globalThis.subItemModal.onSubmit({ ...row, priority: "urgent" }));
  await settleQuery();
  assert.deepEqual(view.writes, []);
  assert.equal(globalThis.subItemToasts.length, 0);
  assert.equal(view.reads.length, 2);
});
await test("a modal acknowledgement after session retirement cannot fail an accepted save", async (t) => {
  const view = await setup(t, undefined, true);
  await act(() =>
    globalThis.subItemList.onAction({
      kind: "update",
      issue: row,
      parent: { projectId: "p1", issueId: "parent", serviceType: EIssueServiceType.ISSUES },
    })
  );
  const accepted = globalThis.subItemModal.onSubmit;
  await act(() => identity.advanceIdentitySession(view.client));
  await act(() => accepted({ ...row, priority: "urgent" }));
  await settleQuery();
  assert.deepEqual(view.writes, []);
  assert.equal(globalThis.subItemToasts.length, 0);
});

await test("editing a nested sub-item refreshes its actual parent", async (t) => {
  const view = await setup(t, undefined, "nested");
  await act(() =>
    globalThis.subItemList.onAction({
      kind: "update",
      issue: { ...row, parent_id: "nested" },
      parent: { projectId: "p1", issueId: "nested", serviceType: EIssueServiceType.ISSUES },
    })
  );
  await act(() => globalThis.subItemModal.onSubmit({ ...row, parent_id: "nested", priority: "urgent" }));
  await settleQuery();
  assert.deepEqual(view.reads, ["parent", "nested", "nested"]);
  assert.deepEqual(view.writes, []);
});
await test("sub-item edits cannot write into an unauthorized project", async (t) => {
  const view = await setup(t);
  await view.render({ ...row, project_id: "denied" });
  await view.click("priority");
  assert.deepEqual(view.writes, []);
  assert.equal(globalThis.subItemToasts.length, 1);
  assert.equal(globalThis.subItemToasts[0].type, "error");
});
await test("sub-item state and assignee updates keep their payloads", async (t) => {
  const view = await setup(t);
  globalThis.subItemChoices.state = "done";
  globalThis.subItemChoices.assignee = ["member"];
  await view.click("state");
  await view.click("assignee");
  assert.deepEqual(view.writes, [
    ["acme", "p1", "child", { state_id: "done" }],
    ["acme", "p1", "child", { assignee_ids: ["member"] }],
  ]);
});

await test("a partial sub-item date range clears the omitted endpoint", async (t) => {
  const view = await setup(t);
  globalThis.subItemChoices.range = { from: new Date(2026, 9, 20) };
  await view.click("range");
  assert.deepEqual(view.writes, [["acme", "p1", "child", { start_date: "2026-10-20", target_date: null }]]);
});
