import { projectWrapperMetadataStubs } from "./helpers/project-wrapper-metadata-stubs.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
import assert from "node:assert/strict";

import test from "node:test";
import { EUserProjectRoles, EUserWorkspaceRoles } from "../../packages/types/dist/index.mjs";
import { React, act } from "./helpers/react.mjs";
import { createQueryRouterFixture, settleQuery } from "./helpers/query-router.mjs";

import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";

globalThis.projectPreferencesConsumersReact = React;
globalThis.projectPreferencesConsumersHarness = { fetchProjectDetails: async () => ({ id: "project-1" }) };
globalThis.projectPreferencesConsumersToasts = [];
const stubs = {
  "@plane/propel/button": `const h=globalThis.projectPreferencesConsumersReact.createElement;export const Button=({children,...props})=>h("button",props,children);`,
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:"success",ERROR:"error"};export const setToast=value=>globalThis.projectPreferencesConsumersToasts.push(value);`,
  "@/components/auth-screens/project/project-access-restriction": `const h=globalThis.projectPreferencesConsumersReact.createElement;export const ProjectAccessRestriction=()=>h("p",null,"restricted");`,
  ...projectWrapperMetadataStubs,
};
registerWebTestModules(stubs);

const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { projectUserPropertiesKeys } = await import("../../apps/web/core/queries/project-user-properties.ts");
const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");
const { ProjectAuthWrapper } = await import("../../apps/web/core/layouts/auth-layout/project-wrapper.tsx");
const { useProjectIssuePreferencesContext } =
  await import("../../apps/web/core/hooks/use-project-issue-preferences-context.tsx");
const { useTabPreferences } = await import("../../apps/web/core/components/navigation/use-tab-preferences.ts");

const scope = { accountId: "alice", workspaceSlug: "acme", projectId: "project-1" };
const initialProperties = {
  preferences: {
    pages: { block_display: true },
    navigation: { default_tab: "cycles", hide_in_more_menu: ["modules"] },
  },
  rich_filters: { and: [] },
  display_filters: {},
  display_properties: {},
  sort_order: 1,
};

function setHarness(overrides = {}) {
  globalThis.projectPreferencesConsumersHarness = {
    fetchProjectDetails: async () => ({ id: scope.projectId }),
    ...overrides,
  };
}

test.beforeEach((t) => {
  t.mock.method(ProjectService.prototype, "getProject", (workspaceSlug, projectId) =>
    globalThis.projectPreferencesConsumersHarness.fetchProjectDetails(workspaceSlug, projectId)
  );
});

async function seedScope(client, { projectRole = 15, workspaceRole, cachedProperties = initialProperties } = {}) {
  client.setQueryData(identityKeys.currentUser(), { id: scope.accountId });
  await seedPermissionQueries(client, {
    userId: scope.accountId,
    workspaceSlug: scope.workspaceSlug,
    workspaceRole,
    projectRoles: projectRole === null ? {} : { [scope.projectId]: projectRole },
  });
  if (cachedProperties !== null) client.setQueryData(projectUserPropertiesKeys.detail(scope), cachedProperties);
}

async function setupQueuedTabPreferenceWrites(t) {
  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-1/issues/");
  await seedScope(fixture.client);
  t.mock.method(ProjectService.prototype, "getProjectUserProperties", async () => initialProperties);
  const firstStarted = Promise.withResolvers();
  const secondStarted = Promise.withResolvers();
  const firstWrite = Promise.withResolvers();
  const secondWrite = Promise.withResolvers();
  const payloads = [];
  let writes = 0;
  t.mock.method(ProjectService.prototype, "updateProjectUserProperties", (_slug, _project, payload) => {
    payloads.push(payload);
    writes++;
    if (writes === 1) {
      firstStarted.resolve();
      return firstWrite.promise;
    }
    secondStarted.resolve();
    return secondWrite.promise;
  });
  t.mock.method(console, "error", () => undefined);
  await fixture.render(React.createElement(TabProbe));

  return { fixture, firstStarted, secondStarted, firstWrite, secondWrite, payloads };
}

function TabProbe() {
  const preferences = useTabPreferences(scope.workspaceSlug, scope.projectId);
  return React.createElement(
    "section",
    null,
    React.createElement("output", { "data-testid": "preferences" }, JSON.stringify(preferences.tabPreferences)),
    React.createElement("button", { onClick: () => preferences.handleHideTab("cycles") }, "Hide cycles"),
    React.createElement("button", { onClick: () => preferences.handleToggleDefaultTab("pages") }, "Default pages"),
    React.createElement("button", { onClick: () => preferences.handleHideTab("pages") }, "Hide pages")
  );
}

await test("an accepted preference save updates only its captured project cache after route replacement", async (t) => {
  setHarness();
  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-1/issues/");
  await seedScope(fixture.client, { workspaceRole: EUserWorkspaceRoles.MEMBER });
  t.mock.method(ProjectService.prototype, "getProjectUserProperties", async () => initialProperties);
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  t.mock.method(ProjectService.prototype, "updateProjectUserProperties", () => {
    started.resolve();
    return response.promise;
  });
  let preferences;
  function PreferenceProbe() {
    const value = useProjectIssuePreferencesContext();
    React.useEffect(() => {
      preferences = value;
    }, [value]);
    return null;
  }
  await fixture.render(
    React.createElement(
      ProjectAuthWrapper,
      { workspaceSlug: scope.workspaceSlug, projectId: scope.projectId },
      React.createElement(PreferenceProbe)
    )
  );
  await settleQuery();
  assert.ok(preferences);
  let pending;
  await act(async () => {
    pending = preferences.save({ display_filters: { group_by: "priority" } });
    await started.promise;
  });
  await act(async () => {
    await fixture.router.navigate({ href: "/acme/projects/another-project/issues/" });
  });
  await fixture.render(React.createElement("main", null, "Replacement project"));
  const replacementScope = { ...scope, projectId: "another-project" };
  const replacementProperties = { ...initialProperties, display_filters: { group_by: "assignee" } };
  fixture.client.setQueryData(projectUserPropertiesKeys.detail(replacementScope), replacementProperties);
  const saved = { ...initialProperties, display_filters: { group_by: "priority" } };
  await act(async () => {
    response.resolve(saved);
    assert.deepEqual(await pending, saved);
  });
  assert.deepEqual(fixture.client.getQueryData(projectUserPropertiesKeys.detail(scope)), saved);
  assert.deepEqual(
    fixture.client.getQueryData(projectUserPropertiesKeys.detail(replacementScope)),
    replacementProperties
  );
  await settleQuery();
});

await test("active project wrapper loads native preferences only after project access and details are ready", async (t) => {
  const detailsStarted = Promise.withResolvers();
  const detailsResponse = Promise.withResolvers();
  setHarness({
    fetchProjectDetails: () => {
      detailsStarted.resolve();
      return detailsResponse.promise;
    },
  });
  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-1/issues/");
  await seedScope(fixture.client, {
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRole: EUserProjectRoles.MEMBER,
    cachedProperties: null,
  });
  let reads = 0;
  t.mock.method(ProjectService.prototype, "getProjectUserProperties", async (slug, projectId) => {
    reads++;
    assert.equal(slug, scope.workspaceSlug);
    assert.equal(projectId, scope.projectId);
    return initialProperties;
  });

  await fixture.render(
    React.createElement(
      ProjectAuthWrapper,
      { workspaceSlug: scope.workspaceSlug, projectId: scope.projectId },
      React.createElement("main", null, "Project content")
    )
  );
  await detailsStarted.promise;
  assert.equal(reads, 0);
  assert.equal(fixture.environment.container.querySelector("main"), null);
  await act(async () => {
    detailsResponse.resolve({ id: scope.projectId });
    await settleQuery();
  });
  await settleQuery();

  assert.equal(fixture.environment.container.querySelector("main")?.textContent, "Project content");
  assert.equal(reads, 1);
  assert.deepEqual(fixture.client.getQueryData(projectUserPropertiesKeys.detail(scope)), initialProperties);
});

await test("tab preferences observe cache and pending patches without fetching per project row", async (t) => {
  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-1/issues/");
  await seedScope(fixture.client);
  let reads = 0;
  t.mock.method(ProjectService.prototype, "getProjectUserProperties", async () => {
    reads++;
    return initialProperties;
  });
  const started = Promise.withResolvers();
  const saved = Promise.withResolvers();
  let writePayload;
  t.mock.method(ProjectService.prototype, "updateProjectUserProperties", (_slug, _project, payload) => {
    writePayload = payload;
    started.resolve();
    return saved.promise;
  });
  await fixture.render(React.createElement(TabProbe));

  const output = fixture.environment.container.querySelector('[data-testid="preferences"]');
  assert.deepEqual(JSON.parse(output.textContent), { defaultTab: "cycles", hiddenTabs: ["modules"] });
  await act(async () => {
    fixture.environment.container.querySelector("button").click();
    await started.promise;
    await settleQuery();
  });
  assert.deepEqual(JSON.parse(output.textContent), { defaultTab: "cycles", hiddenTabs: ["modules", "cycles"] });
  assert.equal(reads, 0);

  await act(async () => {
    saved.resolve({
      ...initialProperties,
      preferences: {
        ...initialProperties.preferences,
        navigation: { default_tab: "cycles", hide_in_more_menu: ["modules", "cycles"] },
      },
    });
    await settleQuery();
  });
  assert.deepEqual(writePayload.preferences, {
    pages: { block_display: true },
    navigation: { default_tab: "cycles", hide_in_more_menu: ["modules", "cycles"] },
  });
  assert.equal(reads, 0);
});

await test("tab preference read and writes fail closed when project role access is unavailable", async (t) => {
  globalThis.projectPreferencesConsumersToasts.length = 0;
  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-1/issues/");
  await seedScope(fixture.client, { projectRole: null });
  let reads = 0;
  let writes = 0;
  t.mock.method(ProjectService.prototype, "getProjectUserProperties", async () => {
    reads++;
    return initialProperties;
  });
  t.mock.method(ProjectService.prototype, "updateProjectUserProperties", async () => {
    writes++;
    return initialProperties;
  });
  t.mock.method(console, "error", () => undefined);
  await fixture.render(React.createElement(TabProbe));

  assert.deepEqual(JSON.parse(fixture.environment.container.querySelector('[data-testid="preferences"]').textContent), {
    defaultTab: "work_items",
    hiddenTabs: [],
  });
  await act(async () => {
    fixture.environment.container.querySelector("button").click();
    await settleQuery();
  });
  assert.equal(reads, 0);
  assert.equal(writes, 0);
  assert.equal(globalThis.projectPreferencesConsumersToasts.at(-1)?.type, "error");

  setHarness();
  await fixture.render(
    React.createElement(
      ProjectAuthWrapper,
      { workspaceSlug: scope.workspaceSlug, projectId: scope.projectId },
      React.createElement("main", null, "Project content")
    )
  );
  await settleQuery();
  assert.equal(fixture.environment.container.querySelector("main"), null);
  assert.equal(reads, 0);
});

await test("failed tab preference writes remove the pending overlay and report an error", async (t) => {
  globalThis.projectPreferencesConsumersToasts.length = 0;
  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-1/issues/");
  await seedScope(fixture.client);
  t.mock.method(ProjectService.prototype, "getProjectUserProperties", async () => initialProperties);
  t.mock.method(ProjectService.prototype, "updateProjectUserProperties", async () => {
    throw new Error("write rejected");
  });
  await fixture.render(React.createElement(TabProbe));

  await act(async () => {
    fixture.environment.container.querySelectorAll("button")[1].click();
    await settleQuery();
  });

  assert.deepEqual(JSON.parse(fixture.environment.container.querySelector('[data-testid="preferences"]').textContent), {
    defaultTab: "cycles",
    hiddenTabs: ["modules"],
  });
  assert.equal(globalThis.projectPreferencesConsumersToasts.at(-1)?.type, "error");
});

await test("a rejected hide intent cannot leak into a later accepted hide intent", async (t) => {
  const { fixture, firstStarted, secondStarted, firstWrite, secondWrite, payloads } =
    await setupQueuedTabPreferenceWrites(t);

  await act(async () => {
    fixture.environment.container.querySelectorAll("button")[0].click();
    await firstStarted.promise;
  });
  await act(async () => {
    fixture.environment.container.querySelectorAll("button")[2].click();
    await settleQuery();
  });
  await act(async () => {
    firstWrite.reject(new Error("first hide rejected"));
    await secondStarted.promise;
    await settleQuery();
  });
  assert.deepEqual(JSON.parse(fixture.environment.container.querySelector('[data-testid="preferences"]').textContent), {
    defaultTab: "cycles",
    hiddenTabs: ["modules", "pages"],
  });
  assert.deepEqual(payloads[1].preferences.navigation, {
    default_tab: "cycles",
    hide_in_more_menu: ["modules", "pages"],
  });

  await act(async () => {
    secondWrite.resolve({
      ...initialProperties,
      preferences: {
        ...initialProperties.preferences,
        navigation: { default_tab: "cycles", hide_in_more_menu: ["modules", "pages"] },
      },
    });
    await settleQuery();
  });
  assert.deepEqual(fixture.client.getQueryData(projectUserPropertiesKeys.detail(scope)).preferences.navigation, {
    default_tab: "cycles",
    hide_in_more_menu: ["modules", "pages"],
  });
});

await test("a failed hide patch preserves an accepted default-tab change", async (t) => {
  const { fixture, firstStarted, secondStarted, firstWrite, secondWrite, payloads } =
    await setupQueuedTabPreferenceWrites(t);

  await act(async () => {
    fixture.environment.container.querySelectorAll("button")[1].click();
    await firstStarted.promise;
  });
  await act(async () => {
    fixture.environment.container.querySelectorAll("button")[0].click();
    await Promise.resolve();
    firstWrite.resolve({
      ...initialProperties,
      preferences: {
        ...initialProperties.preferences,
        navigation: { default_tab: "pages", hide_in_more_menu: ["modules"] },
      },
    });
    await secondStarted.promise;
  });
  assert.deepEqual(payloads[0].preferences, {
    pages: { block_display: true },
    navigation: { default_tab: "pages", hide_in_more_menu: ["modules"] },
  });
  assert.deepEqual(payloads[1].preferences, {
    pages: { block_display: true },
    navigation: { default_tab: "pages", hide_in_more_menu: ["modules", "cycles"] },
  });

  await act(async () => {
    secondWrite.reject(new Error("hide rejected"));
    await settleQuery();
  });
  assert.deepEqual(fixture.client.getQueryData(projectUserPropertiesKeys.detail(scope)).preferences.navigation, {
    default_tab: "pages",
    hide_in_more_menu: ["modules"],
  });
});

await test("preference read failures expose retry and block tab writes until the canonical read succeeds", async (t) => {
  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-1/issues/");
  await seedScope(fixture.client, { cachedProperties: null });
  let reads = 0;
  let writes = 0;
  t.mock.method(ProjectService.prototype, "getProjectUserProperties", async () => {
    reads++;
    if (reads === 1) throw new Error("preference read failed");
    return initialProperties;
  });
  t.mock.method(ProjectService.prototype, "updateProjectUserProperties", async () => {
    writes++;
    return initialProperties;
  });
  await fixture.render(
    React.createElement(
      ProjectAuthWrapper,
      { workspaceSlug: scope.workspaceSlug, projectId: scope.projectId },
      React.createElement(TabProbe)
    )
  );
  await settleQuery();

  const findButton = (label) =>
    [...fixture.environment.container.querySelectorAll("button")].find((button) => button.textContent === label);
  assert.equal(reads, 1);
  assert.ok(
    fixture.environment.container
      .querySelector('[role="alert"]')
      ?.textContent.includes("Unable to load project preferences")
  );
  await act(async () => {
    findButton("Default pages").click();
    await settleQuery();
  });
  assert.equal(writes, 0);

  await act(async () => {
    findButton("Retry").click();
    await settleQuery();
  });
  assert.equal(reads, 2);
  assert.equal(fixture.environment.container.querySelector('[role="alert"]'), null);
  await act(async () => {
    findButton("Default pages").click();
    await settleQuery();
  });
  assert.equal(writes, 1);
});
