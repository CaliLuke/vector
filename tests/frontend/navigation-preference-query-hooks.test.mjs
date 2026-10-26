import { waitForQueryState } from "./helpers/query-state.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { notifyManager } from "@tanstack/react-query";
import { React, act } from "./helpers/react.mjs";
import { createQueryRouterFixture } from "./helpers/query-router.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";
import { changeFormControl, waitForSubmitCompletion } from "./helpers/native-forms.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { waitForMutation } from "./helpers/wait-for-mutation.mjs";
import { EUserWorkspaceRoles } from "../../packages/types/dist/index.mjs";

registerWebModuleHooks();
globalThis.navigationPreferenceReact = React;
globalThis.navigationPreferenceToasts = [];
const stubs = {
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/toast": `export const TOAST_TYPE={ERROR:"error"};export const setToast=value=>globalThis.navigationPreferenceToasts.push(value);`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});

const nativeForms = observeNativeForms(["apps/web/core/components/navigation/customize-navigation-dialog.tsx"]);
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { advanceIdentitySession, identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const preferences = await import("../../apps/web/core/queries/navigation-preferences.ts");
const { usePersonalNavigationPreferences, useProjectNavigationPreferences, useWorkspaceNavigationPreferences } =
  await import("../../apps/web/core/hooks/use-navigation-preferences.ts");

function waitForProjectPreferenceRefresh(client) {
  return waitForQueryState(
    client,
    preferences.navigationPreferenceKeys.workspaceProperties("navigation-user", "acme"),
    (state) => state?.status === "success" && state.data?.navigation_project_limit === 4
  );
}

async function createDelayedProjectPreferences(t) {
  const fixture = await createQueryRouterFixture(t, "/acme/projects/");
  await seedPreferences(fixture.client);
  fixture.client.setQueryData(
    preferences.navigationPreferenceKeys.workspaceProperties("navigation-user", "acme"),
    { navigation_control_preference: "ACCORDION", navigation_project_limit: 10 },
    { updatedAt: 0 }
  );
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceFilters", () => {
    started.resolve();
    return response.promise;
  });
  const { CustomizeNavigationDialog } =
    await import("../../apps/web/core/components/navigation/customize-navigation-dialog.tsx");
  const resolvePreferences = () =>
    response.resolve({
      navigation_control_preference: "ACCORDION",
      navigation_project_limit: 4,
      rich_filters: { and: [] },
      display_filters: {},
      display_properties: {},
    });
  return { fixture, started, resolvePreferences, CustomizeNavigationDialog };
}

async function seedPreferences(client, userId = "navigation-user", slug = "acme") {
  client.setQueryData(identityKeys.currentUser(), { id: userId });
  client.setQueryData(preferences.navigationPreferenceKeys.sidebar(userId, slug), {});
  client.setQueryData(preferences.navigationPreferenceKeys.workspaceProperties(userId, slug), {});
  await seedPermissionQueries(client, {
    userId,
    workspaceSlug: slug,
    workspaceRole: EUserWorkspaceRoles.ADMIN,
  });
}

function useDeterministicQueryNotifications(t) {
  notifyManager.setScheduler(queueMicrotask);
  t.after(() => notifyManager.setScheduler((callback) => setTimeout(callback, 0)));
}

function PreferenceProbe({ id }) {
  const personal = usePersonalNavigationPreferences();
  const workspace = useWorkspaceNavigationPreferences();
  const project = useProjectNavigationPreferences();
  const snapshot = {
    personal: personal.preferences.items,
    workspace: workspace.getWorkspaceItemState("inbox"),
    missingWorkspaceItem: workspace.getWorkspaceItemState("not-present"),
    project: project.preferences,
  };
  return React.createElement(
    "section",
    { "data-preference-probe": id },
    React.createElement("output", null, JSON.stringify(snapshot)),
    React.createElement("button", { onClick: () => workspace.updateWorkspaceItemSortOrder("inbox", 8) }, "Reorder"),
    React.createElement("button", { onClick: () => workspace.toggleWorkspaceItem("inbox", true) }, "Pin")
  );
}

