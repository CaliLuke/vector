import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { React, act, createPermissionUiFixture, settleQuery } from "./helpers/permission-ui-fixture.mjs";

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

const { EUserWorkspaceRoles } = await import("../../packages/types/dist/index.mjs");
const { IssueLabelService } = await import("../../apps/web/core/services/issue/issue_label.service.ts");
const { labelKeys } = await import("../../apps/web/core/queries/labels.ts");
const alpha = { parent: null, id: "label-a", name: "Alpha", project_id: "p1", color: "#123456", sort_order: 1 };
const beta = { parent: null, id: "label-b", name: "Beta", project_id: "p1", color: "#654321", sort_order: 2 };
const scope = {
  accountId: "alice",
  workspaceSlug: "acme",
  projectId: "p1",
  workspaceRole: EUserWorkspaceRoles.MEMBER,
  membershipRevision: "2026-10-24T00:00:00Z",
  projectRole: 20,
  projectMembershipRole: 20,
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
    await import("../../apps/web/core/components/labels/project-setting-label-list.tsx");
  const { DeleteLabelModal } = await import("../../apps/web/core/components/labels/delete-label-modal.tsx");
  const { CreateUpdateLabelInline } =
    await import("../../apps/web/core/components/labels/create-update-label-inline.tsx");
  return { ...view, ProjectSettingsLabelList, DeleteLabelModal, CreateUpdateLabelInline };
}

await test("failed native label loading exposes Retry and recovers through the real settings consumer", async (t) => {
  let offline = true;
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async () => {
    if (offline) throw new Error("offline");
    return [];
  });
  const view = await fixture(t);
  await view.render(React.createElement(view.ProjectSettingsLabelList));
  assert.ok(view.environment.container.querySelector('[role="alert"]'));
  assert.equal(view.environment.container.querySelector('[data-testid="label-loader"]'), null);
  offline = false;
  await click(button(view.environment.container, "Retry"));
  assert.match(view.environment.container.textContent, /No labels/);
  assert.equal(view.environment.container.querySelector('[role="alert"]'), null);
  assert.deepEqual(view.client.getQueryData(labelKeys.project(scope)), []);
});

await test("accepted deletion of A cannot close a reopened B dialog", async (t) => {
  const pending = deferred();
  const started = deferred();
  t.mock.method(IssueLabelService.prototype, "deleteIssueLabel", async (_workspace, _project, id) => {
    started.resolve(id);
    await pending.promise;
  });
  const view = await fixture(t, [alpha, beta]);
  await view.render(React.createElement(view.ProjectSettingsLabelList));
  await click(button(view.environment.container, "Delete Alpha"));
  await click(button(view.environment.container, "Delete"));
  assert.equal(await started.promise, alpha.id);
  await click(button(view.environment.container, "Cancel"));
  await click(button(view.environment.container, "Delete Beta"));
  assert.match(view.environment.container.querySelector('[role="dialog"]').textContent, /Beta/);
  assert.equal(
    (button(view.environment.container, "Delete") ?? button(view.environment.container, "Deleting"))?.disabled,
    false,
    "B owns a fresh enabled delete action while A is pending"
  );
  await act(async () => pending.resolve());
  await settleQuery();
  assert.match(view.environment.container.querySelector('[role="dialog"]')?.textContent ?? "", /Beta/);
  assert.equal(button(view.environment.container, "Delete")?.disabled, false);
  assert.deepEqual(
    view.client.getQueryData(labelKeys.project(scope)).map((label) => label.id),
    [beta.id]
  );
});

