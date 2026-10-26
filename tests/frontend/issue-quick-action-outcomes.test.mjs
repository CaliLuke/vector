import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";

registerWebModuleHooks();
globalThis.issueActionReact = React;
const state = {
  removeFails: false,
  archiveFails: false,
  navigationFails: false,
  removeCalls: 0,
  navigations: 0,
  toasts: [],
  closed: 0,
  peekIssue: { workspaceSlug: "workspace", projectId: "project", issueId: "issue-1" },
  removePromise: undefined,
  mountPeekHeader: false,
  deletePromise: Promise.resolve(),
  updateReceipt: { id: "issue-1", project_id: "project", name: "Canonical" },
  reconciled: [],
  peekWithoutRecord: false,
};
globalThis.issueActionState = state;
const issue = {
  id: "issue-1",
  project_id: "project",
  workspace_id: "workspace-1",
  sequence_id: 1,
  created_by: "alice",
};
globalThis.issueActionIssue = issue;

const stubs = {
  "@tanstack/react-router": `const React=globalThis.issueActionReact;export const Link=props=>React.createElement("a",{href:props.to,onClick:props.onClick},props.children);export const useRouterState=({select})=>select({matches:[]});export const useRouter=()=>({state:{matches:[]},navigate:async()=>{globalThis.issueActionState.navigations++;if(globalThis.issueActionState.navigationFails)throw new Error("navigation failed")}});export const useLocation=()=>({pathname:"/issues"});export const useParams=()=>({workspaceSlug:"workspace"});`,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/icons": `export const CopyLinkIcon=()=>null;export const CenterPanelIcon=()=>null;export const FullScreenPanelIcon=()=>null;export const SidePanelIcon=()=>null;`,
  "@plane/propel/icon-button": `const React=globalThis.issueActionReact;export const IconButton=({icon,...props})=>React.createElement("button",props);`,
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:"success",ERROR:"error"};export const setToast=toast=>globalThis.issueActionState.toasts.push(toast);export const setPromiseToast=()=>{};`,
  "@plane/propel/tooltip": `const React=globalThis.issueActionReact;export const Tooltip=({children})=>React.createElement(React.Fragment,null,children);`,
  "@plane/ui": `const React=globalThis.issueActionReact;export const CustomSelect=Object.assign(({customButton,children})=>React.createElement("div",null,customButton,children),{Option:({children})=>React.createElement("div",null,children)});export const Loader=()=>null;export const AlertModalCore=props=>{globalThis.issueDeleteModal=props;return React.createElement("div",{role:"dialog"},React.createElement("button",{id:"confirm-delete",disabled:props.isSubmitting,onClick:props.handleSubmit},"Delete"),React.createElement("button",{id:"cancel-delete",onClick:props.handleClose},"Cancel"))};`,
  "@/hooks/use-work-item-detail": `export const useWorkItemDetail=()=>{return {issue:globalThis.issueActionState.peekWithoutRecord?undefined:globalThis.issueActionIssue,scope:{accountId:"alice",sessionGeneration:0,workspaceSlug:"workspace",projectId:"project",membershipRole:20,membershipRevision:"v1",projectMembershipRole:20,projectRole:20,serviceType:"issues"},query:{refetch:async()=>globalThis.issueActionIssue}}};`,
  "@/providers/issue-detail-ui-state": `export const useIssueDetailUiState=()=>({peekIssue:globalThis.issueActionState.peekIssue,setPeekIssue:value=>{globalThis.issueActionState.closed++;globalThis.issueActionState.peekIssue=value},getIsIssuePeeked:id=>globalThis.issueActionState.peekIssue?.issueId===id});`,
  "@/hooks/use-issues-actions": `export const useIssuesActions=()=>({removeIssue:async()=>{globalThis.issueActionState.removeCalls++;if(globalThis.issueActionState.removePromise)await globalThis.issueActionState.removePromise;if(globalThis.issueActionState.removeFails)throw new Error("remove failed")},archiveIssue:async()=>{if(globalThis.issueActionState.archiveFails)throw new Error("archive failed")},updateIssue:async()=>globalThis.issueActionState.updateReceipt});`,
  "@/queries/issue-details": `export const captureIssueDetailReconciliation=()=>({reconcile:async receipt=>{globalThis.issueActionState.reconciled.push(receipt)},remove:()=>{},refresh:async()=>{}});`,
  "@/hooks/store/use-project": `export const useProject=()=>({getProjectIdentifierById:()=>"PRJ",getProjectById:()=>({})});`,
  "@/services/user.service":
    "export class UserService { currentUser() { throw new Error('unexpected identity request'); } } export default new UserService();",
  "@/services/workspace.service":
    "export class WorkspaceService { workspaceMemberMe() { throw new Error('unexpected membership request'); } getWorkspaceUserProjectsRole() { throw new Error('unexpected roles request'); } }",
  "@/hooks/use-platform-os": `export const usePlatformOS=()=>({isMobile:false});`,
  "@/hooks/use-current-user": `export const useCurrentUser=()=>({data:{id:"alice"},isSuccess:true,isPending:false,isError:false,refetch:async()=>({})});`,
  "@/hooks/use-projects": `export const useWorkspaceProjectDetails=()=>({projects:[{id:"project",identifier:"PRJ"}]});`,
  "@/providers/work-item-detail-record": `export const useWorkItemDetailRecord=()=>({issueId:globalThis.issueActionIssue.id,issue:globalThis.issueActionIssue});export const WorkItemDetailRecordProvider=({children})=>children;`,
  "@/components/issues/issue-update-status": `export const NameDescriptionUpdateStatus=()=>null;`,
  "@/hooks/use-issue-list-type": `export const useIssueListType=()=>"project";`,
  "@/hooks/use-issue-properties": `export const useWorkItemProperties=()=>{};`,
  "../issue-layouts/quick-action-dropdowns": `const React=globalThis.issueActionReact;export const WorkItemDetailQuickActions=props=>{globalThis.issueActionCallbacks=props;return React.createElement("button",{id:"peek-delete",onClick:()=>{globalThis.issueActionState.deletePromise=props.handleDelete()}},"Delete")};`,
  "@plane/utils/string": "export const copyUrlToClipboard=async()=>{};\nexport const copyTextToClipboard=async()=>{};",
  "@plane/utils/work-item": 'export const generateWorkItemLink=()=>"/issues/issue-1";',
  "./subscription": `export const IssueSubscription=()=>null;`,
  "../issue-detail/subscription": `export const IssueSubscription=()=>null;`,
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "./view" && context.parentURL?.includes("peek-overview/root.tsx")) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent(`const React=globalThis.issueActionReact;export const IssueView=props=>{globalThis.issuePeekOperations=props.issueOperations;if(!globalThis.issueActionState.mountPeekHeader)return null;const Header=globalThis.issuePeekHeader;return React.createElement(Header,{peekMode:"side-peek",setPeekMode:()=>{},removeRoutePeekId:()=>{globalThis.issueActionState.closed++;globalThis.issueActionState.peekIssue=undefined},workspaceSlug:props.workspaceSlug,projectId:props.projectId,issueId:props.issueId,isArchived:props.is_archived,disabled:props.disabled,embedIssue:props.embedIssue,toggleDeleteIssueModal:()=>{},toggleArchiveIssueModal:()=>{},toggleDuplicateIssueModal:()=>{},toggleEditIssueModal:()=>{},handleRestoreIssue:async()=>{},isSubmitting:"saved",removeIssue:()=>props.issueOperations.remove(props.workspaceSlug,props.projectId,props.issueId),archiveIssue:()=>props.issueOperations.archive?.(props.workspaceSlug,props.projectId,props.issueId)??Promise.resolve()})}`)}`,
      };
    }
    return resolvePackageStub(specifier, stubs, context, nextResolve);
  },
});

