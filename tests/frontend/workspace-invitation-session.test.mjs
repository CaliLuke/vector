import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { useQuery } from "@tanstack/react-query";
import { React, act } from "./helpers/react.mjs";
import { createQueryRouterFixture } from "./helpers/query-router.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.workspaceInvitationSessionReact = React;
const stubs = {
  "@plane/propel/icons": `export const CheckIcon = () => null; export const CloseIcon = () => null;`,
  "@/helpers/authentication.helper": `export const EPageTypes = { PUBLIC: "PUBLIC" };`,
  "@/components/common/logo-spinner": `const React = globalThis.workspaceInvitationSessionReact; export const LogoSpinner = () => React.createElement("div", null, "Loading");`,
  "@/components/ui/empty-space": `const React = globalThis.workspaceInvitationSessionReact;
    export const EmptySpace = ({children, title}) => React.createElement("section", null, React.createElement("h1", null, title), children);
    export const EmptySpaceItem = ({title, action}) => React.createElement("button", {type:"button", onClick:action}, title);`,
  "@/lib/wrappers/authentication-wrapper": `const React = globalThis.workspaceInvitationSessionReact; export const AuthenticationWrapper = ({children}) => React.createElement(React.Fragment, null, children);`,
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (context.parentURL?.endsWith("/workspace-invitations/page.tsx") && specifier === "@plane/constants")
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent(`export const WORKSPACE_INVITATION = (slug, id) => ["workspace-invitation", slug, id];`)}`,
      };
    if (context.parentURL?.endsWith("/workspace-invitations/page.tsx") && specifier === "lucide-react")
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent(`export const Boxes = () => null; export const Share2 = () => null; export const User2 = () => null;`)}`,
      };
    return resolvePackageStub(specifier, stubs, context, nextResolve);
  },
});

function waitForButton(container, label) {
  const findButton = () => [...container.querySelectorAll("button")].find((button) => button.textContent === label);
  const found = findButton();
  if (found) return Promise.resolve(found);
  return new Promise((resolve, reject) => {
    const observer = new MutationObserver(() => {
      const button = findButton();
      if (!button) return;
      observer.disconnect();
      clearTimeout(timeout);
      resolve(button);
    });
    observer.observe(container, { childList: true, subtree: true });
    const timeout = setTimeout(() => {
      observer.disconnect();
      reject(new Error(`${label} invitation action did not render`));
    }, 3000);
  });
}

function waitForElementText(container, selector, expected) {
  if (container.querySelector(selector)?.textContent === expected) return Promise.resolve();
  return new Promise((resolve, reject) => {
    const observer = new MutationObserver(() => {
      if (container.querySelector(selector)?.textContent !== expected) return;
      observer.disconnect();
      clearTimeout(timeout);
      resolve();
    });
    observer.observe(container, { childList: true, characterData: true, subtree: true });
    const timeout = setTimeout(() => {
      observer.disconnect();
      reject(new Error(`${selector} did not show the expected value`));
    }, 3000);
  });
}

function WorkspaceCatalogueProbe({ userId, options }) {
  const { data = [] } = useQuery(options(userId));
  return React.createElement("output", { id: "workspace-catalogue" }, JSON.stringify(data));
}

async function createInvitationSession(t) {
  const invitationRoute = await import("../../apps/web/app/(all)/workspace-invitations/page.tsx");
  const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
  const identity = await import("../../apps/web/core/queries/identity.ts");
  const catalogue = await import("../../apps/web/core/queries/workspaces.ts");
  const page = invitationRoute.Route.options.component;
  const fixture = await createQueryRouterFixture(
    t,
    "/workspace-invitations/?invitation_id=invite&slug=team&token=invite-token"
  );
  const user = { id: "user", email: "person@example.test" };
  const workspace = { id: "workspace", slug: "team", name: "Team" };
  fixture.client.setQueryData(identity.identityKeys.currentUser(), user);
  t.mock.method(WorkspaceService.prototype, "getWorkspaceInvitation", async () => ({
    id: "invite",
    email: user.email,
    accepted: false,
    responded_at: null,
    workspace,
  }));
  return {
    ...fixture,
    user,
    workspace,
    page,
    WorkspaceService,
    identity,
    catalogue,
    render: async ({ observeCatalogue = false } = {}) => {
      await fixture.render(
        React.createElement(
          React.Fragment,
          null,
          React.createElement(page),
          observeCatalogue
            ? React.createElement(WorkspaceCatalogueProbe, {
                userId: user.id,
                options: catalogue.workspacesOptions,
              })
            : null
        )
      );
    },
  };
}

