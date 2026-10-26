import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
const componentUrl = new URL("../../apps/web/core/components/workspace/settings/members-list.tsx", import.meta.url)
  .href;
const state = { roster: {}, invitations: {}, retries: { roster: 0, membership: 0, invitations: 0 } };
globalThis.workspaceRosterConsumerState = state;
globalThis.workspaceRosterConsumerReact = React;
const stubs = {
  "@tanstack/react-router": `export const useParams=()=>({workspaceSlug:"acme"});`,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/button": `const React=globalThis.workspaceRosterConsumerReact;export const Button=({children,variant,size,...props})=>React.createElement("button",props,children);`,
  "@plane/propel/icons": `const React=globalThis.workspaceRosterConsumerReact;export const ChevronDownIcon=props=>React.createElement("span",props);`,
  "@plane/ui": `const React=globalThis.workspaceRosterConsumerReact;export const Collapsible=({title,children})=>React.createElement(React.Fragment,null,title,children);`,
  "@headlessui/react": `const React=globalThis.workspaceRosterConsumerReact;const Disclosure=({children})=>React.createElement(React.Fragment,null,children);Disclosure.Panel=({children})=>React.createElement(React.Fragment,null,children);export {Disclosure};`,
  "@/components/common/count-chip": `const React=globalThis.workspaceRosterConsumerReact;export const CountChip=({count})=>React.createElement("span",{"data-count":count},String(count));`,
  "@/components/ui/loader/settings/members": `const React=globalThis.workspaceRosterConsumerReact;export const MembersSettingsLoader=()=>React.createElement("div",{"data-testid":"members-loader"});`,
  "@/hooks/use-member-rosters": `export const useWorkspaceMemberRoster=()=>globalThis.workspaceRosterConsumerState.roster;`,
  "@/hooks/use-member-roster-mutations": `export const useWorkspaceInvitations=()=>globalThis.workspaceRosterConsumerState.invitations;`,
  "./invitations-list-item": `const React=globalThis.workspaceRosterConsumerReact;export const WorkspaceInvitationsListItem=({invitation})=>React.createElement("div",{"data-invitation":invitation.email},invitation.email);`,
  "./members-list-item": `const React=globalThis.workspaceRosterConsumerReact;export const WorkspaceMembersListItem=({memberDetails})=>React.createElement("div",null,...memberDetails.map(row=>React.createElement("span",{"data-member":row.member.id,key:row.member.id},row.member.id)));`,
};

registerWebModuleHooks();
registerHooks({
  resolve: (specifier, context, nextResolve) =>
    context.parentURL === componentUrl
      ? resolvePackageStub(specifier, stubs, context, nextResolve)
      : nextResolve(specifier, context),
});
const { WorkspaceMembersList } = await import(componentUrl);

const member = (id, role, isActive = true, isBot = false) => ({
  id: `membership-${id}`,
  role,
  is_active: isActive,
  member: {
    id,
    first_name: id,
    last_name: "Member",
    display_name: id,
    email: `${id}@example.test`,
    is_bot: isBot,
  },
});
const successQuery = { isPending: false, isError: false, isSuccess: true, refetch: async () => {} };
const defaultInvitations = () => ({
  ...successQuery,
  data: [{ id: "invite-1", email: "pending@example.test", role: 15, invite_link: "/invite" }],
  membershipQuery: { ...successQuery },
});
function render(element, environment) {
  return environment.render(element);
}
function list(props = {}) {
  return React.createElement(WorkspaceMembersList, {
    searchQuery: "",
    isAdmin: true,
    filters: {},
    onFilterUpdate: () => {},
    ...props,
  });
}

await test("settings list filters native roster rows, hides bots, and keeps pending invitations searchable", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  state.roster = {
    query: successQuery,
    members: [member("alice", 20), member("bob", 5, false), member("service", 15, true, true)],
  };
  state.invitations = defaultInvitations();

  await render(list({ filters: { roles: ["suspended"] } }), environment);

  assert.deepEqual(
    [...environment.container.querySelectorAll("[data-member]")].map((element) => element.dataset.member),
    ["bob"]
  );
  assert.equal(environment.container.querySelector('[data-member="service"]'), null);
  assert.equal(
    environment.container.querySelector('[data-invitation="pending@example.test"]')?.textContent,
    "pending@example.test"
  );

  await render(list({ searchQuery: "pending@" }), environment);
  assert.equal(environment.container.querySelectorAll("[data-member]").length, 0);
  assert.equal(environment.container.querySelectorAll("[data-invitation]").length, 1);
  assert.equal(environment.container.querySelector('[data-count="1"]')?.textContent, "1");
});

await test("settings roster preserves workspace joining-date sorting from membership creation time", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  state.roster = {
    query: successQuery,
    members: [
      { ...member("later", 15), created_at: "2025-05-01T00:00:00Z" },
      { ...member("earlier", 15), created_at: "2024-05-01T00:00:00Z" },
    ],
  };
  state.invitations = { ...defaultInvitations(), data: [] };

  await render(list({ filters: { order_by: "joining_date" } }), environment);

  assert.deepEqual(
    [...environment.container.querySelectorAll("[data-member]")].map((element) => element.dataset.member),
    ["earlier", "later"]
  );
});

await test("settings list exposes membership and invitation read failures with distinct retry actions", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  let membershipRetries = 0;
  state.roster = {
    query: {
      ...successQuery,
      isError: true,
      isSuccess: false,
      refetch: async () => {
        state.retries.roster += 1;
      },
    },
  };
  state.invitations = {
    isPending: true,
    isError: false,
    isSuccess: false,
    data: undefined,
    refetch: async () => {
      state.retries.invitations += 1;
    },
    membershipQuery: {
      ...successQuery,
      isError: true,
      isSuccess: false,
      refetch: async () => {
        membershipRetries += 1;
      },
    },
  };

  await render(list(), environment);
  assert.equal(environment.container.querySelector('[data-testid="members-loader"]'), null);
  assert.equal(environment.container.querySelectorAll('[role="alert"]').length, 2);
  const retryButtons = [...environment.container.querySelectorAll("button")].filter(
    (button) => button.textContent === "retry"
  );
  assert.equal(retryButtons.length, 2);
  await act(async () => retryButtons[0].click());
  await act(async () => retryButtons[1].click());
  assert.equal(membershipRetries, 1);
  assert.equal(state.retries.roster, 1);

  state.roster = { query: successQuery, members: [] };
  state.invitations = {
    ...defaultInvitations(),
    isError: true,
    isSuccess: false,
    data: undefined,
    refetch: async () => {
      state.retries.invitations += 1;
    },
    membershipQuery: { ...successQuery },
  };
  await render(list(), environment);
  const invitationRetry = [...environment.container.querySelectorAll("button")].find(
    (button) => button.textContent === "retry"
  );
  await act(async () => invitationRetry.click());
  assert.equal(state.retries.invitations, 1);
  assert.equal(environment.container.querySelector('[data-testid="members-loader"]'), null);
});
