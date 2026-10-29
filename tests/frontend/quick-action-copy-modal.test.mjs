import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, createPermissionUiFixture } from "./helpers/permission-ui-fixture.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

globalThis.quickActionCopyReact = React;
globalThis.quickActionCopyState = {};

const stubs = {
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/icons": `const Icon=()=>null;export const LinkIcon=Icon,CopyIcon=Icon,NewTabIcon=Icon,EditIcon=Icon,ArchiveIcon=Icon,TrashIcon=Icon;`,
  "@plane/propel/icon-button": `export const IconButton=()=>null;`,
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:"success",ERROR:"error"};export const setToast=()=>{};`,
  "@plane/utils/string": "export const copyUrlToClipboard=async()=>{};",
  "@plane/utils/work-item": 'export const generateWorkItemLink=()=>"/work-item";',
  "lucide-react": `const Icon=()=>null;export const XCircle=Icon,ArchiveRestoreIcon=Icon,Ellipsis=Icon;`,
  "@/hooks/use-projects": `export const useWorkspaceProjectDetails=()=>({projects:[{id:"project-1",identifier:"PRJ"}]});`,
  "@/hooks/use-project-states":
    "export const useProjectStates=()=>({states:[],projectStates:[],getStateById:()=>undefined,getProjectStateIds:()=>[],getProjectDefaultStateId:()=>undefined,getProjectIntakeStateIds:()=>[],getIntakeStateById:()=>undefined,accessPending:false,accessError:false,accessDenied:false,retryAccess:async()=>{},refetch:async()=>({})});",
  "@/hooks/use-permissions": `export const useProjectPermissions=()=>({can:()=>true});`,
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (
      context.parentURL?.includes("quick-action-dropdowns/issue-detail.tsx") ||
      context.parentURL?.includes("quick-action-dropdowns/cycle-issue.tsx") ||
      context.parentURL?.includes("quick-action-dropdowns/issue-quick-action-modals.tsx")
    ) {
      if (specifier === "@/components/common/quick-action-menu") {
        return {
          shortCircuit: true,
          url: `data:text/javascript,${encodeURIComponent(
            `const React=globalThis.quickActionCopyReact;export const QuickActionMenu=props=>{globalThis.quickActionCopyState.items=props.items;return React.createElement("div")}`
          )}`,
        };
      }
      if (["../../archive-issue-modal", "../../delete-issue-modal"].includes(specifier)) {
        return {
          shortCircuit: true,
          url: `data:text/javascript,${encodeURIComponent(`export const ${specifier.includes("archive") ? "ArchiveIssueModal" : "DeleteIssueModal"}=()=>null`)}`,
        };
      }
      if (specifier === "../../issue-modal/modal") {
        return {
          shortCircuit: true,
          url: `data:text/javascript,${encodeURIComponent(
            `export const CreateUpdateIssueModal=props=>{globalThis.quickActionCopyState.modal=props;return null}`
          )}`,
        };
      }
    }
    return resolvePackageStub(specifier, stubs, context, nextResolve);
  },
});

const issue = {
  id: "issue-1",
  project_id: "project-1",
  workspace_id: "workspace-1",
  state_id: "state-1",
  name: "Original",
};

async function createCopyFixture(t) {
  return createPermissionUiFixture(t, {
    path: "/workspace/projects/project-1/issues/PRJ-1",
    workspaceSlug: "workspace",
    workspaceRole: 15,
    projectRoles: { "project-1": 20 },
  });
}

async function renderQuickActions(environment, props = {}) {
  const { WorkItemDetailQuickActions } =
    await import("../../apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx");
  await environment.render(
    React.createElement(WorkItemDetailQuickActions, {
      issue,
      readOnly: false,
      ...props,
    })
  );
}

async function renderCycleQuickActions(environment) {
  const { CycleIssueQuickActions } =
    await import("../../apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/cycle-issue.tsx");
  await environment.render(
    React.createElement(CycleIssueQuickActions, {
      issue,
      readOnly: false,
      handleUpdate: async (data) => globalThis.quickActionCopyState.updates.push(data),
    })
  );
}

async function chooseCopyAction() {
  const copyItem = globalThis.quickActionCopyState.items.find((item) => item.key === "make-a-copy");
  assert.ok(copyItem);
  await React.act(async () => copyItem.action());
}

await test("copy uses the local modal and does not mark edit tracking", async (t) => {
  globalThis.quickActionCopyState = {};
  let editTrackingCalls = 0;
  const environment = await createCopyFixture(t);

  await renderQuickActions(environment, {
    toggleEditIssueModal: () => editTrackingCalls++,
  });
  await chooseCopyAction();

  assert.equal(globalThis.quickActionCopyState.modal.isOpen, true);
  assert.equal(globalThis.quickActionCopyState.modal.data.name, "Original (copy)");
  assert.equal(globalThis.quickActionCopyState.modal.data.id, undefined);
  assert.equal(globalThis.quickActionCopyState.modal.data.sourceIssueId, issue.id);
  assert.equal(editTrackingCalls, 0);
});

await test("peek copy notifies duplicate tracking and clears it when the local modal closes", async (t) => {
  globalThis.quickActionCopyState = {};
  const tracking = [];
  const environment = await createCopyFixture(t);

  await renderQuickActions(environment, {
    toggleEditIssueModal: (open) => tracking.push(["edit", open]),
    toggleDuplicateIssueModal: (open) => tracking.push(["duplicate", open]),
  });
  await chooseCopyAction();

  assert.deepEqual(tracking, [["duplicate", true]]);
  assert.equal(globalThis.quickActionCopyState.modal.isOpen, true);
  assert.equal(globalThis.quickActionCopyState.modal.data.name, "Original (copy)");
  assert.equal(globalThis.quickActionCopyState.modal.data.id, undefined);
  assert.equal(globalThis.quickActionCopyState.modal.data.sourceIssueId, issue.id);

  await React.act(async () => globalThis.quickActionCopyState.modal.onClose());
  assert.deepEqual(tracking, [
    ["duplicate", true],
    ["duplicate", false],
  ]);
});

await test("copy after closing edit clears the edit selection before submit", async (t) => {
  globalThis.quickActionCopyState = { updates: [] };
  const environment = await createCopyFixture(t);

  await renderCycleQuickActions(environment);
  const editItem = globalThis.quickActionCopyState.items.find((item) => item.key === "edit");
  assert.ok(editItem);
  await React.act(async () => editItem.action());
  assert.equal(globalThis.quickActionCopyState.modal.data.id, issue.id);

  await React.act(async () => globalThis.quickActionCopyState.modal.onClose());
  await chooseCopyAction();

  const copyData = globalThis.quickActionCopyState.modal.data;
  assert.equal(copyData.id, undefined);
  assert.equal(copyData.name, "Original (copy)");
  assert.equal(copyData.sourceIssueId, issue.id);
  await React.act(async () => globalThis.quickActionCopyState.modal.onSubmit(copyData));
  assert.deepEqual(globalThis.quickActionCopyState.updates, []);
});