await test("late invitation acceptance does not navigate after the identity generation changes", async (t) => {
  const { environment, client, router, render, WorkspaceService, identity } = await createInvitationSession(t);
  let releaseResponse;
  const response = new Promise((resolve) => {
    releaseResponse = resolve;
  });
  let markInvitationResponse;
  const responseStarted = new Promise((resolve) => {
    markInvitationResponse = resolve;
  });
  let markResponseHandled;
  const responseHandled = new Promise((resolve) => {
    markResponseHandled = resolve;
  });
  let joins = 0;
  const initialHref = router.state.location.href;
  t.mock.method(WorkspaceService.prototype, "joinWorkspace", async () => {
    joins++;
    markInvitationResponse();
    await response;
  });
  t.mock.method(console, "error", () => markResponseHandled());
  await render();
  // The page runs inside the real invitation route context while the service response is controlled.
  const acceptButton = await waitForButton(environment.container, "Accept");
  await act(async () => acceptButton.click());
  await responseStarted;
  assert.equal(joins, 1);
  identity.advanceIdentitySession(client);
  releaseResponse();
  await act(async () => responseHandled);

  assert.equal(router.state.location.href, initialHref);
  assert.equal(joins, 1);
});

await test("accepted invitation refreshes the active workspace catalogue before navigation", async (t) => {
  const { environment, client, router, render, user, workspace, WorkspaceService, catalogue } =
    await createInvitationSession(t);
  let joined = false;
  let catalogueReads = 0;
  t.mock.method(WorkspaceService.prototype, "joinWorkspace", async () => {
    joined = true;
  });
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async () => {
    catalogueReads++;
    return joined ? [workspace] : [];
  });
  const routeResolved = Promise.withResolvers();
  const unsubscribe = router.subscribe("onResolved", () => {
    if (router.state.location.pathname !== "/team") return;
    routeResolved.resolve(client.getQueryData(catalogue.workspaceKeys.list(user.id)));
    unsubscribe();
  });
  t.after(unsubscribe);

  await render({ observeCatalogue: true });
  assert.deepEqual(client.getQueryData(catalogue.workspaceKeys.list(user.id)), []);
  const acceptButton = await waitForButton(environment.container, "Accept");
  await act(async () => acceptButton.click());

  assert.deepEqual(await act(async () => routeResolved.promise), [workspace]);
  await act(async () => waitForElementText(environment.container, "#workspace-catalogue", JSON.stringify([workspace])));
  assert.deepEqual(client.getQueryData(catalogue.workspaceKeys.list(user.id)), [workspace]);
  assert.equal(catalogueReads, 2);
});

await test("accepted membership refreshes its catalogue even after the invitation route changes", async (t) => {
  const { environment, client, router, render, user, workspace, WorkspaceService, catalogue } =
    await createInvitationSession(t);
  let resolveJoin;
  let markJoinStarted;
  const joinStarted = new Promise((resolve) => {
    markJoinStarted = resolve;
  });
  const joinResponse = new Promise((resolve) => {
    resolveJoin = resolve;
  });
  let joined = false;
  let catalogueReads = 0;
  let markStaleFlowHandled;
  const staleFlowHandled = new Promise((resolve) => {
    markStaleFlowHandled = resolve;
  });
  t.mock.method(WorkspaceService.prototype, "joinWorkspace", async () => {
    markJoinStarted();
    await joinResponse;
    joined = true;
  });
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async () => {
    catalogueReads++;
    return joined ? [workspace] : [];
  });
  t.mock.method(console, "error", () => markStaleFlowHandled());
  await render({ observeCatalogue: true });
  assert.deepEqual(client.getQueryData(catalogue.workspaceKeys.list(user.id)), []);
  const acceptButton = await waitForButton(environment.container, "Accept");
  await act(async () => acceptButton.click());
  await joinStarted;
  await act(async () => router.navigate({ href: "/elsewhere" }));
  resolveJoin();
  await act(async () => staleFlowHandled);

  assert.equal(router.state.location.pathname, "/elsewhere");
  await act(async () => waitForElementText(environment.container, "#workspace-catalogue", JSON.stringify([workspace])));
  assert.deepEqual(client.getQueryData(catalogue.workspaceKeys.list(user.id)), [workspace]);
  assert.equal(catalogueReads, 2);
});
