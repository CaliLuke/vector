import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const packageStubs = {
  "mobx-react": "export const observer = component => component;",
  "lodash-es": "export const isNil = value => value == null;",
  "lucide-react": "export const Bell = () => null; export const BellOff = () => null;",
  "@plane/constants":
    "export const EUserPermissions = { ADMIN: 'admin', MEMBER: 'member' }; export const EUserPermissionsLevel = { PROJECT: 'project' };",
  "@plane/i18n": `export const useTranslation = () => ({ t: key => ({ "common.actions.subscribe": "Subscribe", "common.actions.unsubscribe": "Unsubscribe", "common.loading": "Loading", "toast.success": "Success", "toast.error": "Error", "issue.subscription.actions.subscribed": "Subscribed", "issue.subscription.actions.unsubscribed": "Unsubscribed", "common.error.message": "Error" })[key] ?? key });`,
  "@plane/propel/toast":
    "export const TOAST_TYPE = { SUCCESS: 'success', ERROR: 'error' }; export const setToast = () => {};",
  "@plane/types": "export const EIssueServiceType = { ISSUES: 'issues' };",
  "@plane/ui": "export const Loader = Object.assign(({ children }) => children, { Item: () => null });",
  "@/hooks/store/use-issue-detail": "export const useIssueDetail = () => globalThis.__subscriptionIssueDetail;",
  "@/hooks/store/user":
    "export const useUserPermissions = () => ({ allowPermissions: () => globalThis.__subscriptionEditable });",
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/propel/button") {
      const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
      return nextResolve(pathToFileURL(path.join(repositoryRoot, "packages/propel/src/button/index.ts")).href, context);
    }
    if (Object.hasOwn(packageStubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(packageStubs[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

const subscriptions = [];
let subscribed = false;
let pendingAction;
globalThis.__subscriptionEditable = true;
globalThis.__subscriptionIssueDetail = {
  subscription: { getSubscriptionByIssueId: () => subscribed },
  createSubscription: (...args) => {
    subscriptions.push(["subscribe", ...args]);
    return pendingAction ?? Promise.resolve();
  },
  removeSubscription: (...args) => {
    subscriptions.push(["unsubscribe", ...args]);
    return pendingAction ?? Promise.resolve();
  },
};

const { IssueSubscription } = await import("../../apps/web/core/components/issues/issue-detail/subscription.tsx");

function renderSubscription(environment) {
  return environment.render(
    React.createElement(IssueSubscription, {
      workspaceSlug: "default",
      projectId: "project-id",
      issueId: "issue-id",
    })
  );
}

test("issue subscription buttons keep translated accessible names in both states", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  subscriptions.length = 0;
  subscribed = false;
  await renderSubscription(environment);

  let button = environment.container.querySelector("button");
  assert.ok(button);
  assert.equal(button.getAttribute("aria-label"), "Subscribe");
  assert.equal(button.tabIndex, 0);
  await React.act(async () => button.click());
  assert.deepEqual(subscriptions, [["subscribe", "default", "project-id", "issue-id"]]);

  subscribed = true;
  await renderSubscription(environment);
  button = environment.container.querySelector("button");
  assert.equal(button.getAttribute("aria-label"), "Unsubscribe");
  await React.act(async () => button.click());
  assert.deepEqual(subscriptions[1], ["unsubscribe", "default", "project-id", "issue-id"]);
});

test("subscription action becomes disabled while saving and when the user cannot edit", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  subscriptions.length = 0;
  subscribed = false;
  let resolveAction;
  pendingAction = new Promise((resolve) => {
    resolveAction = resolve;
  });
  await renderSubscription(environment);

  let button = environment.container.querySelector("button");
  await React.act(async () => {
    button.focus();
    button.click();
    await Promise.resolve();
  });
  button = environment.container.querySelector("button");
  assert.equal(button.disabled, true);
  assert.equal(button.getAttribute("aria-label"), "Subscribe");
  assert.equal(button.textContent, "Loading");

  await React.act(async () => resolveAction());
  pendingAction = undefined;
  globalThis.__subscriptionEditable = false;
  await renderSubscription(environment);
  button = environment.container.querySelector("button");
  assert.equal(button.disabled, true);
  await React.act(async () => button.click());
  assert.equal(subscriptions.length, 1, "disabled controls do not trigger another subscription request");
});
