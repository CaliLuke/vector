import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { settleQuery } from "./helpers/query-router.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

const state = { toasts: [] };
globalThis.subscriptionTestState = state;
registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    return resolvePackageStub(
      specifier,
      {
        "@plane/i18n": `export const useTranslation=()=>({t:key=>({"common.actions.subscribe":"Subscribe","common.actions.unsubscribe":"Unsubscribe","common.loading":"Loading"})[key]??key});`,
        "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:'success',ERROR:'error'};export const setToast=toast=>globalThis.subscriptionTestState.toasts.push(toast);`,
        "@plane/ui": "export const Loader=Object.assign(({children})=>children,{Item:()=>null});",
      },
      context,
      nextResolve
    );
  },
});
const { IssueSubscription } = await import("../../apps/web/core/components/issues/issue-detail/subscription.tsx");
const { IssueService } = await import("../../apps/web/core/services/issue/issue.service.ts");
const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
const { issueSubscriptionOptions } = await import("../../apps/web/core/queries/issue-subscription.ts");
const { EIssueServiceType } = webRequire("@plane/types");
const scope = {
  userId: "alice",
  workspaceSlug: "default",
  projectId: "project-id",
  issueId: "issue-id",
  serviceType: EIssueServiceType.ISSUES,
};

async function fixture(t, projectRole = webRequire("@plane/types").EUserProjectRoles.ADMIN) {
  state.toasts = [];
  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false, gcTime: Infinity } },
  });
  client.setQueryData(currentUserOptions().queryKey, { id: "alice" });
  await seedPermissionQueries(client, {
    userId: "alice",
    workspaceSlug: "default",
    projectRoles: { "project-id": projectRole },
  });
  client.setQueryData(issueSubscriptionOptions(scope).queryKey, false);
  t.after(async () => {
    await environment.render(null);
    await client.cancelQueries();
    client.clear();
    await environment.unmount();
  });
  const render = (issueId = "issue-id") =>
    environment.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(IssueSubscription, { ...scope, issueId })
      )
    );
  await render();
  return { environment, client, render };
}

await test("issue subscription buttons keep accessible names after native Query updates", async (t) => {
  const subscribe = t.mock.method(IssueService.prototype, "subscribeToIssueNotifications", async () => {});
  const unsubscribe = t.mock.method(IssueService.prototype, "unsubscribeFromIssueNotifications", async () => {});
  const { environment, client } = await fixture(t);
  let button = environment.container.querySelector("button");
  assert.equal(button.getAttribute("aria-label"), "Subscribe");
  assert.equal(button.tabIndex, 0);
  await act(async () => button.click());
  await settleQuery();
  button = environment.container.querySelector("button");
  assert.equal(button.getAttribute("aria-label"), "Unsubscribe");
  assert.equal(client.getQueryData(issueSubscriptionOptions(scope).queryKey), true);
  await act(async () => button.click());
  await settleQuery();
  assert.equal(client.getQueryData(issueSubscriptionOptions(scope).queryKey), false);
  assert.deepEqual(subscribe.mock.calls[0].arguments, ["default", "project-id", "issue-id"]);
  assert.deepEqual(unsubscribe.mock.calls[0].arguments, ["default", "project-id", "issue-id"]);
});

await test("subscription action stays disabled during persistence and for a reader", async (t) => {
  const pending = Promise.withResolvers();
  const subscribe = t.mock.method(IssueService.prototype, "subscribeToIssueNotifications", () => pending.promise);
  const { environment, client } = await fixture(t);
  await act(async () => environment.container.querySelector("button").click());
  await settleQuery();
  let button = environment.container.querySelector("button");
  assert.equal(button.disabled, true);
  assert.equal(button.textContent, "Loading");
  await act(async () => pending.resolve());
  await settleQuery();
  await seedPermissionQueries(client, {
    userId: "alice",
    workspaceSlug: "default",
    workspaceRole: webRequire("@plane/types").EUserWorkspaceRoles.MEMBER,
    projectRoles: { "project-id": webRequire("@plane/types").EUserProjectRoles.GUEST },
  });
  await settleQuery();
  button = environment.container.querySelector("button");
  assert.equal(button.disabled, true);
  await act(async () => button.click());
  assert.equal(subscribe.mock.callCount(), 1);
});

await test("failed subscriptions retain confirmed state and allow retry", async (t) => {
  const subscribe = t.mock.method(IssueService.prototype, "subscribeToIssueNotifications", async () => {
    throw new Error("offline");
  });
  const { environment, client } = await fixture(t);
  await act(async () => environment.container.querySelector("button").click());
  await settleQuery();
  assert.equal(client.getQueryData(issueSubscriptionOptions(scope).queryKey), false);
  assert.equal(environment.container.querySelector("button").disabled, false);
  assert.equal(state.toasts.at(-1).type, "error");
  subscribe.mock.mockImplementation(async () => {});
  await act(async () => environment.container.querySelector("button").click());
  await settleQuery();
  assert.equal(client.getQueryData(issueSubscriptionOptions(scope).queryKey), true);
});

await test("a late subscription response cannot toast in a replacement work item", async (t) => {
  const pending = Promise.withResolvers();
  t.mock.method(IssueService.prototype, "subscribeToIssueNotifications", () => pending.promise);
  const { environment, client, render } = await fixture(t);
  await act(async () => environment.container.querySelector("button").click());
  await settleQuery();
  client.setQueryData(issueSubscriptionOptions({ ...scope, issueId: "next" }).queryKey, false);
  await render("next");
  await act(async () => pending.resolve());
  await settleQuery();
  assert.equal(state.toasts.length, 0);
  assert.equal(client.getQueryData(issueSubscriptionOptions({ ...scope, issueId: "next" }).queryKey), false);
  assert.equal(environment.container.querySelector("button").getAttribute("aria-label"), "Subscribe");
});
