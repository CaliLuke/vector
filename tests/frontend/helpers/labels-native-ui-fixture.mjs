import * as ContractEnums from "../../../packages/types/dist/index.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import { React, act, createPermissionUiFixture, settleQuery } from "./permission-ui-fixture.mjs";

globalThis.nativeLabelsUiReact = React;
globalThis.nativeLabelsUiToasts = [];
const ui = `const React = globalThis.nativeLabelsUiReact;
export const Input = ({hasError, ...props}) => React.createElement('input', props);
export const Loader = Object.assign(({children}) => React.createElement('div', {'data-testid':'label-loader'}, children), {Item: () => null});
export function AlertModalCore({isOpen, handleClose, handleSubmit, isSubmitting, content}) {
 return isOpen ? React.createElement('div', {role:'dialog'}, content,
  React.createElement('button', {onClick:handleClose}, 'Cancel'),
  React.createElement('button', {onClick:handleSubmit, disabled:isSubmitting}, isSubmitting ? 'Deleting' : 'Delete')) : null;
}`;
const stubs = {
  "./label-drag-n-drop-HOC": "export const LabelDndHOC = ({children}) => children(false, false, {current:null});",
  "./label-block/label-item-block": `const React = globalThis.nativeLabelsUiReact;
export const LabelItemBlock = ({label, customMenuItems}) => React.createElement('div', null,
 customMenuItems.filter(item => item.isVisible).map(item => React.createElement('button', {key:item.key, onClick:()=>item.onClick(label)},item.text)));`,
  "@plane/ui": ui,
  "@plane/i18n": "export const useTranslation = () => ({t: key => key});",
  "@plane/propel/empty-state": `const React=globalThis.nativeLabelsUiReact; export const EmptyStateCompact = () => React.createElement('p', null, 'No labels');`,
  "@plane/propel/toast":
    "export const TOAST_TYPE={ERROR:'error',SUCCESS:'success'}; export const setToast=toast=>globalThis.nativeLabelsUiToasts.push(toast);",
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    const parent = context.parentURL ?? "";
    if (parent.includes("/components/labels/") && stubs[specifier])
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(stubs[specifier])}` };
    if (parent.endsWith("/components/labels/project-setting-label-list.tsx") && specifier === "@/components/labels") {
      const inline = nextResolve("./create-update-label-inline", context).url;
      const deletion = nextResolve("./delete-label-modal", context).url;
      const source = `const React=globalThis.nativeLabelsUiReact;
export {CreateUpdateLabelInline} from '${inline}';
export {DeleteLabelModal} from '${deletion}';
export const ProjectSettingLabelItem=({label,handleLabelDelete})=>React.createElement('button',{onClick:()=>handleLabelDelete(label)},'Delete '+label.name);
export const ProjectSettingLabelGroup=ProjectSettingLabelItem;`;
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(source)}` };
    }
    return nextResolve(specifier, context);
  },
});

const { EUserWorkspaceRoles } = await import("../../../packages/types/dist/index.mjs");
const { IssueLabelService } = await import("../../../apps/web/core/services/issue/issue_label.service.ts");
const { labelKeys } = await import("../../../apps/web/core/queries/labels.ts");
const alpha = { parent: null, id: "label-a", name: "Alpha", project_id: "p1", color: "#123456", sort_order: 1 };
const beta = { parent: null, id: "label-b", name: "Beta", project_id: "p1", color: "#654321", sort_order: 2 };
const scope = {
  accountId: "alice",
  workspaceSlug: "acme",
  projectId: "p1",
  workspaceRole: EUserWorkspaceRoles.MEMBER,
  membershipRevision: "2026-10-24T00:00:00Z",
  projectRole: ContractEnums.EUserPermissions.ADMIN,
  projectMembershipRole: ContractEnums.EUserPermissions.ADMIN,
  sessionGeneration: 0,
};
function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((complete, fail) => {
    resolve = complete;
    reject = fail;
  });
  return { promise, resolve, reject };
}
function button(container, text) {
  return [...container.querySelectorAll("button")].find((node) => node.textContent === text);
}
async function click(node) {
  assert.ok(node, "the requested control is rendered");
  await act(async () => node.click());
  await settleQuery();
}
async function changeName(container, value) {
  const input = container.querySelector('input[name="name"]');
  assert.ok(input, "the label form is open");
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
async function fixture(t, initialLabels) {
  globalThis.nativeLabelsUiToasts.length = 0;
  const view = await createPermissionUiFixture(t, {
    path: "/acme/settings/projects/p1/labels",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { p1: 20, p2: 20 },
  });
  if (initialLabels !== undefined) view.client.setQueryData(labelKeys.project(scope), initialLabels);
  const { ProjectSettingsLabelList } =
    await import("../../../apps/web/core/components/labels/project-setting-label-list.tsx");
  const { DeleteLabelModal } = await import("../../../apps/web/core/components/labels/delete-label-modal.tsx");
  const { CreateUpdateLabelInline } =
    await import("../../../apps/web/core/components/labels/create-update-label-inline.tsx");
  return { ...view, ProjectSettingsLabelList, DeleteLabelModal, CreateUpdateLabelInline };
}

export {
  React,
  act,
  settleQuery,
  fixture,
  button,
  click,
  changeName,
  deferred,
  scope,
  alpha,
  beta,
  labelKeys,
  IssueLabelService,
};
