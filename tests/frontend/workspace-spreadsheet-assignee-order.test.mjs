import { registerHooks } from "node:module";
import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { React } = await import("./helpers/react.mjs");
const { createQueryRouterFixture } = await import("./helpers/query-router.mjs");
const { seedPermissionQueries } = await import("./helpers/permission-query-fixture.mjs");
const { EUserWorkspaceRoles } = await import("../../packages/types/dist/index.mjs");
const { ALL_ISSUES } = await import("../../packages/constants/dist/index.mjs");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { memberRosterKeys } = await import("../../apps/web/core/queries/member-rosters.ts");
const captured = `export function SpreadsheetView(props) { globalThis.workspaceSpreadsheetProps = props; return null; }`;
const stubs = {
  "@/components/estimates/catalogue-boundary":
    "export function EstimateCatalogueBoundary({children}) { return children; }",
  "@/hooks/use-issue-estimate-projection":
    "export function useIssueEstimateOrder() { return { projection: undefined, isPending: false, isError: false, retry: async () => {} }; }",
  "@/components/issues/issue-layouts/quick-action-dropdowns": "export function AllIssueQuickActions() { return null; }",
  "@/components/ui/loader/layouts/spreadsheet-layout-loader":
    "export function SpreadsheetLayoutLoader() { return null; }",
  "@/hooks/use-workspace-issue-properties": "export function useWorkspaceIssueProperties() {}",
  "@/hooks/use-issues-actions":
    "export function useIssuesActions() { return { removeIssue: async () => {}, updateIssue: async () => {}, archiveIssue: async () => {} }; }",
  "@/providers/workspace-issue-preferences":
    "export function useWorkspaceIssuePreferences() { return globalThis.workspacePreferenceFixture; }",
  "../spreadsheet-view": captured,
  "../../issue-layout-HOC": "export function IssueLayoutHOC({children}) { return children; }",
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    const inRoot = context.parentURL?.endsWith("/spreadsheet/roots/workspace-root.tsx");
    if (inRoot && stubs[specifier])
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(stubs[specifier])}` };
    return nextResolve(specifier, context);
  },
});
const { WorkspaceSpreadsheetRoot } =
  await import("../../apps/web/core/components/issues/issue-layouts/spreadsheet/roots/workspace-root.tsx");

function issue(id, assigneeIds) {
  return { id, assignee_ids: assigneeIds, created_at: "2026-10-25T00:00:00Z", project_id: "p1" };
}

await test("workspace spreadsheet projects assignee order from the scoped native roster and Query rows", async (t) => {
  const view = await createQueryRouterFixture(t, "/acme/workspace-views/view-a");
  t.after(() => {
    delete globalThis.workspaceSpreadsheetProps;
    delete globalThis.workspacePreferenceFixture;
  });
  const scope = {
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: EUserWorkspaceRoles.MEMBER,
    membershipRevision: "2026-10-24T00:00:00Z",
  };
  view.client.setQueryData(identityKeys.currentUser(), { id: "actor" });
  await seedPermissionQueries(view.client, {
    userId: "actor",
    workspaceSlug: "acme",
    workspaceRole: scope.membershipRole,
  });
  view.client.setQueryData(memberRosterKeys.workspace(scope), [
    { id: "row-bob", member: { id: "bob", first_name: "Bob", display_name: "Bob" } },
    { id: "row-alice", member: { id: "alice", first_name: "Alice", display_name: "Alice" } },
  ]);
  globalThis.workspacePreferenceFixture = {
    issueFilters: {
      displayFilters: { layout: "spreadsheet", order_by: "assignees__first_name" },
      displayProperties: {},
    },
    updateFilters: async () => {},
  };
  const issuesById = {
    unassigned: issue("unassigned", []),
    bob: issue("bob", ["bob"]),
    alice: issue("alice", ["alice"]),
  };
  const readState = {
    issuesById,
    groupedIssueIds: { [ALL_ISSUES]: ["unassigned", "bob", "alice"] },
    labelsById: new Map(),
    stateProjection: undefined,
    parentProjection: undefined,
    isPending: false,
    isError: false,
    retry: async () => {},
    issueCount: 3,
    getPaginationData: () => ({ nextPageResults: false }),
    getGroupIssueCount: () => 3,
    getIssueLoader: () => undefined,
    fetchNextIssues: async () => {},
  };

  await view.render(
    React.createElement(WorkspaceSpreadsheetRoot, {
      isDefaultView: false,
      workspaceSlug: "acme",
      globalViewId: "view-a",
      routeFilters: {},
      readState,
      globalViewsLoading: false,
    })
  );

  assert.deepEqual(globalThis.workspaceSpreadsheetProps.issueIds, ["alice", "bob", "unassigned"]);
});