const { IssuePeekOverviewHeader } = await import("../../apps/web/core/components/issues/peek-overview/header.tsx");
globalThis.issuePeekHeader = IssuePeekOverviewHeader;

async function createIssueActionEnvironment() {
  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
  client.setQueryData(currentUserOptions().queryKey, { id: "alice" });
  await seedPermissionQueries(client, {
    userId: "alice",
    workspaceSlug: "workspace",
    workspaceRole: 20,
    projectRoles: { project: 20, "project-1": 20 },
  });
  return {
    ...environment,
    render: (element) => environment.render(React.createElement(QueryClientProvider, { client }, element)),
    unmount: async () => {
      await environment.unmount();
      client.clear();
    },
  };
}

async function renderIssueDetailQuickActions(environment) {
  const { IssueDetailQuickActions } =
    await import("../../apps/web/core/components/issues/issue-detail/issue-detail-quick-actions.tsx");
  await environment.render(
    React.createElement(IssueDetailQuickActions, {
      workspaceSlug: "workspace",
      projectId: "project",
      issueId: issue.id,
    })
  );
}

async function assertDeleteFailureLeavesModalRetryable(t, DeleteModal) {
  state.toasts = [];
  state.closed = 0;
  const environment = await createIssueActionEnvironment();
  t.after(() => environment.unmount());
  await environment.render(
    React.createElement(DeleteModal, {
      isOpen: true,
      data: issue,
      handleClose: () => state.closed++,
      onSubmit: async () => {
        throw new Error("remove failed");
      },
    })
  );
  await React.act(async () => environment.container.querySelector("#confirm-delete").click());
  assert.equal(state.closed, 0);
  assert.equal(environment.container.querySelector('[role="dialog"]') !== null, true);
  assert.equal(environment.container.querySelector("#confirm-delete").disabled, false);
  assert.equal(state.toasts.length, 1);
  assert.equal(state.toasts[0].type, "error");
}