function Consumers() {
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(PreferenceProbe, { id: "first" }),
    React.createElement(PreferenceProbe, { id: "second" })
  );
}

const dialogStubs = {
  ...stubs,
  "lucide-react": `export const GripVertical=()=>null;export const X=()=>null;`,
  "@plane/constants": `export const API_BASE_URL="/api";export const WORKSPACE_SIDEBAR_DYNAMIC_NAVIGATION_ITEMS_LINKS=[];export const EUserPermissionsLevel={WORKSPACE:"workspace"};export const EUserPermissions={ADMIN:20,MEMBER:15,GUEST:5};`,
  "@plane/propel/button": `const React=globalThis.navigationPreferenceReact;export const Button=({children,loading,...props})=>React.createElement("button",props,children);`,
  "@plane/ui": `const React=globalThis.navigationPreferenceReact;export const EModalPosition={CENTER:"center"};export const EModalWidth={XXL:"xxl"};export const ModalCore=({children})=>React.createElement("div",{role:"dialog"},children);export const Checkbox=({checked,onChange})=>React.createElement("input",{type:"checkbox",checked,onChange});export const Sortable=({data,render})=>React.createElement(React.Fragment,null,data.map(item=>React.createElement(React.Fragment,{key:item.key},render(item))));`,
  "@plane/utils/common": 'export const cn=(...classes)=>classes.filter(Boolean).join(" ");',
  "@/components/workspace/sidebar/helper": `export const getSidebarNavigationItemIcon=()=>null;`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, dialogStubs, context, nextResolve),
});

await test("navigation consumers share defaults and ordered pending overlays without persisting failed fields", async (t) => {
  useDeterministicQueryNotifications(t);
  globalThis.navigationPreferenceToasts.length = 0;
  const fixture = await createQueryRouterFixture(t, "/acme/projects/");
  await seedPreferences(fixture.client);
  fixture.client.setQueryData(preferences.navigationPreferenceKeys.sidebar("navigation-user", "acme"), {
    inbox: { is_pinned: false, sort_order: 3 },
  });
  const { client, environment } = fixture;
  await fixture.render(React.createElement(Consumers));

  const output = (id) => environment.container.querySelector(`[data-preference-probe="${id}"] output`).textContent;
  const expectedDefaults = {
    personal: {
      stickies: { enabled: false, sort_order: 0 },
      your_work: { enabled: true, sort_order: 1 },
      drafts: { enabled: true, sort_order: 2 },
    },
    workspace: { is_pinned: false, sort_order: 3 },
    missingWorkspaceItem: { is_pinned: false, sort_order: 0 },
    project: { navigationMode: "ACCORDION", limitedProjectsCount: 10, showLimitedProjects: false },
  };
  assert.deepEqual(JSON.parse(output("first")), expectedDefaults);
  assert.equal(output("first"), output("second"));

  const started = Promise.withResolvers();
  const finishReorder = Promise.withResolvers();
  const pinStarted = Promise.withResolvers();
  const finishPin = Promise.withResolvers();
  const patches = [];
  let call = 0;
  t.mock.method(WorkspaceService.prototype, "updateBulkSidebarPreferences", (_slug, patch) => {
    patches.push(patch);
    call++;
    if (call === 1) {
      started.resolve();
      return finishReorder.promise;
    }
    pinStarted.resolve();
    return finishPin.promise;
  });
  t.mock.method(WorkspaceService.prototype, "fetchSidebarNavigationPreferences", async () => ({
    inbox: { is_pinned: true, sort_order: 3 },
  }));

  const reorder = environment.container.querySelector('[data-preference-probe="first"] button');
  const pin = environment.container.querySelector('[data-preference-probe="second"] button:last-child');
  const reorderPending = waitForMutation(
    client,
    ({ state }) => state.status === "pending" && state.variables?.[0]?.sort_order === 8
  );
  await act(async () => {
    reorder.click();
    await Promise.all([started.promise, reorderPending]);
  });
  const pinPending = waitForMutation(
    client,
    ({ state }) => state.status === "pending" && state.variables?.[0]?.is_pinned === true
  );
  await act(async () => {
    pin.click();
    await pinPending;
  });
  assert.deepEqual(JSON.parse(output("first")).workspace, { is_pinned: true, sort_order: 8 });
  assert.equal(output("first"), output("second"), "both consumers read the same MutationCache overlay");

  const failedReorder = waitForMutation(
    client,
    ({ state }) => state.status === "error" && state.variables?.[0]?.sort_order === 8
  );
  const queuedPin = pinPending;
  await act(async () => {
    finishReorder.reject(new Error("reorder unavailable"));
    await Promise.all([failedReorder, queuedPin, pinStarted.promise]);
  });
  assert.deepEqual(patches, [[{ key: "inbox", sort_order: 8 }], [{ key: "inbox", is_pinned: true }]]);
  assert.deepEqual(JSON.parse(output("first")).workspace, { is_pinned: true, sort_order: 3 });
  assert.equal(globalThis.navigationPreferenceToasts.length, 1, "each failed immediate action reports once");
  const savedPin = waitForMutation(
    client,
    ({ state }) => state.status === "success" && state.variables?.[0]?.is_pinned === true
  );
  await act(async () => {
    finishPin.resolve({ message: "Saved" });
    await savedPin;
  });
  assert.deepEqual(JSON.parse(output("first")).workspace, { is_pinned: true, sort_order: 3 });
  assert.deepEqual(client.getQueryData(preferences.navigationPreferenceKeys.sidebar("navigation-user", "acme")), {
    inbox: { is_pinned: true, sort_order: 3 },
  });
  assert.equal(globalThis.navigationPreferenceToasts.length, 1);
});

