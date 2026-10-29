import assert from "node:assert/strict";
import {
  React,
  act,
  createPermissionUiFixture,
  settleQuery,
  registerPermissionUiStubs,
} from "../../../../tests/frontend/helpers/permission-ui-fixture.mjs";
import { registerWebTestModules } from "../../../../tests/frontend/helpers/web-test-modules.mjs";
import { EIssuesStoreType } from "@plane/types";
globalThis.React = React;
registerWebTestModules({
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/toast":
    "export const TOAST_TYPE={ERROR:'error',SUCCESS:'success'};export const setToast=t=>globalThis.lifecycleState.toasts.push(t);",
  "@plane/propel/icons":
    "export const CopyLinkIcon=()=>null,CenterPanelIcon=()=>null,FullScreenPanelIcon=()=>null,SidePanelIcon=()=>null,LinkIcon=()=>null,CopyIcon=()=>null,NewTabIcon=()=>null,EditIcon=()=>null,ArchiveIcon=()=>null,TrashIcon=()=>null;",
  "@plane/propel/icon-button":
    "export const IconButton=({icon,...props})=>globalThis.React.createElement('button',props);",
  "@plane/propel/tooltip": "export const Tooltip=({children})=>children;",
  "@plane/propel/button":
    "export const Button=({children,loading,...props})=>globalThis.React.createElement('button',props,children);",
  "@plane/ui":
    "const R=globalThis.React;export const ContextMenu=()=>null;export const CustomMenu=({children})=>children;export const CustomSelect=Object.assign(({children})=>children,{Option:({children})=>children});export const EModalPosition={},EModalWidth={};export const ModalCore=({children,isOpen})=>isOpen?R.createElement('div',{role:'dialog'},children):null;export const AlertModalCore=props=>props.isOpen?R.createElement('div',{role:'dialog'},R.createElement('button',{'data-confirm':'delete',disabled:props.isSubmitting,onClick:props.handleSubmit},'Delete'),R.createElement('button',{onClick:props.handleClose},'Cancel')):null;",
  "@/components/common/quick-action-menu":
    "export const QuickActionMenu=({items})=>{globalThis.lifecycleMenu=items;return globalThis.React.createElement('div',null,...items.filter(i=>i.shouldRender!==false).map(i=>globalThis.React.createElement('button',{key:i.key,'data-action':i.key,disabled:i.disabled,onClick:()=>{const result=i.action();if(result?.catch)result.catch(e=>globalThis.lifecycleState.unhandled.push(e));}},i.title)));};export const renderQuickActionMenuItems=items=>globalThis.React.createElement(QuickActionMenu,{items});",
  "@/providers/issue-detail-ui-state":
    "export const useIssueDetailUiState=()=>({peekIssue:globalThis.lifecycleState.peek,setPeekIssue:value=>{globalThis.lifecycleState.closed++;globalThis.lifecycleState.peek=value;},getIsIssuePeeked:id=>globalThis.lifecycleState.peek?.issueId===id,isAnyModalOpen:false});",
  "@/hooks/use-issue-list-type": `export const useIssueListType=()=>${JSON.stringify(EIssuesStoreType.PROJECT)};`,
  "@/hooks/use-issue-properties": "export const useWorkItemProperties=()=>{};",
  "@/hooks/use-projects":
    "export const useWorkspaceProjectDetails=()=>({projects:[{id:'p1',identifier:'P1',estimate:'estimate',cycle_view:true,module_view:true}]});",
  "@/hooks/use-project-states":
    "export const useProjectStates=()=>({getStateById:()=>({group:'completed'})});export const useWorkspaceStates=()=>({getStateById:()=>({group:'completed'}),getProjectStates:()=>[]});",
  "@/hooks/use-member-rosters": "export const useWorkspaceMemberRoster=()=>({memberById:()=>undefined});",
  "@/hooks/use-estimates":
    "export const useProjectEstimates=()=>({isEnabled:true,estimates:[],selectedId:null});export const useWorkspaceEstimates=()=>({});",
  "@/hooks/use-platform-os": "export const usePlatformOS=()=>({isMobile:false});",
  "@/hooks/use-keypress": "export default ()=>{};",
  "@/hooks/use-peek-overview-outside-click": "export default ()=>{};",
});
for (const parent of ["use-cycles", "use-modules", "use-workspace-issue-properties"])
  registerPermissionUiStubs(`/hooks/${parent}.ts`, {
    "./use-estimates":
      "export const useProjectEstimates=()=>({isEnabled:true,estimates:[],selectedId:null});export const useWorkspaceEstimates=()=>({});",
  });
registerPermissionUiStubs("/components/issues/peek-overview/view.tsx", {
  "../issue-detail/issue-activity": "export const IssueActivity=()=>null;",
  "../issue-detail-widgets": "export const IssueDetailWidgets=()=>null;",
  "./error": "export const IssuePeekOverviewError=()=>null;",
  "./loader": "export const IssuePeekOverviewLoader=()=>null;",
  "./issue-detail":
    "export const PeekOverviewIssueDetails=props=>{globalThis.lifecycleOperations=props.issueOperations;return null;};",
  "./properties": "export const PeekOverviewProperties=()=>null;",
});
for (const parent of ["peek-overview/header.tsx", "issue-detail/issue-detail-quick-actions.tsx"])
  registerPermissionUiStubs(`/components/issues/${parent}`, {
    "../issue-layouts/quick-action-dropdowns":
      "export const WorkItemDetailQuickActions=props=>globalThis.React.createElement(globalThis.lifecycleQuickActions,props);",
    "../issue-detail/subscription": "export const IssueSubscription=()=>null;",
    "./subscription": "export const IssueSubscription=()=>null;",
    "../issue-update-status": "export const NameDescriptionUpdateStatus=()=>null;",
  });
