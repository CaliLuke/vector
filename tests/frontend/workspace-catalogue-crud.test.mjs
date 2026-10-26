import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl, submitForm, waitForSubmitCompletion } from "./helpers/native-forms.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import {
  catalogueSettings as settings,
  catalogueUser as user,
  fallbackWorkspace as fallback,
  removedWorkspace as removed,
  renderWorkspaceCatalogueRoute,
} from "./helpers/workspace-catalogue-fixture.mjs";

const state = { columns: {}, toasts: [] };
globalThis.workspaceCatalogueCrud = state;
globalThis.workspaceCatalogueCrudReact = React;
const workspaceDetailsSource = "apps/web/core/components/workspace/settings/workspace-details.tsx";
const confirmationSource = "apps/web/core/components/common/confirm-action-modal.tsx";
const formObserver = observeNativeForms([workspaceDetailsSource, confirmationSource]);
const stubs = {
  "@/components/core/modals/workspace-image-upload-modal": `export const WorkspaceImageUploadModal = () => null;`,
  "@/components/global/timezone-select": `const React=globalThis.workspaceCatalogueCrudReact;export const TimezoneSelect=({value,onChange,disabled})=>React.createElement("select",{value,disabled,onChange:event=>onChange(event.target.value)},React.createElement("option",{value:"UTC"},"UTC"),React.createElement("option",{value:"PST"},"PST"));`,
  "@/components/workspace/delete-workspace-section": `export const DeleteWorkspaceSection = () => null;`,
  "@/components/ui/loader/layouts/members-layout-loader": `export const MembersLayoutLoader = () => null;`,
  "@/components/workspace/confirm-workspace-member-remove": `const React=globalThis.workspaceCatalogueCrudReact;export const ConfirmWorkspaceMemberRemove=({onSubmit})=>React.createElement("button",{type:"button",onClick:()=>void onSubmit()},"Confirm leave");`,
  "@/components/workspace/settings/useMemberColumns": `export const useMemberColumns=()=>globalThis.workspaceCatalogueCrud.columns;`,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/button": `const React=globalThis.workspaceCatalogueCrudReact;export const Button=({loading,variant,size,...props})=>React.createElement("button",props,props.children);`,
  "@plane/propel/toast": `export const TOAST_TYPE={ERROR:"error",SUCCESS:"success"};export const setToast=value=>globalThis.workspaceCatalogueCrud.toasts.push(value);`,
  "@plane/ui": `export { ModalCore } from "${new URL("../../packages/ui/src/modals/modal-core.tsx", import.meta.url).href}";export { EModalPosition, EModalWidth } from "${new URL("../../packages/ui/src/modals/constants.ts", import.meta.url).href}";const React=globalThis.workspaceCatalogueCrudReact;export const Input=({hasError,...props})=>React.createElement("input",props);export const CustomSelect=Object.assign(({children,value,onChange,disabled})=>React.createElement("select",{value,onChange:event=>onChange(event.target.value),disabled},children),{Option:({children,value})=>React.createElement("option",{value},children)});export const Table=()=>null;`,
};
registerWebModuleHooks();
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});

async function loadDeleteWorkspaceHarness() {
  const { DeleteWorkspaceModal } = await import("../../apps/web/core/components/workspace/delete-workspace-modal.tsx");
  const { useWorkspaceBySlug } = await import("../../apps/web/core/hooks/use-workspaces.ts");
  function DeleteWorkspaceHarness() {
    const { data: workspace } = useWorkspaceBySlug("removed");
    return workspace
      ? React.createElement(DeleteWorkspaceModal, { isOpen: true, data: workspace, onClose: () => {} })
      : null;
  }
  return DeleteWorkspaceHarness;
}

const renderDeleteWorkspaceModal = (t) =>
  renderWorkspaceCatalogueRoute(t, { path: "/removed/settings/general", loadComponent: loadDeleteWorkspaceHarness });

async function fillDeleteWorkspaceConfirmation() {
  const form = formObserver.getForms(confirmationSource).at(-1);
  await changeFormControl(document.body.querySelector('[name="name"]'), removed.name);
  await changeFormControl(document.body.querySelector('[name="confirmation"]'), "delete my workspace");
  return form;
}

