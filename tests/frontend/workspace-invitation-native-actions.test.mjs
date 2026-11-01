import assert from "node:assert/strict";
import test from "node:test";
import {
  React,
  act,
  settleQuery,
  createPermissionUiFixture,
  registerPermissionUiStubs,
} from "./helpers/permission-ui-fixture.mjs";
import { changeFormControl } from "./helpers/native-forms.mjs";

globalThis.invitationActions = { React, toasts: [] };
const stubs = {
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:'success',ERROR:'error'};export const setToast=toast=>globalThis.invitationActions.toasts.push(toast);`,
  "@plane/ui": `const React=globalThis.invitationActions.React;
    export const EModalPosition={CENTER:'center'},EModalWidth={XXL:'xxl'};
    export const ModalCore=({children,isOpen})=>isOpen?React.createElement('section',{role:'dialog'},children):null;
    export const CustomSelect=Object.assign(({children,value,onChange,disabled})=>React.createElement('select',{'aria-label':'Invitation role',value,onChange:event=>onChange(Number(event.target.value)),disabled},children),{Option:({value,children})=>React.createElement('option',{value},children)});
    export const CustomMenu=({children})=>children;CustomMenu.ActionItem=({item})=>item.shouldRender?React.createElement('button',{type:'button',onClick:item.action},item.title):null;`,
  "@plane/propel/button": `const React=globalThis.invitationActions.React;export const Button=({children,loading,variant,size,...props})=>React.createElement('button',{...props,disabled:props.disabled||loading},children);`,
};
registerPermissionUiStubs("/invitations-list-item.tsx", stubs);
registerPermissionUiStubs("/confirm-workspace-member-remove.tsx", stubs);
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { useWorkspaceInvitations } = await import("../../apps/web/core/hooks/use-member-roster-mutations.ts");
const invitation = { id: "invite", email: "new@example.test", role: 15, invite_link: "/invite/new" };

await test("native invitation role and removal actions preserve failed state and update the shared list on retry", async (t) => {
  const view = await createPermissionUiFixture(t, { path: "/acme/settings/members", workspaceRole: 20 });
  const { WorkspaceInvitationsListItem } =
    await import("../../apps/web/core/components/workspace/settings/invitations-list-item.tsx");
  globalThis.invitationActions.toasts = [];
  t.mock.method(WorkspaceService.prototype, "workspaceInvitations", async () => [invitation]);
  let roleFailure = true,
    deleteFailure = true;
  const updates = [],
    removals = [];
  t.mock.method(WorkspaceService.prototype, "updateWorkspaceInvitation", async (workspace, id, patch) => {
    updates.push({ workspace, id, patch });
    if (roleFailure) throw { error: "Role change refused" };
    return { ...invitation, ...patch };
  });
  t.mock.method(WorkspaceService.prototype, "deleteWorkspaceInvitations", async (workspace, id) => {
    removals.push({ workspace, id });
    if (deleteFailure) throw new Error("Removal offline");
  });
  function List() {
    const rows = useWorkspaceInvitations("acme");
    return React.createElement(
      React.Fragment,
      null,
      ...(rows.data ?? []).map((row) =>
        React.createElement(WorkspaceInvitationsListItem, { key: row.id, invitation: row })
      )
    );
  }
  await view.render(React.createElement(List));
  await settleQuery();
  const select = () => view.environment.container.querySelector("select");
  assert.deepEqual(
    [...select().options].map((option) => option.value),
    ["5", "15", "20"]
  );
  await changeFormControl(select(), "5");
  await settleQuery();
  assert.equal(select().value, "15");
  assert.equal(globalThis.invitationActions.toasts.at(-1).message, "Role change refused");
  roleFailure = false;
  await changeFormControl(select(), "5");
  await settleQuery();
  assert.equal(select().value, "5");
  assert.deepEqual(
    updates,
    Array.from({ length: 2 }, () => ({ workspace: "acme", id: "invite", patch: { role: 5 } }))
  );
  const button = (label) =>
    [...view.environment.container.querySelectorAll("button")].find((item) => item.textContent === label);
  await act(async () => button("common.remove").click());
  assert.match(view.environment.container.querySelector('[role="dialog"]').textContent, /new@example.test/);
  await act(async () => button("cancel").click());
  assert.equal(view.environment.container.querySelector('[role="dialog"]'), null);
  await act(async () => button("common.remove").click());
  await act(async () => button("remove").click());
  await settleQuery();
  assert.match(view.environment.container.querySelector('[role="alert"]').textContent, /Removal offline/);
  deleteFailure = false;
  await act(async () => button("remove").click());
  await settleQuery();
  assert.equal(view.environment.container.querySelector("select"), null);
  assert.equal(view.environment.container.querySelector('[role="dialog"]'), null);
  assert.deepEqual(removals, [
    { workspace: "acme", id: "invite" },
    { workspace: "acme", id: "invite" },
  ]);
  assert.ok(globalThis.invitationActions.toasts.some((toast) => toast.message === "Invitation removed successfully."));
});

await test("workspace members cannot grant an administrator invitation or remove one", async (t) => {
  const view = await createPermissionUiFixture(t, { path: "/acme/settings/members", workspaceRole: 15 });
  const { WorkspaceInvitationsListItem } =
    await import("../../apps/web/core/components/workspace/settings/invitations-list-item.tsx");
  await view.render(React.createElement(WorkspaceInvitationsListItem, { invitation }));
  const options = [...view.environment.container.querySelector("select").options].map((option) => option.value);
  assert.deepEqual(options, ["5", "15"]);
  assert.doesNotMatch(view.environment.container.textContent, /common.remove/);
});
