import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
import assert from "node:assert/strict";

import test from "node:test";

import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";

registerWebTestModules({
  "@plane/constants": `
          export const EIssueFilterType = { DISPLAY_FILTERS: "display_filters", DISPLAY_PROPERTIES: "display_properties" };
          export const ISSUE_DISPLAY_FILTERS_BY_PAGE = { profile_issues: { layoutOptions: { list: { source: "profile-list" } } } };
        `,
  "@plane/types": `export const EIssuesStoreType = { PROFILE: "PROFILE", PROJECT: "PROJECT" };`,
  "@tanstack/react-router": `export const useParams = () => globalThis.profileIssueParams;`,
  "@/providers/local-issue-preferences": `export const useLocalIssuePreferences = () => globalThis.profileIssuePreferences;`,
  "@/hooks/use-issue-filter-preferences": `export const useIssueFilterPreferences = (storeType) => { globalThis.projectStoreType = storeType; return globalThis.projectIssuePreferences; };`,
});

await test("shared display preference hooks retain profile and project write scopes", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const [{ useProfileIssueDisplayPreferences }, { useProjectIssueDisplayPreferences }] = await Promise.all([
    import("../../apps/web/core/hooks/use-profile-issue-display-preferences.ts"),
    import("../../apps/web/core/hooks/use-project-issue-display-preferences.ts"),
  ]);
  const profileWrites = [];
  const projectWrites = [];
  globalThis.profileIssueParams = { workspaceSlug: "acme", userId: "alice" };
  globalThis.profileIssuePreferences = {
    domain: "PROFILE",
    issueFilters: { displayFilters: { layout: "list" }, displayProperties: { priority: true } },
    updateFilters: (...args) => profileWrites.push(args),
  };
  globalThis.projectIssuePreferences = {
    issueFilters: { displayFilters: { layout: "kanban" }, displayProperties: { priority: false } },
    projectPreferences: { save: (value) => projectWrites.push(value) },
  };

  function ProfileActions() {
    const preferences = useProfileIssueDisplayPreferences();
    return React.createElement(
      "div",
      null,
      React.createElement("button", { id: "profile-layout", onClick: () => preferences.handleLayoutChange("kanban") }),
      React.createElement("button", {
        id: "profile-display",
        onClick: () => preferences.handleDisplayFilters({ group_by: "priority" }),
      }),
      React.createElement("button", {
        id: "profile-properties",
        onClick: () => preferences.handleDisplayProperties({ priority: false }),
      }),
      React.createElement(
        "output",
        { id: "profile-scope" },
        JSON.stringify({
          layout: preferences.activeLayout,
          options: preferences.layoutDisplayFiltersOptions,
        })
      )
    );
  }

  function ProjectActions({ projectId }) {
    const preferences = useProjectIssueDisplayPreferences({ workspaceSlug: "acme", projectId });
    return React.createElement(
      "div",
      null,
      React.createElement("button", { id: "project-layout", onClick: () => preferences.handleLayoutChange("list") }),
      React.createElement("button", {
        id: "project-display",
        onClick: () => preferences.handleDisplayFilters({ group_by: "priority" }),
      }),
      React.createElement("button", {
        id: "project-properties",
        onClick: () => preferences.handleDisplayProperties({ priority: true }),
      })
    );
  }

  async function clickDisplayActions(prefix) {
    await React.act(() => environment.container.querySelector(`#${prefix}-layout`).click());
    await React.act(() => environment.container.querySelector(`#${prefix}-display`).click());
    await React.act(() => environment.container.querySelector(`#${prefix}-properties`).click());
  }

  await environment.render(React.createElement(ProfileActions));
  assert.deepEqual(JSON.parse(environment.container.querySelector("#profile-scope").textContent), {
    layout: "list",
    options: { source: "profile-list" },
  });
  await clickDisplayActions("profile");
  assert.deepEqual(profileWrites, [
    ["display_filters", { layout: "kanban" }],
    ["display_filters", { group_by: "priority" }],
    ["display_properties", { priority: false }],
  ]);

  await environment.render(React.createElement(ProjectActions, { projectId: "project-1" }));
  assert.equal(globalThis.projectStoreType, "PROJECT");
  await clickDisplayActions("project");
  assert.deepEqual(projectWrites, [
    { display_filters: { layout: "list" } },
    { display_filters: { group_by: "priority" } },
    { display_properties: { priority: true } },
  ]);

  await environment.render(React.createElement(ProjectActions, { projectId: undefined }));
  await clickDisplayActions("project");
  assert.equal(projectWrites.length, 3, "updates require both workspace and project scope");
});
