import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act, createPermissionUiFixture, settleQuery } from "./helpers/permission-ui-fixture.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.projectKanbanWriteToasts = [];
registerHooks({
  resolve: (specifier, context, nextResolve) =>
    resolvePackageStub(
      specifier,
      {
        "@plane/propel/toast": `export const TOAST_TYPE={ERROR:"error"};export const setToast=value=>globalThis.projectKanbanWriteToasts.push(value);`,
      },
      context,
      nextResolve
    ),
});

const { useProjectKanbanFilters } = await import("../../apps/web/core/hooks/use-project-kanban-filters.ts");
const { useIssueKanbanCollapse } = await import("../../apps/web/core/hooks/use-issue-kanban-collapse.ts");
const { EIssuesStoreType } = await import("../../packages/types/dist/index.mjs");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");

async function createFixture(t, userId = "viewer") {
  return createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/",
    userId,
    workspaceRole: "member",
    projectRoles: { p1: "member" },
  });
}

function accountScopedCollapseRecords(viewerSubGroups = []) {
  return [
    {
      key: EIssuesStoreType.PROJECT,
      workspaceSlug: "acme",
      viewId: "p1",
      userId: "viewer",
      filters: { kanban_filters: { group_by: ["state"], sub_group_by: viewerSubGroups } },
    },
    {
      key: EIssuesStoreType.PROJECT,
      workspaceSlug: "acme",
      viewId: "p2",
      userId: "viewer",
      filters: { kanban_filters: { group_by: ["priority"], sub_group_by: [] } },
    },
    {
      key: EIssuesStoreType.PROJECT,
      workspaceSlug: "acme",
      viewId: "p1",
      userId: "other",
      filters: { kanban_filters: { group_by: ["assignees"], sub_group_by: [] } },
    },
  ];
}

function Probe({ workspaceSlug, projectId, onUpdate }) {
  const value = useProjectKanbanFilters(workspaceSlug, projectId);
  React.useEffect(() => onUpdate(value), [onUpdate, value]);
  return null;
}

function NonProjectCollapseWriteFailureProbe() {
  const value = useIssueKanbanCollapse(
    EIssuesStoreType.CYCLE,
    "acme",
    "p1",
    { group_by: [], sub_group_by: [] },
    async () => Promise.reject(new Error("Collapse write failed"))
  );
  return React.createElement("button", { onClick: () => value.toggle("group_by", "state") }, "Collapse");
}

function NonProjectCollapseMissingWriterProbe() {
  const value = useIssueKanbanCollapse(EIssuesStoreType.CYCLE, "acme", "p1", undefined, undefined);
  return React.createElement("button", { onClick: () => value.toggle("group_by", "state") }, "Collapse");
}

function NonProjectCollapseEmptyTargetProbe({ onWrite }) {
  const value = useIssueKanbanCollapse(EIssuesStoreType.PROFILE, "acme", undefined, undefined, onWrite);
  return React.createElement("button", { onClick: () => value.toggle("group_by", "state") }, "Collapse");
}

await test("project collapse toggles persist in the legacy local preference record", async (t) => {
  const fixture = await createFixture(t);
  window.localStorage.clear();
  let projectKanban;
  const onUpdate = (value) => {
    projectKanban = value;
  };
  await fixture.render(React.createElement(Probe, { workspaceSlug: "acme", projectId: "p1", onUpdate }));
  assert.deepEqual(projectKanban.filters, { group_by: [], sub_group_by: [] });

  await act(async () => projectKanban.update({ group_by: ["priority"] }));

  assert.deepEqual(projectKanban.filters, { group_by: ["priority"], sub_group_by: [] });
  assert.deepEqual(JSON.parse(window.localStorage.getItem("issue_local_filters")), [
    {
      key: EIssuesStoreType.PROJECT,
      workspaceSlug: "acme",
      viewId: "p1",
      userId: "viewer",
      filters: { kanban_filters: { group_by: ["priority"], sub_group_by: [] } },
    },
  ]);
});

