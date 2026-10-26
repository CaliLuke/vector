import assert from "node:assert/strict";
import test from "node:test";
import { EUserPermissions, EUserWorkspaceRoles, EUserProjectRoles } from "../../packages/types/dist/index.mjs";
import { React, act, createPermissionUiFixture, registerPermissionUiStubs } from "./helpers/permission-ui-fixture.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";

globalThis.deleteIssueModalPermissionTest = { modal: undefined, toasts: [], submissions: 0, closes: 0 };
const reactBinding = "const React=globalThis.deleteIssueModalPermissionTest.React;";
globalThis.deleteIssueModalPermissionTest.React = React;
registerPermissionUiStubs("/delete-issue-modal.tsx", {
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/ui": `${reactBinding}
    export const AlertModalCore=props=>{
      globalThis.deleteIssueModalPermissionTest.modal=props;
      return React.createElement('div',{role:'dialog'},props.content,
        React.createElement('button',{id:'cancel-delete',onClick:props.handleClose},'Cancel'),
        React.createElement('button',{id:'confirm-delete',disabled:props.isSubmitting,onClick:props.handleSubmit},'Delete'));
    };`,
});
registerPermissionUiStubs("/workspace-draft/delete-modal.tsx", {
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/ui": `${reactBinding}
    export const AlertModalCore=props=>{
      globalThis.deleteIssueModalPermissionTest.modal=props;
      return React.createElement('div',{role:'dialog'},props.content,
        React.createElement('button',{id:'cancel-delete',onClick:props.handleClose},'Cancel'),
        React.createElement('button',{id:'confirm-delete',disabled:props.isSubmitting,onClick:props.handleSubmit},'Delete'));
    };`,
});
registerPermissionUiStubs("/archive-issue-modal.tsx", {
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/button": `${reactBinding}
    export const Button=props=>React.createElement('button',{
      id:props.variant==='primary'?'archive-issue':'cancel-archive',
      disabled:props.disabled||props.loading,onClick:props.onClick
    },props.children);`,
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:'success',ERROR:'error'};
    export const setToast=value=>globalThis.deleteIssueModalPermissionTest.toasts.push(value);`,
  "@plane/ui": `${reactBinding}
    export const EModalPosition={CENTER:'center'};export const EModalWidth={LG:'large'};
    export const ModalCore=props=>React.createElement('div',{role:'dialog'},props.children);`,
});
registerPermissionUiStubs("/issue-delete-modal-feedback.ts", {
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:'success',ERROR:'error'};
    export const setToast=value=>globalThis.deleteIssueModalPermissionTest.toasts.push(value);`,
});

const { DeleteIssueModal } = await import("../../apps/web/core/components/issues/delete-issue-modal.tsx");
const { ArchiveIssueModal } = await import("../../apps/web/core/components/issues/archive-issue-modal.tsx");
const { WorkspaceDraftIssueDeleteIssueModal } =
  await import("../../apps/web/core/components/issues/workspace-draft/delete-modal.tsx");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const { waitForQueryState } = await import("./helpers/query-state.mjs");
const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");

const issue = {
  id: "issue-1",
  project_id: "project-1",
  sequence_id: 12,
  created_by: "alice",
};

function modalElement({ data, handleClose, onSubmit } = {}) {
  return React.createElement(DeleteIssueModal, {
    isOpen: true,
    handleClose: handleClose ?? (() => globalThis.deleteIssueModalPermissionTest.closes++),
    onSubmit: onSubmit ?? (async () => globalThis.deleteIssueModalPermissionTest.submissions++),
    data,
  });
}

async function renderModal(t, options = {}) {
  globalThis.deleteIssueModalPermissionTest.modal = undefined;
  globalThis.deleteIssueModalPermissionTest.toasts = [];
  globalThis.deleteIssueModalPermissionTest.submissions = 0;
  globalThis.deleteIssueModalPermissionTest.closes = 0;
  t.mock.method(ProjectService.prototype, "getProjects", async () => [
    { id: "project-1", identifier: "PRJ", member_role: EUserProjectRoles.ADMIN },
    { id: "legacy-project", identifier: "OLD", member_role: EUserProjectRoles.ADMIN },
  ]);
  return createPermissionUiFixture(t, {
    path: "/acme/projects/project-1/issues/",
    workspaceRole: EUserWorkspaceRoles.ADMIN,
    ...options,
  });
}

async function confirmDelete(fixture) {
  await act(async () => {
    fixture.environment.container.querySelector("#confirm-delete").click();
  });
}

async function pendingDeletion(t, handleClose) {
  const fixture = await renderModal(t, {
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { "project-1": EUserProjectRoles.MEMBER },
  });
  const pending = Promise.withResolvers();
  await fixture.render(modalElement({ data: issue, handleClose, onSubmit: () => pending.promise }));
  await act(async () => fixture.environment.container.querySelector("#confirm-delete").click());
  return { fixture, resolveDelete: pending.resolve };
}

await test("modal keeps hook order when an issue appears after an empty render", async (t) => {
  const fixture = await renderModal(t, {
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { "project-1": EUserProjectRoles.MEMBER },
  });
  await fixture.render(modalElement());
  assert.equal(fixture.environment.container.querySelector('[role="dialog"]'), null);

  await fixture.render(modalElement({ data: issue }));
  assert.ok(fixture.environment.container.querySelector('[role="dialog"]'));
  assert.match(fixture.environment.container.textContent, /PRJ-12/);

  await confirmDelete(fixture);
  assert.equal(globalThis.deleteIssueModalPermissionTest.submissions, 1);
});

await test("issue creator is denied when the issue project has no membership despite cached metadata admin", async (t) => {
  const fixture = await renderModal(t, {
    projectRoles: { "legacy-project": EUserPermissions.ADMIN },
  });
  await fixture.render(modalElement({ data: issue }));

  await confirmDelete(fixture);
  assert.equal(globalThis.deleteIssueModalPermissionTest.submissions, 0);
  assert.equal(globalThis.deleteIssueModalPermissionTest.closes, 1);
  assert.equal(globalThis.deleteIssueModalPermissionTest.toasts.at(-1)?.type, "error");
});

await test("issue creator is denied when the cached project role read errors", async (t) => {
  const fixture = await renderModal(t, {
    projectRoles: { "project-1": EUserPermissions.ADMIN },
    rolesRead: async () => {
      throw new Error("roles unavailable");
    },
  });
  const key = permissionKeys.workspaceProjectRoles("alice", "acme", EUserWorkspaceRoles.ADMIN, "2026-10-24T00:00:00Z");
  await fixture.client.invalidateQueries({ queryKey: key, exact: true });
  await fixture.render(modalElement({ data: issue }));
  await waitForQueryState(fixture.client, key, (state) => state?.status === "error");

  await confirmDelete(fixture);
  assert.equal(globalThis.deleteIssueModalPermissionTest.submissions, 0);
  assert.equal(globalThis.deleteIssueModalPermissionTest.closes, 0);
  assert.equal(globalThis.deleteIssueModalPermissionTest.toasts.length, 0);
  assert.match(fixture.environment.container.textContent, /Retry/);
});

await test("pending delete cannot dismiss, and an old item completion cannot close or toast over its replacement", async (t) => {
  let oldCloses = 0;
  const { fixture, resolveDelete } = await pendingDeletion(t, () => oldCloses++);
  assert.equal(fixture.environment.container.querySelector("#confirm-delete").disabled, true);
  await act(async () => fixture.environment.container.querySelector("#cancel-delete").click());
  assert.equal(oldCloses, 0, "pending confirmation cannot dismiss its dialog");

  const replacement = { ...issue, id: "issue-2", sequence_id: 13 };
  await fixture.render(modalElement({ data: replacement }));
  await act(async () => {
    resolveDelete();
    await Promise.resolve();
  });
  assert.equal(oldCloses, 0);
  assert.equal(globalThis.deleteIssueModalPermissionTest.toasts.length, 0);
  assert.ok(fixture.environment.container.querySelector('[role="dialog"]'));
  assert.equal(globalThis.deleteIssueModalPermissionTest.modal.isSubmitting, false);
});

await test("a completion from a replaced account session does not emit a toast or close the new dialog", async (t) => {
  let oldCloses = 0;
  const { fixture, resolveDelete } = await pendingDeletion(t, () => oldCloses++);

  await act(async () => {
    fixture.client.setQueryData(identityKeys.currentUser(), { id: "bob" });
    await seedPermissionQueries(fixture.client, {
      userId: "bob",
      workspaceSlug: "acme",
      workspaceRole: EUserWorkspaceRoles.MEMBER,
      projectRoles: { "project-1": EUserProjectRoles.MEMBER },
    });
    advanceIdentitySession(fixture.client);
  });
  await fixture.render(modalElement({ data: issue }));
  await act(async () => {
    resolveDelete();
    await Promise.resolve();
  });

  assert.equal(oldCloses, 0);
  assert.equal(globalThis.deleteIssueModalPermissionTest.toasts.length, 0);
  assert.ok(fixture.environment.container.querySelector('[role="dialog"]'));
});

await test("missing actor and creator identifiers never authorize deletion by equality", async (t) => {
  const fixture = await renderModal(t, {
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { "project-1": EUserProjectRoles.MEMBER },
  });
  await act(async () => fixture.client.setQueryData(identityKeys.currentUser(), {}));
  await fixture.render(modalElement({ data: { ...issue, created_by: undefined } }));

  assert.equal(fixture.environment.container.querySelector("#confirm-delete").disabled, true);
  await confirmDelete(fixture);
  assert.equal(globalThis.deleteIssueModalPermissionTest.submissions, 0);
  assert.equal(globalThis.deleteIssueModalPermissionTest.closes, 0);
});

await test("workspace draft deletion stays scoped to its mounted draft while pending", async (t) => {
  const fixture = await renderModal(t, { workspaceRole: EUserWorkspaceRoles.MEMBER });
  let resolveDelete;
  let oldCloses = 0;
  const draft = { id: "draft-1", created_by: "alice" };
  const element = (data, handleClose) =>
    React.createElement(WorkspaceDraftIssueDeleteIssueModal, {
      isOpen: true,
      data,
      handleClose,
      onSubmit: () => new Promise((resolve) => (resolveDelete = resolve)),
    });
  await fixture.render(element(draft, () => oldCloses++));
  await act(async () => fixture.environment.container.querySelector("#confirm-delete").click());
  await act(async () => fixture.environment.container.querySelector("#cancel-delete").click());
  assert.equal(oldCloses, 0);

  await fixture.render(element({ ...draft, id: "draft-2" }, () => globalThis.deleteIssueModalPermissionTest.closes++));
  await act(async () => {
    resolveDelete();
    await Promise.resolve();
  });
  assert.equal(oldCloses, 0);
  assert.equal(globalThis.deleteIssueModalPermissionTest.toasts.length, 0);
  assert.ok(fixture.environment.container.querySelector('[role="dialog"]'));
});

await test("archive confirmation cannot close a replacement item after the old write completes", async (t) => {
  const fixture = await renderModal(t, {
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { "project-1": EUserProjectRoles.MEMBER },
  });
  let resolveArchive;
  let oldCloses = 0;
  const element = (data, handleClose) =>
    React.createElement(ArchiveIssueModal, {
      isOpen: true,
      data,
      handleClose,
      onSubmit: () => new Promise((resolve) => (resolveArchive = resolve)),
    });
  await fixture.render(element(issue, () => oldCloses++));
  await act(async () => fixture.environment.container.querySelector("#archive-issue").click());
  assert.equal(fixture.environment.container.querySelector("#archive-issue").disabled, true);
  await act(async () => fixture.environment.container.querySelector("#cancel-archive").click());
  assert.equal(oldCloses, 0);

  await fixture.render(
    element({ ...issue, id: "issue-2", sequence_id: 13 }, () => globalThis.deleteIssueModalPermissionTest.closes++)
  );
  await act(async () => {
    resolveArchive();
    await Promise.resolve();
  });
  assert.equal(oldCloses, 0);
  assert.equal(globalThis.deleteIssueModalPermissionTest.toasts.length, 0);
  assert.ok(fixture.environment.container.querySelector('[role="dialog"]'));
});
