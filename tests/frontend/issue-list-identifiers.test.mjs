import assert from "node:assert/strict";
import test from "node:test";
import { React, registerPermissionUiStubs } from "./helpers/permission-ui-fixture.mjs";
import { createQueryRouterFixture } from "./helpers/query-router.mjs";

registerPermissionUiStubs("/list/block.tsx", {
  "@/hooks/use-projects": `
    export const useRouteProject = () => ({});
    export const useWorkspaceProjectDetails = () => ({ projects: [
      { id: "p1", identifier: "VEC" }, { id: "p2", identifier: "WEB" },
    ] });`,
  "@/providers/sidebar-state": "export const useSidebarCollapsed = () => false;",
  "@/providers/issue-detail-ui-state":
    "export const useIssueDetailUiState = () => ({ getIsIssuePeeked: () => false });",
  "@/hooks/use-platform-os": "export const usePlatformOS = () => ({ isMobile: false });",
  "@/components/core/multiple-select": "export const MultipleSelectEntityAction = () => null;",
  "@/components/issues/issue-layouts/properties": "export const IssueProperties = () => null;",
});

registerPermissionUiStubs("/issue-detail/issue-identifier.tsx", {
  "@/hooks/use-projects": "export const useWorkspaceProjectDetails = () => ({ projects: [] });",
  "@/hooks/use-work-item-detail": `export const useWorkItemDetail = (_, __, id, ___, enabled) => {
    if (id || enabled) throw new Error("An issue list row must not request issue details");
    return {};
  };`,
  "@/components/work-item-types/work-item-type-badge": "export const WorkItemTypeBadge = () => null;",
});

await test("published list rows render their IDs with the titles and links before any detail reads", async (t) => {
  const fixture = await createQueryRouterFixture(t, "/acme/projects/p1/issues/");
  const { IssueBlock } = await import("../../apps/web/core/components/issues/issue-layouts/list/block.tsx");
  const issuesMap = {
    first: { id: "first", project_id: "p1", sequence_id: 36, name: "First listed item" },
    second: { id: "second", project_id: "p2", sequence_id: 7, name: "Cross-project item" },
    pending: { id: "pending", project_id: "p1", sequence_id: 37, name: "Pending item", tempId: "pending" },
  };
  const render = (key) =>
    fixture.render(
      React.createElement(
        React.Fragment,
        null,
        ...Object.keys(issuesMap).map((issueId) =>
          React.createElement(IssueBlock, {
            key: issueId,
            issueId,
            issuesMap,
            displayProperties: { key },
            canEditProperties: () => false,
            selectionHelpers: { getIsEntitySelected: () => false, getIsEntityActive: () => false },
            quickActions: () => null,
            nestingLevel: 0,
          })
        )
      )
    );
  const { container } = fixture.environment;

  await render(true);
  for (const [identifier, title] of [
    ["VEC-36", "First listed item"],
    ["WEB-7", "Cross-project item"],
  ]) {
    const row = container.querySelector(`a[href="/acme/browse/${identifier}/"]`);
    assert.ok(row);
    assert.ok(row.textContent.includes(identifier));
    assert.ok(row.textContent.includes(title));
  }
  assert.match(container.textContent, /Pending item/);
  assert.doesNotMatch(container.textContent, /VEC-37/);

  await render(false);
  assert.doesNotMatch(container.textContent, /VEC-36|WEB-7/);
  assert.equal(container.querySelectorAll('a[href*="/acme/browse/"]').length, 2);
});