async function submitAndWait(environment, form) {
  let sawSubmitting = form.state.isSubmitting;
  let subscription;
  let timeout;
  const settled = new Promise((resolve) => {
    subscription = form.store.subscribe(() => {
      if (form.state.isSubmitting) sawSubmitting = true;
      else if (sawSubmitting) resolve();
    });
  });
  try {
    await submitForm(environment.container);
    await Promise.race([
      settled,
      new Promise((_, reject) => {
        timeout = setTimeout(() => reject(new Error("native workspace form submission did not settle")), 3000);
      }),
    ]);
  } finally {
    clearTimeout(timeout);
    subscription.unsubscribe();
  }
}

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((complete, fail) => {
    resolve = complete;
    reject = fail;
  });
  return { promise, resolve, reject };
}

function waitForRoute(router, pathname) {
  if (router.state.location.pathname === pathname) return Promise.resolve();
  return new Promise((resolve, reject) => {
    let timeout;
    const unsubscribe = router.subscribe("onResolved", () => {
      if (router.state.location.pathname === pathname) {
        clearTimeout(timeout);
        unsubscribe();
        resolve();
      }
    });
    timeout = setTimeout(() => {
      unsubscribe();
      reject(new Error(`router did not reach ${pathname}; current path is ${router.state.location.pathname}`));
    }, 3000);
  });
}

async function submitWorkspaceDeletion(t, deleteWorkspace) {
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const started = deferred();
  t.mock.method(WorkspaceService.prototype, "deleteWorkspace", (...args) => {
    started.resolve();
    return deleteWorkspace(...args);
  });
  const form = await fillDeleteWorkspaceConfirmation();
  await submitForm({ querySelector: () => document.body.querySelector("form") });
  await started.promise;
  return form;
}

await test("workspace details retain failed edits and reset to the canonical response after retry", async (t) => {
  const { environment, client } = await renderWorkspaceCatalogueRoute(t, {
    path: "/removed/settings/general",
    loadComponent: async () =>
      (await import("../../apps/web/core/components/workspace/settings/workspace-details.tsx")).WorkspaceDetails,
  });
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const { workspaceKeys } = await import("../../apps/web/core/queries/workspaces.ts");
  let fail = true;
  const updates = [];
  t.mock.method(WorkspaceService.prototype, "updateWorkspace", async (_slug, patch) => {
    updates.push(patch);
    if (fail) throw new Error("offline");
    return { ...removed, ...patch, name: "Canonical workspace name" };
  });
  const form = formObserver.getForms(workspaceDetailsSource).at(-1);
  const nameInput = environment.container.querySelector('[name="name"]');
  await changeFormControl(nameInput, "Edited draft");
  assert.equal(nameInput.value, "Edited draft");
  assert.equal(form.state.values.name, "Edited draft");
  await submitAndWait(environment, form);
  assert.equal(form.state.isSubmitSuccessful, false);
  assert.equal(
    form.state.values.name,
    "Edited draft",
    JSON.stringify({ updates, html: environment.container.innerHTML })
  );
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /Could not update/);
  fail = false;
  await submitAndWait(environment, form);
  assert.equal(form.state.isSubmitSuccessful, true);
  assert.equal(environment.container.querySelector('[name="name"]').value, "Canonical workspace name");
  assert.equal(client.getQueryData(workspaceKeys.list(user.id))[0].name, "Canonical workspace name");
});

await test("workspace details do not show stale success after the route changes during save", async (t) => {
  state.toasts = [];
  const { environment, router } = await renderWorkspaceCatalogueRoute(t, {
    path: "/removed/settings/general",
    loadComponent: async () =>
      (await import("../../apps/web/core/components/workspace/settings/workspace-details.tsx")).WorkspaceDetails,
  });
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const started = deferred();
  const result = deferred();
  t.mock.method(WorkspaceService.prototype, "updateWorkspace", async (_slug, patch) => {
    started.resolve();
    return result.promise.then(() => ({ ...removed, ...patch, name: "Saved on server" }));
  });
  const form = formObserver.getForms(workspaceDetailsSource).at(-1);
  const input = environment.container.querySelector('[name="name"]');
  await changeFormControl(input, "Draft before navigation");
  await submitForm(environment.container);
  await started.promise;
  await act(async () => {
    await router.navigate({ href: "/fallback/settings/general" });
  });
  await act(async () => {
    result.resolve();
    await waitForSubmitCompletion(form);
  });
  assert.equal(form.state.values.name, "Draft before navigation");
  assert.equal(
    state.toasts.some((toast) => toast.message === "Workspace updated successfully."),
    false
  );
});