function holdCreation(t) {
  const pending = deferred();
  const started = deferred();
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async () => []);
  t.mock.method(IssueLabelService.prototype, "createIssueLabel", async (_workspace, projectId, data) => {
    started.resolve({ projectId, data });
    await pending.promise;
    return { ...alpha, name: data.name };
  });
  return { pending, started };
}
async function submitName(view, name) {
  await changeName(view.environment.container, name);
  await act(async () =>
    view.environment.container
      .querySelector("form")
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
}

await test("an inline new-label draft resets on project change and accepted old work cannot close its replacement", async (t) => {
  const { pending, started } = holdCreation(t);
  const view = await fixture(t, []);
  await view.render(React.createElement(view.ProjectSettingsLabelList));
  await click(button(view.environment.container, "common.add_label"));
  await changeName(view.environment.container, "Draft A");
  await act(async () =>
    view.environment.container
      .querySelector("form")
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
  assert.equal((await started.promise).projectId, "p1");
  await act(async () => view.router.navigate({ to: "/acme/settings/projects/p2/labels" }));
  await settleQuery();
  assert.equal(view.environment.container.querySelector('input[name="name"]')?.value ?? "", "");
  if (!view.environment.container.querySelector("form"))
    await click(button(view.environment.container, "common.add_label"));
  await changeName(view.environment.container, "Draft B");
  assert.equal(view.environment.container.querySelector('input[name="name"]').disabled, false);
  await act(async () => pending.resolve());
  await settleQuery();
  assert.equal(view.environment.container.querySelector('input[name="name"]')?.value, "Draft B");
  assert.equal(view.client.getQueryData(labelKeys.project(scope))[0].name, "Draft A");
  assert.deepEqual(view.client.getQueryData(labelKeys.project({ ...scope, projectId: "p2" })), []);
});

await test("same-project inline replacement survives an accepted submission from its closed predecessor", async (t) => {
  const { pending, started } = holdCreation(t);
  const view = await fixture(t, []);
  await view.render(React.createElement(React.StrictMode, null, React.createElement(view.ProjectSettingsLabelList)));
  await click(button(view.environment.container, "common.add_label"));
  await submitName(view, "Draft A");
  assert.equal((await started.promise).data.name, "Draft A");
  await click(button(view.environment.container, "common.add_label"));
  assert.equal(view.environment.container.querySelector('input[name="name"]').value, "");
  await changeName(view.environment.container, "Draft B");
  assert.equal(view.environment.container.querySelector('input[name="name"]').disabled, false);
  await act(async () => pending.resolve());
  await settleQuery();
  assert.equal(view.environment.container.querySelector('input[name="name"]')?.value, "Draft B");
  assert.equal(view.client.getQueryData(labelKeys.project(scope))[0].name, "Draft A");
});

await test("returning to the original project does not revive an earlier inline completion", async (t) => {
  const { pending, started } = holdCreation(t);
  const view = await fixture(t, []);
  await view.render(React.createElement(view.ProjectSettingsLabelList));
  await click(button(view.environment.container, "common.add_label"));
  await submitName(view, "Draft A");
  assert.equal((await started.promise).projectId, "p1");
  await act(async () => view.router.navigate({ to: "/acme/settings/projects/p2/labels" }));
  await settleQuery();
  await act(async () => view.router.navigate({ to: "/acme/settings/projects/p1/labels" }));
  await settleQuery();
  await click(button(view.environment.container, "common.add_label"));
  await changeName(view.environment.container, "Draft B");
  assert.equal(view.environment.container.querySelector('input[name="name"]').disabled, false);
  await act(async () => pending.resolve());
  await settleQuery();
  assert.equal(view.environment.container.querySelector('input[name="name"]')?.value, "Draft B");
  assert.equal(view.client.getQueryData(labelKeys.project(scope))[0].name, "Draft A");
});

await test("rejected work from an unmounted inline draft cannot toast into its replacement", async (t) => {
  const { pending, started } = holdCreation(t);
  const view = await fixture(t, []);
  await view.render(React.createElement(React.StrictMode, null, React.createElement(view.ProjectSettingsLabelList)));
  await click(button(view.environment.container, "common.add_label"));
  await submitName(view, "Draft A");
  assert.equal((await started.promise).data.name, "Draft A");
  await click(button(view.environment.container, "common.add_label"));
  await changeName(view.environment.container, "Draft B");
  await act(async () => pending.reject(new Error("late failure")));
  await settleQuery();
  assert.equal(view.environment.container.querySelector('input[name="name"]')?.value, "Draft B");
  assert.deepEqual(globalThis.nativeLabelsUiToasts, []);
  assert.deepEqual(view.client.getQueryData(labelKeys.project(scope)), []);
});

await test("a failed current inline draft remains editable and exposes its failure after effect replay", async (t) => {
  const { pending, started } = holdCreation(t);
  const view = await fixture(t, []);
  await view.render(React.createElement(React.StrictMode, null, React.createElement(view.ProjectSettingsLabelList)));
  await click(button(view.environment.container, "common.add_label"));
  await submitName(view, "Draft A");
  assert.equal((await started.promise).data.name, "Draft A");
  await act(async () => pending.reject(new Error("save failed")));
  await settleQuery();
  const input = view.environment.container.querySelector('input[name="name"]');
  assert.equal(input?.value, "Draft A");
  assert.equal(input.disabled, false);
  assert.equal(globalThis.nativeLabelsUiToasts.length, 1);
  assert.equal(globalThis.nativeLabelsUiToasts[0].type, "error");
  assert.deepEqual(view.client.getQueryData(labelKeys.project(scope)), []);
});

for (const unmountBeforeFailure of [false, true]) {
  void test(`remove-from-group feedback belongs to its ${unmountBeforeFailure ? "unmounted" : "current"} label item`, async (t) => {
    const view = await fixture(t, [alpha]);
    const { ProjectSettingLabelItem } =
      await import("../../apps/web/core/components/labels/project-setting-label-item.tsx");
    const pending = deferred();
    const updateLabel = t.mock.fn(() => pending.promise);
    t.mock.method(IssueLabelService.prototype, "patchIssueLabel", () => pending.promise);
    const props = {
      label: { ...alpha, parent: "parent-label" },
      isChild: true,
      isLastChild: true,
      isEditable: true,
      handleLabelDelete() {},
      setIsUpdating() {},
      onDrop() {},
      labelOperationsCallbacks: { createLabel: async () => alpha, updateLabel },
    };
    await view.render(React.createElement(React.StrictMode, null, React.createElement(ProjectSettingLabelItem, props)));
    await click(button(view.environment.container, "Remove from group"));
    assert.deepEqual(
      updateLabel.mock.calls.map((call) => call.arguments),
      [[alpha.id, { parent: null }]],
      "the settings owner receives the update"
    );
    if (unmountBeforeFailure) await view.render(React.createElement("p", null, "Replacement settings"));
    await act(async () => pending.reject(new Error("save failed")));
    await settleQuery();
    assert.equal(globalThis.nativeLabelsUiToasts.length, unmountBeforeFailure ? 0 : 1);
    if (!unmountBeforeFailure) assert.equal(globalThis.nativeLabelsUiToasts[0].type, "error");
  });
}
