import assert from "node:assert/strict";
import test from "node:test";
import {
  React,
  act,
  createPermissionUiFixture,
  registerPermissionUiStubs,
  settleQuery,
} from "./helpers/permission-ui-fixture.mjs";

registerPermissionUiStubs("/work-item-filters/filters-hoc/project-level.tsx", {
  "@/components/issues/issue-layouts/utils": "export const removeNillKeys = value => value ?? {};",
  "@/components/views/modal": "export const CreateUpdateProjectViewModal = () => null;",
  "@/hooks/store/use-project": "export const useProject = () => ({getProjectById: () => ({issue_views_view:true})});",
  "@/hooks/use-project-states": "export const useProjectStates = () => ({getProjectStateIds: () => []});",
  "@/hooks/use-labels": "export const useProjectLabels = () => ({labelIds: []});",
  "@/hooks/use-member-rosters":
    "export const useProjectMemberRoster = () => ({memberships:[]}); export const useWorkspaceMemberRoster = () => ({membersById:{}});",
  "@plane/propel/toast":
    "export const TOAST_TYPE = {SUCCESS:'success',ERROR:'error'}; export const setToast = () => {};",
});
registerPermissionUiStubs("/work-item-filters/filters-hoc/use-view-filter-actions.ts", {
  "@/components/issues/issue-layouts/utils":
    "export const removeNillKeys = value => Object.fromEntries(Object.entries(value ?? {}).filter(([,entry]) => entry != null));",
});
registerPermissionUiStubs("/work-item-filters/filters-hoc/base.tsx", {
  "@/hooks/work-item-filters/use-work-item-filters-config":
    "export const useWorkItemFiltersConfig = () => ({configs:[],areAllConfigsInitialized:true,configMap:{},isFilterEnabled:()=>false,members:[],catalogError:false,retryCatalogs:async()=>{}});",
});
const { ProjectLevelWorkItemFiltersHOC } =
  await import("../../apps/web/core/components/work-item-filters/filters-hoc/project-level.tsx");
const { WorkItemFilterUiProvider } = await import("../../apps/web/core/providers/work-item-filter-ui.tsx");
const { ViewService } = await import("../../apps/web/core/services/view.service.ts");

await test("the project filter row retains unsaved and newer edits until acknowledged", async (t) => {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/views/view",
    workspaceRole: 15,
    projectRoles: { p1: 15 },
  });
  const getCachedView = () =>
    fixture.client
      .getQueryCache()
      .findAll({ queryKey: ["project-views"] })
      .find((query) => query.queryKey.at(-2) === "detail" && query.queryKey.at(-1) === "view")?.state.data;
  const saved = {
    id: "view",
    project: "p1",
    name: "Saved",
    owned_by: "alice",
    is_locked: false,
    rich_filters: { priority__exact: "low" },
  };
  let serverView = saved;
  t.mock.method(ViewService.prototype, "getViewDetails", async () => serverView);
  let editor;
  function Probe({ value }) {
    React.useLayoutEffect(() => {
      editor = value;
    });
    return null;
  }
  await fixture.render(
    React.createElement(
      WorkItemFilterUiProvider,
      { scopeKey: "alice:acme:p1:view" },
      React.createElement(
        ProjectLevelWorkItemFiltersHOC,
        {
          entityType: "PROJECT_VIEW",
          entityId: "view",
          workspaceSlug: "acme",
          projectId: "p1",
          enableUpdateView: true,
          filtersToShowByLayout: [],
          initialWorkItemFilters: { richFilters: saved.rich_filters, displayFilters: {}, displayProperties: {} },
          updateFilters: () => {},
        },
        ({ editor: current }) => React.createElement(Probe, { value: current })
      )
    )
  );
  await settleQuery();
  const id = editor.conditions[0].id;
  await act(async () => {
    editor.onUpdateConditionValue(id, "high");
  });
  assert.equal(editor.canUpdate, true);
  const failed = Promise.withResolvers();
  const transport = t.mock.method(ViewService.prototype, "patchView", () => failed.promise);
  let rejected;
  await act(async () => {
    rejected = assert.rejects(editor.onUpdate(), /offline/);
  });
  assert.equal(editor.hasChanges, true);
  await act(async () => {
    failed.reject(new Error("offline"));
    await rejected;
  });
  assert.equal(editor.canUpdate, true);
  assert.deepEqual(getCachedView()?.rich_filters, saved.rich_filters);
  const accepted = Promise.withResolvers();
  transport.mock.mockImplementation(() => accepted.promise);
  let request;
  await act(async () => {
    request = editor.onUpdate();
  });
  await act(async () => {
    editor.onUpdateConditionValue(id, "urgent");
  });
  await act(async () => {
    serverView = { ...saved, rich_filters: { priority__exact: "high" } };
    accepted.resolve(serverView);
    await request;
  });
  assert.equal(editor.hasChanges, true);
  assert.equal(editor.conditions[0].value, "urgent");
  assert.deepEqual(getCachedView()?.rich_filters, { priority__exact: "high" });
});

await test("project issue filters do not request a view using the project ID", async (t) => {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues",
    workspaceRole: 15,
    projectRoles: { p1: 15 },
  });
  const readView = t.mock.method(ViewService.prototype, "getViewDetails", async () => {
    throw new Error("a project issue route has no saved-view ID");
  });

  await fixture.render(
    React.createElement(
      WorkItemFilterUiProvider,
      { scopeKey: "alice:acme:p1:project:p1" },
      React.createElement(
        ProjectLevelWorkItemFiltersHOC,
        {
          entityType: "PROJECT",
          entityId: "p1",
          workspaceSlug: "acme",
          projectId: "p1",
          filtersToShowByLayout: [],
          initialWorkItemFilters: { richFilters: {}, displayFilters: {}, displayProperties: {} },
          updateFilters: () => {},
        },
        () => null
      )
    )
  );
  await settleQuery();

  assert.equal(readView.mock.callCount(), 0);
});