await test("project collapse state reloads from the existing account-scoped local format", async (t) => {
  const fixture = await createFixture(t);
  window.localStorage.setItem("issue_local_filters", JSON.stringify(accountScopedCollapseRecords(["blocked"])));
  let projectKanban;
  const onUpdate = (value) => {
    projectKanban = value;
  };
  await fixture.render(React.createElement(Probe, { workspaceSlug: "acme", projectId: "p1", onUpdate }));

  assert.deepEqual(projectKanban.filters, { group_by: ["state"], sub_group_by: ["blocked"] });
});

await test("project collapse state follows project and account changes without overwriting other scopes", async (t) => {
  const fixture = await createFixture(t);
  window.localStorage.setItem("issue_local_filters", JSON.stringify(accountScopedCollapseRecords()));
  let projectKanban;
  const onUpdate = (value) => {
    projectKanban = value;
  };
  const renderProbe = (projectId) => React.createElement(Probe, { workspaceSlug: "acme", projectId, onUpdate });
  await fixture.render(renderProbe("p1"));
  assert.deepEqual(projectKanban.filters.group_by, ["state"]);

  await act(async () => projectKanban.update({ group_by: ["state", "labels"] }));
  await fixture.render(renderProbe("p2"));
  assert.deepEqual(projectKanban.filters.group_by, ["priority"]);
  await fixture.render(renderProbe("p1"));
  assert.deepEqual(projectKanban.filters.group_by, ["state", "labels"]);

  await act(async () => {
    fixture.client.setQueryData(identityKeys.currentUser(), { id: "other" });
  });
  await settleQuery();
  assert.deepEqual(projectKanban.filters.group_by, ["assignees"]);
  assert.deepEqual(
    JSON.parse(window.localStorage.getItem("issue_local_filters")).map((entry) => [
      entry.viewId,
      entry.userId,
      entry.filters.kanban_filters.group_by,
    ]),
    [
      ["p1", "viewer", ["state", "labels"]],
      ["p2", "viewer", ["priority"]],
      ["p1", "other", ["assignees"]],
    ]
  );
});

await test("non-project collapse write failures are surfaced to the user", async (t) => {
  const fixture = await createFixture(t);
  globalThis.projectKanbanWriteToasts = [];
  await fixture.render(React.createElement(NonProjectCollapseWriteFailureProbe));

  await act(async () => {
    fixture.environment.container
      .querySelector("button")
      .dispatchEvent(new window.MouseEvent("click", { bubbles: true }));
    await Promise.resolve();
  });

  assert.equal(globalThis.projectKanbanWriteToasts.length, 1);
  assert.equal(globalThis.projectKanbanWriteToasts[0].type, "error");
  assert.equal(globalThis.projectKanbanWriteToasts[0].message, "Collapse write failed");
});

await test("non-project collapse without a writer is visibly rejected", async (t) => {
  const fixture = await createFixture(t);
  globalThis.projectKanbanWriteToasts = [];
  await fixture.render(React.createElement(NonProjectCollapseMissingWriterProbe));

  await act(async () => {
    fixture.environment.container
      .querySelector("button")
      .dispatchEvent(new window.MouseEvent("click", { bubbles: true }));
  });

  assert.equal(globalThis.projectKanbanWriteToasts.length, 1);
  assert.equal(globalThis.projectKanbanWriteToasts[0].type, "error");
  assert.equal(
    globalThis.projectKanbanWriteToasts[0].message,
    "Issue group preferences are unavailable for this view."
  );
});

await test("non-project collapse writers keep their empty target for views without a project route", async (t) => {
  const fixture = await createFixture(t);
  globalThis.projectKanbanWriteToasts = [];
  let call;
  await fixture.render(
    React.createElement(NonProjectCollapseEmptyTargetProbe, {
      onWrite: async (...args) => {
        call = args;
      },
    })
  );

  await act(async () => {
    fixture.environment.container
      .querySelector("button")
      .dispatchEvent(new window.MouseEvent("click", { bubbles: true }));
  });

  assert.deepEqual(call, ["", "kanban_filters", { group_by: ["state"] }]);
  assert.equal(globalThis.projectKanbanWriteToasts.length, 0);
});