await test("a failed immediate preference action does not show stale-session feedback", async (t) => {
  useDeterministicQueryNotifications(t);
  globalThis.navigationPreferenceToasts.length = 0;
  const fixture = await createQueryRouterFixture(t, "/acme/projects/");
  await seedPreferences(fixture.client);
  await fixture.render(React.createElement(PreferenceProbe, { id: "session" }));
  const started = Promise.withResolvers();
  const failWrite = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "updateBulkSidebarPreferences", () => {
    started.resolve();
    return failWrite.promise;
  });

  await act(async () => fixture.environment.container.querySelector("button:last-child").click());
  await started.promise;
  await act(async () => advanceIdentitySession(fixture.client));
  assert.deepEqual(
    JSON.parse(fixture.environment.container.querySelector("output").textContent).workspace,
    { is_pinned: false, sort_order: 0 },
    "a new session generation hides the old pending preference overlay"
  );
  await act(async () => failWrite.reject(new Error("offline")));

  assert.deepEqual(globalThis.navigationPreferenceToasts, []);
});

await test("a failed immediate preference action does not show feedback after navigation", async (t) => {
  useDeterministicQueryNotifications(t);
  globalThis.navigationPreferenceToasts.length = 0;
  const fixture = await createQueryRouterFixture(t, "/acme/projects/");
  await seedPreferences(fixture.client);
  await fixture.render(React.createElement(PreferenceProbe, { id: "route" }));
  const started = Promise.withResolvers();
  const failWrite = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "updateBulkSidebarPreferences", () => {
    started.resolve();
    return failWrite.promise;
  });
  const mutationFailed = waitForMutation(
    fixture.client,
    ({ state }) => state.status === "error" && state.variables?.[0]?.is_pinned === true
  );

  await act(async () => {
    fixture.environment.container.querySelector("button:last-child").click();
    await started.promise;
  });
  await act(async () => fixture.router.navigate({ to: "/acme/projects/p1/issues/" }));
  await act(async () => {
    failWrite.reject(new Error("offline"));
    await mutationFailed;
  });

  assert.deepEqual(globalThis.navigationPreferenceToasts, []);
});

