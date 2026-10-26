import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import {
  catalogueSettings,
  catalogueUser,
  fallbackWorkspace,
  renderWorkspaceCatalogueRoute,
} from "./helpers/workspace-catalogue-fixture.mjs";

const modalCoreUrl = new URL("../../packages/ui/src/modals/modal-core.tsx", import.meta.url).href;
const modalConstantsUrl = new URL("../../packages/ui/src/modals/constants.ts", import.meta.url).href;
const state = { membershipTransport: {}, columns: {}, toasts: [] };
globalThis.workspaceMemberOutcomes = state;
globalThis.workspaceMemberOutcomesReact = React;
const stubs = {
  "@/components/ui/loader/layouts/members-layout-loader": `export const MembersLayoutLoader=()=>null;`,
  "@/components/workspace/settings/useMemberColumns": `const React=globalThis.workspaceMemberOutcomesReact;export const useMemberColumns=()=>{const [removeMemberModal,setRemoveMemberModal]=React.useState(globalThis.workspaceMemberOutcomes.columns.removeMemberModal);return {...globalThis.workspaceMemberOutcomes.columns,removeMemberModal,setRemoveMemberModal};};`,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/toast": `export const TOAST_TYPE={ERROR:"error",SUCCESS:"success"};export const setToast=value=>globalThis.workspaceMemberOutcomes.toasts.push(value);`,
  "@plane/ui": `export { ModalCore } from ${JSON.stringify(modalCoreUrl)};export { EModalPosition, EModalWidth } from ${JSON.stringify(modalConstantsUrl)};const React=globalThis.workspaceMemberOutcomesReact;export const Table=()=>null;`,
  "@plane/propel/button": `const React=globalThis.workspaceMemberOutcomesReact;export const Button=({variant,size,loading,...props})=>React.createElement("button",{...props,"aria-busy":loading||undefined},props.children);`,
};
registerWebModuleHooks();
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});

const user = catalogueUser;
const settings = catalogueSettings;
const fallback = fallbackWorkspace;

async function renderMembers(t, memberId = user.id) {
  const userService = (await import("../../apps/web/core/services/user.service.ts")).default;
  t.mock.method(userService, "leaveWorkspace", state.membershipTransport.leaveWorkspace);
  state.columns = {
    columns: [{ key: "member" }],
    workspaceSlug: "removed",
    removeMemberModal: { id: `membership-${memberId}`, member: { id: memberId, display_name: "Member" } },
    setRemoveMemberModal: () => {},
  };
  return renderWorkspaceCatalogueRoute(t, {
    path: "/removed/settings/members",
    loadComponent: async () => {
      const { WorkspaceMembersListItem } =
        await import("../../apps/web/core/components/workspace/settings/members-list-item.tsx");
      const { useRouterState } = await import("@tanstack/react-router");
      return function WorkspaceMembersRoute() {
        const pathname = useRouterState({ select: (routerState) => routerState.location.pathname });
        return pathname.includes("/settings/members")
          ? React.createElement(WorkspaceMembersListItem, { memberDetails: [] })
          : null;
      };
    },
  });
}

const findButton = (text) =>
  [...document.body.querySelectorAll("button")].find((button) => button.textContent === text);
const click = (button) => act(async () => button.click());
const settleReact = () => act(async () => new Promise((resolve) => setTimeout(resolve, 0)));

function createDeferredLeave() {
  let resolveLeave;
  const calls = [];
  return {
    calls,
    leave: async (slug) => {
      calls.push(slug);
      await new Promise((resolve) => (resolveLeave = resolve));
    },
    resolve: () => resolveLeave?.(),
  };
}

function waitForBody(predicate, description) {
  if (predicate()) return Promise.resolve();
  return new Promise((resolve, reject) => {
    const observer = new MutationObserver(() => {
      if (!predicate()) return;
      clearTimeout(timeout);
      observer.disconnect();
      resolve();
    });
    observer.observe(document.body, { childList: true, subtree: true });
    const timeout = setTimeout(() => {
      observer.disconnect();
      reject(new Error(`${description}: ${document.body.textContent}`));
    }, 2000);
  });
}

