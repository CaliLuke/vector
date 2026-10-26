import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
const pageUrl = new URL(
  "../../apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx",
  import.meta.url
).href;
globalThis.workspaceMemberRouteReact = React;
globalThis.workspaceMemberRouteState = { workspaceSlug: "acme", invitedWorkspace: undefined };
const stubs = {
  "@tanstack/react-router": `export const createFileRoute=()=>options=>({...options,useParams:()=>({workspaceSlug:globalThis.workspaceMemberRouteState.workspaceSlug})});`,
  "@plane/constants": `export const EUserPermissions={MEMBER:15,ADMIN:20};`,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/button": `const React=globalThis.workspaceMemberRouteReact;export const Button=({children,...props})=>React.createElement("button",props,children);`,
  "@plane/propel/icons": `const React=globalThis.workspaceMemberRouteReact;export const SearchIcon=props=>React.createElement("span",props);`,
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:"success",ERROR:"error"};export const setToast=()=>{};`,
  "@plane/utils/common": 'export const cn=(...values)=>values.filter(Boolean).join(" ");',
  "@/components/auth-screens/not-authorized-view": `const React=globalThis.workspaceMemberRouteReact;export const NotAuthorizedView=()=>React.createElement("div",{"data-testid":"denied"});`,
  "@/components/common/count-chip": `const React=globalThis.workspaceMemberRouteReact;export const CountChip=({count})=>React.createElement("span",null,String(count));`,
  "@/components/core/page-title": `const React=globalThis.workspaceMemberRouteReact;export const PageHead=()=>React.createElement(React.Fragment);`,
  "@/components/project/dropdowns/filters/member-list": `const React=globalThis.workspaceMemberRouteReact;export const MemberListFiltersDropdown=()=>React.createElement(React.Fragment);`,
  "@/components/workspace/settings/members-list": `const React=globalThis.workspaceMemberRouteReact;export const WorkspaceMembersList=()=>React.createElement("div",{"data-testid":"member-list"});`,
  "@/components/settings/content-wrapper": `const React=globalThis.workspaceMemberRouteReact;export const SettingsContentWrapper=({children})=>React.createElement("main",null,children);`,
  "@/components/workspace/members": `const React=globalThis.workspaceMemberRouteReact;export const SendWorkspaceInvitationModal=({isOpen})=>isOpen?React.createElement("div",{"data-testid":"invitation-modal"},React.createElement("input",{"data-testid":"invitation-draft",defaultValue:"old-workspace@example.test"})):null;`,
  "@/hooks/use-workspaces": `export const useCurrentWorkspace=()=>({data:{name:globalThis.workspaceMemberRouteState.workspaceSlug}});`,
  "@/hooks/use-permissions": `export const useWorkspacePermissions=()=>({can:()=>true,isPending:false,isError:false,membershipQuery:{refetch:async()=>{}}});`,
  "@/hooks/use-member-rosters": `export const useWorkspaceMemberRoster=()=>({members:[],query:{isError:false}});`,
  "@/hooks/use-member-roster-mutations": `export const useInviteWorkspaceMembers=(workspaceSlug)=>({mutateAsync:async()=>{globalThis.workspaceMemberRouteState.invitedWorkspace=workspaceSlug;}});`,
  "./header": `const React=globalThis.workspaceMemberRouteReact;export const MembersWorkspaceSettingsHeader=()=>React.createElement("div");`,
};

registerWebModuleHooks();
registerHooks({
  resolve: (specifier, context, nextResolve) =>
    context.parentURL && fileURLToPath(context.parentURL) === fileURLToPath(pageUrl)
      ? resolvePackageStub(specifier, stubs, context, nextResolve)
      : nextResolve(specifier, context),
});
const { Route } = await import(pageUrl);
const { component: WorkspaceMembersSettingsPage } = Route;

await test("workspace switch closes invitation draft and resets its search scope", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  globalThis.workspaceMemberRouteState.workspaceSlug = "acme";
  globalThis.workspaceMemberRouteState.invitedWorkspace = undefined;

  await environment.render(React.createElement(WorkspaceMembersSettingsPage));
  const search = environment.container.querySelector('input[placeholder="search..."]');
  assert.ok(search);
  await act(async () => {
    const invite = [...environment.container.querySelectorAll("button")].find((button) =>
      button.textContent.includes("add_member")
    );
    invite.click();
  });
  assert.equal(
    environment.container.querySelector('[data-testid="invitation-draft"]')?.value,
    "old-workspace@example.test"
  );
  await act(async () => {
    search.value = "old workspace search";
    search.dispatchEvent(new Event("input", { bubbles: true }));
  });
  assert.equal(environment.container.querySelector('input[placeholder="search..."]')?.value, "old workspace search");

  globalThis.workspaceMemberRouteState.workspaceSlug = "beta";
  await environment.render(React.createElement(WorkspaceMembersSettingsPage));

  assert.equal(environment.container.querySelector('[data-testid="invitation-modal"]'), null);
  assert.equal(environment.container.querySelector('input[placeholder="search..."]')?.value, "");
  assert.equal(globalThis.workspaceMemberRouteState.invitedWorkspace, undefined);
});