registerPermissionUiStubs("/components/issues/issue-layouts/quick-action-dropdowns/issue-quick-action-modals.tsx", {
  "../../issue-modal/modal": "export const CreateUpdateIssueModal=()=>null;",
});
globalThis.lifecycleQuickActions = (
  await import("../../core/components/issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx")
).WorkItemDetailQuickActions;
const { IssuePeekOverview } = await import("../../core/components/issues/peek-overview/root.tsx");
const { IssueDetailQuickActions } =
  await import("../../core/components/issues/issue-detail/issue-detail-quick-actions.tsx");
const { IssueService } = await import("../../core/services/issue/issue.service.ts");
const { IssueArchiveService } = await import("../../core/services/issue/issue_archive.service.ts");
const { LocalIssuePreferencesProvider } = await import("../../core/providers/local-issue-preferences.tsx");
const { ParentIssuePreferencesProvider } = await import("../../core/providers/parent-issue-preferences.tsx");
const { IssueFiltersService } = await import("../../core/services/issue_filter.service.ts");
const { CycleService } = await import("../../core/services/cycle.service.ts");
const { ModuleService } = await import("../../core/services/module.service.ts");
const identity = await import("../../core/queries/identity.ts");
async function click(button) {
  assert.ok(button);
  await act(async () => {
    button.click();
    await Promise.resolve();
  });
  await settleQuery();
}
async function setup(t, mode = "peek", options = {}) {
  const fixture = await createPermissionUiFixture(t, {
    path: options.parent
      ? `/acme/projects/p1/${options.parent}s/parent`
      : options.archived
        ? "/acme/projects/p1/archives/issues/item"
        : "/acme/projects/p1/issues/P1-1",
    workspaceRole: 15,
    projectRoles: { p1: 20, p2: 20 },
  });
  const state = {
    peek: { workspaceSlug: "acme", projectId: "p1", issueId: "item", isArchived: !!options.archived },
    closed: 0,
    toasts: [],
    unhandled: [],
    writes: [],
    parentReads: 0,
  };
  globalThis.lifecycleState = state;
  let row = {
    id: "item",
    project_id: "p1",
    name: "Original",
    sequence_id: 1,
    created_by: "alice",
    state_id: "completed",
    archived_at: options.archived ? "2026-10-01" : null,
    assignee_ids: [],
    label_ids: [],
    module_ids: [],
  };
  const transport = { write: async () => {}, read: async () => {}, canonical: {} };
  globalThis.lifecycleOperations = undefined;
  t.mock.method(IssueService.prototype, "patchIssue", async (_w, p, id, data) => {
    row = { ...row, ...data, ...transport.canonical };
    return { ...row, id, project_id: p };
  });
  t.mock.method(IssueService.prototype, "retrieve", async (_w, p, id) => {
    await transport.read();
    return { ...row, id, project_id: p };
  });
  for (const [Service, method, kind] of [
    [IssueService, "deleteIssue", "delete"],
    [IssueArchiveService, "archiveIssue", "archive"],
    [IssueArchiveService, "restoreIssue", "restore"],
  ])
    t.mock.method(Service.prototype, method, async (...args) => {
      state.writes.push([kind, ...args]);
      await transport.write();
      if (kind === "restore") row = { ...row, archived_at: null };
      if (kind === "archive") row = { ...row, archived_at: "2026-10-29" };
    });
  if (options.parent) {
    const service = options.parent === "cycle" ? CycleService : ModuleService;
    const method = options.parent === "cycle" ? "retrieve" : "getModuleDetails";
    t.mock.method(service.prototype, method, async () => {
      state.parentReads++;
      return { id: "parent", project_id: "p1", name: "Parent", total_issues: state.writes.length ? 0 : 1 };
    });
    t.mock.method(
      IssueFiltersService.prototype,
      options.parent === "cycle" ? "fetchCycleIssueFilters" : "fetchModuleIssueFilters",
      async () => ({})
    );
  }
  const content = () =>
    mode === "peek"
      ? React.createElement(IssuePeekOverview, {
          embedIssue: true,
          embedRemoveCurrentNotification: options.onDismiss,
          storeType:
            options.parent === "cycle"
              ? EIssuesStoreType.CYCLE
              : options.parent === "module"
                ? EIssuesStoreType.MODULE
                : EIssuesStoreType.PROJECT,
        })
      : React.createElement(IssueDetailQuickActions, {
          workspaceSlug: "acme",
          projectId: "p1",
          issueId: state.peek?.issueId ?? "item",
        });
  const render = () =>
    fixture.render(
      options.parent
        ? React.createElement(
            ParentIssuePreferencesProvider,
            { domain: options.parent, workspaceSlug: "acme", projectId: "p1", parentId: "parent" },
            content()
          )
        : options.archived && !options.noArchivePrefs
          ? React.createElement(
              LocalIssuePreferencesProvider,
              { domain: EIssuesStoreType.ARCHIVED, workspaceSlug: "acme", entityId: "p1" },
              content()
            )
          : content()
    );
  await render();
  await settleQuery();
  const action = (key) => fixture.environment.container.querySelector(`[data-action="${key}"]`);
  assert.ok(action("delete"), "native action menu mounted");

  return {
    ...fixture,
    state,
    transport,
    render,
    renderElement: fixture.render,
    action,
    click,
    dialog: () => fixture.environment.container.querySelector('[role="dialog"]'),
    confirm: () => fixture.environment.container.querySelector('[data-confirm="delete"]'),
  };
}
export { setup, React, act, identity, settleQuery };
