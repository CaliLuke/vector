import assert from "node:assert/strict";
import test from "node:test";
import { React, act, createPermissionUiFixture } from "./helpers/permission-ui-fixture.mjs";
import { seedParentListPreferences } from "./helpers/parent-catalogue-fixture.mjs";

const { ParentIssuePreferencesProvider, useParentIssuePreferences } =
  await import("../../apps/web/core/providers/parent-issue-preferences.tsx");
const { IssueFiltersService } = await import("../../apps/web/core/services/issue_filter.service.ts");

const { EstimateService } = await import("../../apps/web/core/services/estimate.service.ts");
const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");

for (const domain of ["cycle", "module"]) {
  void test(`${domain} preference edits preserve failed drafts, retry the native write and normalize grouping`, async (t) => {
    const view = await createPermissionUiFixture(t, {
      path: `/acme/projects/p1/${domain}s/parent/`,
      workspaceRole: 15,
      projectRoles: { p1: 20 },
    });
    t.mock.method(EstimateService.prototype, "fetchProjectEstimates", async () => []);
    t.mock.method(ProjectService.prototype, "getProject", async () => ({ id: "p1", estimate: null }));
    let canonical = await seedParentListPreferences(t, view.client, domain, "parent", {
      richFilters: {},
      displayFilters: { layout: "list" },
      displayProperties: {},
    });
    let fail = true;
    const writes = [];
    t.mock.method(
      IssueFiltersService.prototype,
      domain === "cycle" ? "patchCycleIssueFilters" : "patchModuleIssueFilters",
      async (workspace, project, parent, patch) => {
        writes.push({ workspace, project, parent, patch });
        if (fail) throw new Error("offline");
        canonical = { ...canonical, ...patch };
        return canonical;
      }
    );
    let current;
    function Probe() {
      const value = useParentIssuePreferences();
      React.useLayoutEffect(() => {
        current = value;
      }, [value]);
      return null;
    }
    await view.render(
      React.createElement(
        ParentIssuePreferencesProvider,
        { domain, workspaceSlug: "acme", projectId: "p1", parentId: "parent" },
        React.createElement(Probe)
      )
    );
    assert.ok(current.requestInput?.isScopeCurrent());
    const rich = { priority__in: "high" };
    await act(async () => {
      await assert.rejects(current.setRich(rich), /offline/);
    });
    assert.equal(current.isSaveError, true);
    assert.deepEqual(current.filters.richFilters, rich);
    assert.equal(current.isRichDirty, true);
    fail = false;
    await act(async () => current.retrySave());
    assert.equal(current.isSaveError, false);
    assert.equal(current.isRichDirty, false);
    assert.deepEqual(
      writes.slice(0, 2),
      Array.from({ length: 2 }, () => ({
        workspace: "acme",
        project: "p1",
        parent: "parent",
        patch: { rich_filters: rich },
      }))
    );
    await act(async () => current.setRich(rich));
    assert.equal(writes.length, 2, "an unchanged filter does not repeat a write");
    await act(async () => current.setDisplay({ layout: "kanban", group_by: null, sub_group_by: "priority" }));
    assert.equal(current.filters.displayFilters.group_by, "state");
    assert.equal(current.filters.displayFilters.sub_group_by, null);
    await act(async () => current.setDisplay({ group_by: "priority", sub_group_by: "priority" }));
    assert.equal(current.filters.displayFilters.sub_group_by, null);
    await act(async () => current.setProperties({ key: false }));
    assert.equal(current.filters.displayProperties.key, false);
    await act(async () => current.setKanban({ group_by: ["completed"] }));
    assert.deepEqual(current.filters.kanbanFilters.group_by, ["completed"]);
    const stale = current;
    await view.environment.render(null);
    await assert.rejects(stale.setRich({ priority__in: "urgent" }), /editor was replaced/);
    const acceptedWrites = writes.length;
    await act(async () => stale.setKanban({ group_by: ["backlog"] }));
    assert.equal(writes.length, acceptedWrites);
  });
}