await test("workspace deletion routes using refreshed settings and the remaining catalogue", async (t) => {
  state.toasts = [];
  const { client, router } = await renderDeleteWorkspaceModal(t);
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const userService = (await import("../../apps/web/core/services/user.service.ts")).default;
  const { workspaceKeys } = await import("../../apps/web/core/queries/workspaces.ts");
  const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  const deleted = [];
  t.mock.method(WorkspaceService.prototype, "deleteWorkspace", async (slug) => deleted.push(slug));
  t.mock.method(userService, "currentUserSettings", async () => settings);
  const form = await fillDeleteWorkspaceConfirmation();
  assert.equal(form.state.values.name, removed.name);
  assert.equal(form.state.values.confirmation, "delete my workspace");
  const routed = waitForRoute(router, "/fallback");
  await submitAndWait({ container: { querySelector: () => document.body.querySelector("form") } }, form);
  await routed.catch((error) => {
    throw new Error(
      `${error.message}; list=${JSON.stringify(client.getQueryData(workspaceKeys.list(user.id)))}; settings=${JSON.stringify(client.getQueryData(identityKeys.settings(user.id)))}; deleted=${JSON.stringify(deleted)}`
    );
  });
  assert.deepEqual(deleted, [removed.slug]);
  assert.deepEqual(client.getQueryData(workspaceKeys.list(user.id)), [fallback]);
  assert.equal(client.getQueryData(identityKeys.settings(user.id)).id, user.id);
});

await test("workspace deletion skips routing after the account session changes", async (t) => {
  state.toasts = [];
  const { client, router } = await renderDeleteWorkspaceModal(t);
  const { advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
  const result = deferred();
  const form = await submitWorkspaceDeletion(t, () => result.promise);
  const previousPath = router.state.location.pathname;
  await act(async () => {
    advanceIdentitySession(client);
    result.resolve();
    await waitForSubmitCompletion(form);
  });
  assert.equal(router.state.location.pathname, previousPath);
  assert.deepEqual(state.toasts, []);
});

await test("workspace deletion does not report a stale settings-refresh failure", async (t) => {
  state.toasts = [];
  const { client } = await renderDeleteWorkspaceModal(t);
  const userService = (await import("../../apps/web/core/services/user.service.ts")).default;
  const { advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
  const refreshStarted = deferred();
  const refreshResult = deferred();
  t.mock.method(userService, "currentUserSettings", async () => {
    refreshStarted.resolve();
    return refreshResult.promise;
  });
  const form = await submitWorkspaceDeletion(t, async () => {});
  await refreshStarted.promise;
  await act(async () => advanceIdentitySession(client));
  await act(async () => {
    refreshResult.reject(new Error("refresh failed"));
    await waitForSubmitCompletion(form);
  });
  assert.equal(
    state.toasts.some((toast) => /Unable to refresh your account settings/.test(toast.message)),
    false
  );
});

await test("leaving refreshes the workspace list and never routes back to the departed workspace", async (t) => {
  const leaveCalls = [];
  const service = (await import("../../apps/web/core/services/user.service.ts")).default;
  t.mock.method(service, "leaveWorkspace", async (slug) => {
    leaveCalls.push(slug);
  });
  state.columns = {
    columns: [{ key: "member" }],
    workspaceSlug: removed.slug,
    removeMemberModal: { member: { id: user.id, display_name: "Current user" } },
    setRemoveMemberModal: () => {},
  };
  const { environment, client, router } = await renderWorkspaceCatalogueRoute(t, {
    path: "/removed/settings/members",
    loadComponent: async () => {
      const { WorkspaceMembersListItem } =
        await import("../../apps/web/core/components/workspace/settings/members-list-item.tsx");
      return () => React.createElement(WorkspaceMembersListItem, { memberDetails: [] });
    },
  });
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const userService = (await import("../../apps/web/core/services/user.service.ts")).default;
  const { workspaceKeys } = await import("../../apps/web/core/queries/workspaces.ts");
  const refreshedLists = [];
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async () => {
    refreshedLists.push(true);
    return [fallback];
  });
  t.mock.method(userService, "currentUserSettings", async () => settings);
  const routed = waitForRoute(router, "/fallback");
  await act(async () => environment.container.querySelector("button").click());
  await routed.catch((error) => {
    throw new Error(
      `${error.message}; list=${JSON.stringify(client.getQueryData(workspaceKeys.list(user.id)))}; left=${JSON.stringify(leaveCalls)}; refreshed=${JSON.stringify(refreshedLists)}`
    );
  });
  assert.deepEqual(leaveCalls, [removed.slug]);
  assert.equal(refreshedLists.length, 1);
  assert.deepEqual(client.getQueryData(workspaceKeys.list(user.id)), [fallback]);
});

formObserver.restore();