await test("project-count Form reports a native failed outcome and retries the same write without a hook toast", async (t) => {
  globalThis.navigationPreferenceToasts.length = 0;
  const fixture = await createQueryRouterFixture(t, "/acme/projects/");
  await seedPreferences(fixture.client);
  fixture.client.setQueryData(preferences.navigationPreferenceKeys.workspaceProperties("navigation-user", "acme"), {
    navigation_control_preference: "ACCORDION",
    navigation_project_limit: 4,
  });
  const { CustomizeNavigationDialog } =
    await import("../../apps/web/core/components/navigation/customize-navigation-dialog.tsx");
  let attempts = 0;
  t.mock.method(WorkspaceService.prototype, "patchWorkspaceFilters", async (_slug, patch) => {
    attempts++;
    assert.deepEqual(patch, { navigation_project_limit: 8 });
    if (attempts === 1) throw new Error("offline");
    return {
      navigation_control_preference: "ACCORDION",
      navigation_project_limit: 8,
      rich_filters: { and: [] },
      display_filters: {},
      display_properties: {},
    };
  });

  await fixture.render(React.createElement(CustomizeNavigationDialog, { isOpen: true, onClose() {} }));
  const input = fixture.environment.container.querySelector('[name="projectCount"]');
  await changeFormControl(input, "8");
  await act(async () => input.dispatchEvent(new FocusEvent("focusout", { bubbles: true })));
  let formApi = nativeForms.getForms("apps/web/core/components/navigation/customize-navigation-dialog.tsx").at(-1);
  await waitForSubmitCompletion(formApi);
  const retry = fixture.environment.container.querySelector('button[type="submit"]');
  assert.equal(retry.textContent, "Retry");
  assert.equal(input.value, "8");
  assert.match(
    fixture.environment.container.querySelector('[role="alert"]').textContent,
    /Could not save the project limit/
  );
  assert.equal(formApi.state.isSubmitSuccessful, false);
  assert.deepEqual(globalThis.navigationPreferenceToasts, []);

  await act(async () => retry.click());
  await waitForSubmitCompletion(formApi);
  assert.equal(attempts, 2);
  assert.equal(formApi.state.isSubmitSuccessful, true);
  assert.equal(fixture.environment.container.querySelector('[role="alert"]'), null);
  assert.deepEqual(globalThis.navigationPreferenceToasts, []);
});

await test("project-count Form adopts asynchronously loaded native defaults while pristine", async (t) => {
  useDeterministicQueryNotifications(t);
  const { fixture, started, resolvePreferences, CustomizeNavigationDialog } = await createDelayedProjectPreferences(t);

  await fixture.render(
    React.createElement(
      React.Fragment,
      null,
      React.createElement(CustomizeNavigationDialog, { isOpen: true, onClose() {} }),
      React.createElement(PreferenceProbe, { id: "async-default" })
    )
  );
  const input = fixture.environment.container.querySelector('[name="projectCount"]');
  assert.equal(input.value, "10", "the Form initially uses the cached value while a stale query refreshes");
  await started.promise;

  await act(async () => resolvePreferences());
  await act(async () => waitForProjectPreferenceRefresh(fixture.client));
  assert.equal(
    JSON.parse(
      fixture.environment.container.querySelector('[data-preference-probe="async-default"] output').textContent
    ).project.limitedProjectsCount,
    4,
    "the native preference hook observes the refreshed query value"
  );

  assert.equal(
    fixture.environment.container.querySelector('[name="projectCount"]').value,
    "4",
    "the pristine input adopts the refreshed native default"
  );
});

await test("project-count Form preserves an edited value when the workspace default loads", async (t) => {
  useDeterministicQueryNotifications(t);
  const { fixture, started, resolvePreferences, CustomizeNavigationDialog } = await createDelayedProjectPreferences(t);

  await fixture.render(React.createElement(CustomizeNavigationDialog, { isOpen: true, onClose() {} }));
  const input = fixture.environment.container.querySelector('[name="projectCount"]');
  await started.promise;
  await act(async () => input.dispatchEvent(new FocusEvent("focusout", { bubbles: true })));
  await changeFormControl(input, "7");

  await act(async () => resolvePreferences());
  await act(async () => waitForProjectPreferenceRefresh(fixture.client));

  assert.equal(
    fixture.environment.container.querySelector('[name="projectCount"]').value,
    "7",
    "a loaded default does not overwrite the user's touched draft"
  );
});

nativeForms.restore();