await test("issue-detail delete and archive callbacks reject primary failures", async (t) => {
  state.removeFails = true;
  state.archiveFails = true;
  state.navigations = 0;
  state.removeCalls = 0;
  const environment = await createIssueActionEnvironment();
  t.after(() => environment.unmount());
  await renderIssueDetailQuickActions(environment);
  await assert.rejects(globalThis.issueActionCallbacks.handleDelete(), /remove failed/);
  await assert.rejects(globalThis.issueActionCallbacks.handleArchive(), /archive failed/);
  assert.equal(state.navigations, 0, "failed primary actions do not navigate or complete the modal callback");
});

await test("peek archive callback rejects the store failure", async (t) => {
  state.archiveFails = true;
  state.peekWithoutRecord = true;
  const environment = await createIssueActionEnvironment();
  t.after(async () => {
    state.mountPeekHeader = false;
    await environment.unmount();
  });
  const { IssuePeekOverview } = await import("../../apps/web/core/components/issues/peek-overview/root.tsx");
  await environment.render(React.createElement(IssuePeekOverview, {}));
  await assert.rejects(globalThis.issuePeekOperations.archive("workspace", "project", "issue"), /archive failed/);
});

await test("peek delete failure keeps the peek open and reports one error", async (t) => {
  state.removeFails = true;
  state.removePromise = undefined;
  state.toasts = [];
  state.closed = 0;
  state.mountPeekHeader = true;
  state.peekIssue = { workspaceSlug: "workspace", projectId: "project", issueId: issue.id };
  const environment = await createIssueActionEnvironment();
  t.after(async () => {
    state.mountPeekHeader = false;
    await environment.unmount();
  });
  const { IssuePeekOverview } = await import("../../apps/web/core/components/issues/peek-overview/root.tsx");
  await environment.render(React.createElement(IssuePeekOverview, {}));
  React.act(() => environment.container.querySelector("#peek-delete").click());
  await state.deletePromise;
  assert.equal(state.closed, 0);
  assert.equal(state.toasts.length, 1);
  assert.equal(state.toasts[0].type, "error");
});

await test("late accepted peek delete does not close a replacement peek", async (t) => {
  state.removeFails = false;
  state.toasts = [];
  state.closed = 0;
  state.mountPeekHeader = true;
  state.peekIssue = { workspaceSlug: "workspace", projectId: "project", issueId: issue.id };
  let resolveRemove;
  state.removePromise = new Promise((resolve) => {
    resolveRemove = resolve;
  });
  const environment = await createIssueActionEnvironment();
  t.after(async () => {
    state.mountPeekHeader = false;
    await environment.unmount();
  });
  const { IssuePeekOverview } = await import("../../apps/web/core/components/issues/peek-overview/root.tsx");
  await environment.render(React.createElement(IssuePeekOverview, {}));
  React.act(() => environment.container.querySelector("#peek-delete").click());
  state.peekIssue = { workspaceSlug: "workspace", projectId: "project", issueId: "replacement-issue" };
  await environment.render(React.createElement(IssuePeekOverview, {}));
  resolveRemove();
  await state.deletePromise;
  assert.equal(state.closed, 0);
  assert.equal(state.peekIssue.issueId, "replacement-issue");
  state.removePromise = undefined;
});

await test("navigation failure after deletion does not reject the completed delete", async (t) => {
  state.peekWithoutRecord = false;
  state.removeFails = false;
  state.navigationFails = true;
  state.navigations = 0;
  state.removeCalls = 0;
  const environment = await createIssueActionEnvironment();
  t.after(() => environment.unmount());
  await renderIssueDetailQuickActions(environment);
  await globalThis.issueActionCallbacks.handleDelete();
  assert.equal(state.removeCalls, 1, "the completed deletion runs exactly once");
  assert.equal(state.navigations, 1);
  assert.match(state.toasts.at(-1).message, /deleted.*list could not be opened/);
  state.navigationFails = false;
});

await test("delete modal stays open with retry enabled after a failed delete", async (t) => {
  const { DeleteIssueModal } = await import("../../apps/web/core/components/issues/delete-issue-modal.tsx");
  await assertDeleteFailureLeavesModalRetryable(t, DeleteIssueModal);
});

await test("workspace draft delete modal stays open with retry enabled after failure", async (t) => {
  const { WorkspaceDraftIssueDeleteIssueModal } =
    await import("../../apps/web/core/components/issues/workspace-draft/delete-modal.tsx");
  await assertDeleteFailureLeavesModalRetryable(t, WorkspaceDraftIssueDeleteIssueModal);
});

await test("peek text update reconciles the canonical accepted receipt", async (t) => {
  state.toasts = [];
  state.reconciled = [];
  state.peekWithoutRecord = true;
  const environment = await createIssueActionEnvironment();
  t.after(() => environment.unmount());
  const { IssuePeekOverview } = await import("../../apps/web/core/components/issues/peek-overview/root.tsx");
  await environment.render(React.createElement(IssuePeekOverview, {}));
  await globalThis.issuePeekOperations.updateText("workspace", "project", issue.id, { name: "Edited" });
  assert.deepEqual(state.reconciled, [state.updateReceipt]);
});
