import assert from "node:assert/strict";
import {
  React,
  act,
  createPermissionUiFixture,
  settleQuery,
  registerPermissionUiStubs,
} from "../../../../tests/frontend/helpers/permission-ui-fixture.mjs";
import { registerWebTestModules } from "../../../../tests/frontend/helpers/web-test-modules.mjs";
import { webRequire } from "../../../../tests/frontend/helpers/react.mjs";
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
      onClick: () => globalThis.handleDetailFixtureChange(kind, props),
    },
    kind
  );
};
globalThis.handleDetailFixtureChange = (kind, props) => {
  const value =
    kind === "cycle"
      ? globalThis.assignmentCycleValue
      : kind === "module"
        ? globalThis.assignmentModuleValue
        : kind === "assignee"
          ? ["bob"]
          : kind === "date"
            ? new Date("2026-10-30")
            : "urgent";
  const result = props.onChange(value);
  if (result?.catch) result.catch((error) => globalThis.detailUnhandled.push(error));
};
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/toast":
    "export const TOAST_TYPE={ERROR:'error',SUCCESS:'success'};export const setToast=t=>globalThis.detailWriteToasts.push(t);export const setPromiseToast=(promise,messages)=>{promise.then(()=>globalThis.detailWriteToasts.push({type:'success',title:messages.success.title,message:messages.success.message()}),()=>globalThis.detailWriteToasts.push({type:'error',title:messages.error.title,message:messages.error.message()}));};",
  "@/app/assets/empty-state/issue.svg?url": "export default '';",
  "@/components/common/empty-state": "export const EmptyState=()=>null;",
  "@/providers/sidebar-state": "export const useIssueDetailSidebarCollapsed=()=>false;",
  "@/providers/issue-detail-ui-state":
    "export const useIssueDetailUiState=()=>({peekIssue:globalThis.detailWritePeek,setPeekIssue:()=>{},isParentIssueModalOpen:'item',toggleParentIssueModal:()=>{}});export const useWorkItemPeek=()=>({});",
  "@/hooks/use-issue-list-type": `export const useIssueListType=()=>${JSON.stringify(EIssuesStoreType.PROJECT)};`,
  "@/hooks/use-issue-properties": "export const useWorkItemProperties=()=>{};",
  "@/hooks/use-projects":
    "export const useWorkspaceProjectDetails=()=>({projects:[{id:'p1',identifier:'P1',estimate:'estimate',cycle_view:true,module_view:true}]});",
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
  "@/components/dropdowns/cycle": dropdown("cycle", "CycleDropdown"),
  "@/components/dropdowns/module/dropdown": dropdown("module", "ModuleDropdown"),
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
const { IssueDetailRoot } = await import("../../core/components/issues/issue-detail/root.tsx");
const { IssuePeekOverview } = await import("../../core/components/issues/peek-overview/root.tsx");
const { PeekOverviewProperties } = await import("../../core/components/issues/peek-overview/properties.tsx");
const { IssueParentDetail } = await import("../../core/components/issues/issue-detail/parent/root.tsx");
const { useWorkItemDetailRecord } = await import("../../core/providers/work-item-detail-record.tsx");
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
const { IssueService } = await import("../../core/services/issue/index.ts");
const { ModuleService } = await import("../../core/services/module.service.ts");
const identity = await import("../../core/queries/identity.ts");
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
  row = { ...row, cycle_id: "old-cycle", module_ids: ["old-module"] };
  const transport = { patch: async () => {}, assignment: async () => {}, read: async () => {} };
  globalThis.assignmentCycleValue = "new-cycle";
  globalThis.assignmentModuleValue = ["new-module"];
  globalThis.detailUnhandled = [];
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
  t.mock.method(IssueService.prototype, "retrieve", async () => {
    await transport.read();
    return row;
  });
  t.mock.method(IssueService.prototype, "addIssueToCycle", async (workspace, project, cycleId, data) => {
    writes.push(["cycle-add", workspace, project, cycleId, data]);
    await transport.assignment();
    row = { ...row, cycle_id: cycleId };
  });
  t.mock.method(IssueService.prototype, "removeIssueFromCycle", async (workspace, project, cycleId, issueId) => {
    writes.push(["cycle-remove", workspace, project, cycleId, issueId]);
    await transport.assignment();
    row = { ...row, cycle_id: null };
  });
  t.mock.method(ModuleService.prototype, "addModulesToIssue", async (workspace, project, issueId, data) => {
    writes.push(["module-change", workspace, project, issueId, data]);
    await transport.assignment();
    row = {
      ...row,
      module_ids: [...new Set([...row.module_ids.filter((id) => !data.removed_modules.includes(id)), ...data.modules])],
    };
  });
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

export { act, identity, setup, settleQuery };