await test("failed leave keeps confirmation open, permits dismissal only after settlement, and retries primary action", async (t) => {
  let attempt = 0;
  let rejectFirst;
  let resolveSecond;
  t.after(() => resolveSecond?.());
  const leaveCalls = [];
  state.membershipTransport = {
    leaveWorkspace: async (slug) => {
      leaveCalls.push(slug);
      attempt += 1;
      if (attempt === 1) await new Promise((_, reject) => (rejectFirst = reject));
      else await new Promise((resolve) => (resolveSecond = resolve));
    },
  };
  state.toasts = [];
  const { environment, client, router } = await renderMembers(t);
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const userService = (await import("../../apps/web/core/services/user.service.ts")).default;
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async () => {
    throw new Error("Workspace refresh failed");
  });
  t.mock.method(userService, "currentUserSettings", async () => {
    throw new Error("Settings refresh failed");
  });

  await click(findButton("leave"));
  await settleReact();
  assert.equal(attempt, 1);
  assert.equal(findButton("leaving")?.disabled, true, document.body.innerHTML);
  const cancel = findButton("cancel");
  assert.equal(cancel.disabled, true);
  await click(cancel);
  await act(async () => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
  assert.ok(document.body.querySelector('[role="dialog"]'));
  assert.equal(attempt, 1);
  await act(async () => {
    rejectFirst(new Error("Leave failed"));
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  await settleReact();
  await waitForBody(() => Boolean(document.body.querySelector('[role="alert"]')), "leave failure did not render");
  assert.match(document.body.querySelector('[role="alert"]')?.textContent ?? document.body.innerHTML, /Leave failed/);
  assert.equal(findButton("leave").disabled, false);
  assert.ok(document.body.querySelector('[role="dialog"]'));

  await click(findButton("leave"));
  await settleReact();
  assert.equal(findButton("leaving").disabled, true);
  await act(async () => {
    resolveSecond();
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  await waitForBody(
    () => !document.body.querySelector('[role="dialog"]'),
    "successful leave did not close confirmation"
  );
  assert.deepEqual(leaveCalls, ["removed", "removed"]);
  assert.equal(document.body.querySelector('[role="dialog"]'), null);
  assert.deepEqual(client.getQueryData(["workspaces", user.id]), [fallback]);
  assert.equal(router.state.location.pathname, "/fallback");
  assert.equal(state.toasts.at(-1).title, "Workspace left");
  assert.equal(environment.container.querySelector('[role="alert"]'), null);
});

await test("failed member removal stays open and retries without reporting success", async (t) => {
  let attempt = 0;
  state.membershipTransport = { leaveWorkspace: async () => {} };
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const removals = [];
  t.mock.method(WorkspaceService.prototype, "deleteWorkspaceMember", async (slug, membershipId) => {
    removals.push({ slug, membershipId });
    attempt += 1;
    if (attempt === 1) throw new Error("Remove failed");
  });
  await renderMembers(t, "another-user");

  await click(findButton("remove"));
  await settleReact();
  await waitForBody(
    () => Boolean(document.body.querySelector('[role="alert"]')),
    "member removal failure did not render"
  );
  assert.match(document.body.querySelector('[role="alert"]')?.textContent ?? document.body.innerHTML, /Remove failed/);
  assert.ok(document.body.querySelector('[role="dialog"]'));
  assert.equal(findButton("remove").disabled, false);
  await click(findButton("remove"));
  await settleReact();
  assert.equal(attempt, 2);
  assert.deepEqual(removals, [
    { slug: "removed", membershipId: "membership-another-user" },
    { slug: "removed", membershipId: "membership-another-user" },
  ]);
  await waitForBody(
    () => !document.body.querySelector('[role="dialog"]'),
    "successful removal did not close confirmation"
  );
});

await test("accepted leave does not refresh, toast, or navigate after the identity session changes", async (t) => {
  const leave = createDeferredLeave();
  state.toasts = [];
  state.membershipTransport = { leaveWorkspace: leave.leave };
  const { client, router } = await renderMembers(t);
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const userService = (await import("../../apps/web/core/services/user.service.ts")).default;
  let listRefreshes = 0;
  let settingsRefreshes = 0;
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async () => {
    listRefreshes += 1;
    return [fallback];
  });
  t.mock.method(userService, "currentUserSettings", async () => {
    settingsRefreshes += 1;
    return settings;
  });

  await click(findButton("leave"));
  await settleReact();
  const { advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
  await act(async () => {
    advanceIdentitySession(client);
    leave.resolve();
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  await settleReact();

  assert.deepEqual(leave.calls, ["removed"]);
  assert.equal(listRefreshes, 0);
  assert.equal(settingsRefreshes, 0);
  assert.deepEqual(state.toasts, []);
  assert.equal(router.state.location.pathname, "/removed/settings/members");
});

await test("accepted leave does not redirect after navigation changes while it is pending", async (t) => {
  const leave = createDeferredLeave();
  state.toasts = [];
  state.membershipTransport = { leaveWorkspace: leave.leave };
  const { client, router } = await renderMembers(t);
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const userService = (await import("../../apps/web/core/services/user.service.ts")).default;
  let refreshes = 0;
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async () => {
    refreshes += 1;
    return [fallback];
  });
  t.mock.method(userService, "currentUserSettings", async () => {
    refreshes += 1;
    return settings;
  });

  await click(findButton("leave"));
  await settleReact();
  await act(async () => router.navigate({ href: "/fallback" }));
  await act(async () => {
    leave.resolve();
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  await settleReact();

  assert.deepEqual(leave.calls, ["removed"]);
  assert.equal(refreshes, 0);
  assert.deepEqual(state.toasts, []);
  assert.deepEqual(client.getQueryData(["workspaces", user.id]), [fallback]);
  assert.equal(router.state.location.pathname, "/fallback");
});
