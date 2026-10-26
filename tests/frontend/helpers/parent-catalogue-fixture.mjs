import { createPermissionQueryClient } from "./permission-query-fixture.mjs";
export const parentScope = {
  accountId: "alice",
  sessionGeneration: 0,
  workspaceSlug: "acme",
  workspaceRole: 15,
  membershipRevision: "2026-10-24T00:00:00Z",
  projectId: "p1",
  projectMembershipRole: 20,
  projectRole: 20,
};
export async function parentCatalogueFixture(t, scope = parentScope) {
  return createPermissionQueryClient(t, {
    ...scope,
    membershipRole: scope.workspaceRole,
    workspaceRole: scope.workspaceRole,
  });
}
export function parentRow(id, extra = {}) {
  return { id, project_id: "p1", workspace_id: "w1", name: id, sort_order: 1, archived_at: null, ...extra };
}

export async function seedParentListPreferences(t, client, domain, parentId, preferences) {
  const { currentParentCatalogueScope } = await import("../../../apps/web/core/queries/cycle-module-scope.ts");
  const { parentIssuePreferenceKey } = await import("../../../apps/web/core/queries/parent-issue-preferences.ts");
  const { IssueFiltersService } = await import("../../../apps/web/core/services/issue_filter.service.ts");
  const { ModuleService } = await import("../../../apps/web/core/services/module.service.ts");
  const { CycleService } = await import("../../../apps/web/core/services/cycle.service.ts");
  const scope = currentParentCatalogueScope(client, "acme", "p1").project;
  const canonical = {
    rich_filters: preferences.richFilters,
    display_filters: preferences.displayFilters,
    display_properties: preferences.displayProperties,
  };
  client.setQueryData(parentIssuePreferenceKey(domain, scope, parentId), canonical);
  t.mock.method(
    IssueFiltersService.prototype,
    domain === "cycle" ? "fetchCycleIssueFilters" : "fetchModuleIssueFilters",
    async () => canonical
  );
  t.mock.method(
    domain === "cycle" ? CycleService.prototype : ModuleService.prototype,
    domain === "cycle" ? "retrieve" : "getModuleDetails",
    async () => parentRow(parentId)
  );
  return canonical;
}
