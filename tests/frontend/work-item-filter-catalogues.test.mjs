import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

const fixture = {
  calls: [],
  projectStates: {
    states: undefined,
    isPending: true,
    refetch: async () => fixture.calls.push("state"),
    retryAccess: async () => fixture.calls.push("state access"),
  },
  projectLabels: [{ id: "project-label", name: "Project label", color: "#fff" }],
  workspaceLabels: [
    { id: "workspace-label", name: "Workspace label", workspace_id: "w1" },
    { id: "other-label", name: "Other label", workspace_id: "w2" },
  ],
};
globalThis.vectorFilterCatalogues = fixture;
registerWebTestModules({
  "@/hooks/use-projects":
    "export const useWorkspaceProjectDetails = () => ({ projects: [], isError: false, accessDenied: false });",
  "@/hooks/use-member-rosters": "export const useWorkspaceMemberRoster = () => ({});",
  "@/hooks/use-project-states":
    "export const useProjectStates = () => globalThis.vectorFilterCatalogues.projectStates;",
  "@/hooks/use-labels": `
    const fixture = globalThis.vectorFilterCatalogues;
    const refetch = async () => fixture.calls.push("labels");
    export const useProjectLabels = () => ({ labels: fixture.projectLabels, isError: fixture.labelError, refetch });
    export const useWorkspaceLabels = () => ({ labels: fixture.workspaceLabels, isError: false, refetch });
  `,
  "@/hooks/use-cycles": `
    export const useProjectCycles = (_workspace, project) => ({
      cycles: project ? [{ id: "cycle-1", name: "Cycle one" }] : undefined,
      isError: false, isAccessDenied: false,
      retry: async () => globalThis.vectorFilterCatalogues.calls.push("cycle"),
    });
  `,
  "@/hooks/use-modules": `
    export const useProjectModules = (_workspace, project) => ({
      modules: project ? [{ id: "module-1", name: "Module one" }] : undefined,
      isError: false, isAccessDenied: false,
      retry: async () => globalThis.vectorFilterCatalogues.calls.push("module"),
    });
  `,
  "./use-type-filter-config": "export const useTypeFilterConfig = () => ({ config: {}, isError: false });",
  "./use-property-filter-configs": "export const usePropertyFilterConfigs = () => ({ configs: [], isError: false });",
});
const { useWorkItemFiltersConfig } =
  await import("../../apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx");

await test("filter choices belong to the displayed project or workspace catalogue", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const filters = ["state_id", "label_id", "cycle_id", "module_id"];
  let current;
  function Probe({ projectId }) {
    const config = useWorkItemFiltersConfig({
      allowedFilters: filters,
      workspaceSlug: "fixture",
      workspaceId: "w1",
      projectId,
    });
    React.useLayoutEffect(() => {
      current = config;
    }, [config]);
    return null;
  }
  const render = (projectId) => env.render(React.createElement(Probe, { projectId }));
  const optionIds = (filter) =>
    current.configMap[filter].supportedOperatorConfigsMap.get("in").getOptions.map((option) => option.id);

  await render("p1");
  assert.equal(current.areAllConfigsInitialized, false);
  assert.equal(current.configMap.state_id.isEnabled, false);

  fixture.projectStates = {
    ...fixture.projectStates,
    isPending: false,
    states: [{ id: "backlog", name: "Backlog", group: "backlog" }],
  };
  await render("p1");
  assert.equal(current.areAllConfigsInitialized, true);
  assert.deepEqual(optionIds("state_id"), ["backlog"]);
  assert.deepEqual(optionIds("label_id"), ["project-label"]);
  assert.deepEqual(optionIds("cycle_id"), ["cycle-1"]);
  assert.deepEqual(optionIds("module_id"), ["module-1"]);

  fixture.projectStates = { ...fixture.projectStates, states: undefined, isError: true };
  fixture.labelError = true;
  await render("p1");
  assert.equal(current.catalogError, true);
  await current.retryCatalogs();
  assert.deepEqual(
    fixture.calls.toSorted((left, right) => left.localeCompare(right)),
    ["cycle", "labels", "module", "state"]
  );

  await render(undefined);
  assert.equal(current.catalogError, false);
  assert.equal(current.configMap.state_id.isEnabled, false);
  assert.deepEqual(optionIds("label_id"), ["workspace-label"]);
});
