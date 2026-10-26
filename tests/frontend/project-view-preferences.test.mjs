import assert from "node:assert/strict";
import test from "node:test";
import { createPermissionUiFixture, React, act } from "./helpers/permission-ui-fixture.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { EIssuesStoreType } from "../../packages/types/dist/index.mjs";
import { EIssueFilterType } from "../../packages/constants/dist/index.mjs";

registerWebModuleHooks();

const { useIssueFilterPreferences } = await import("../../apps/web/core/hooks/use-issue-filter-preferences.ts");
const { useIssueKanbanCollapse } = await import("../../apps/web/core/hooks/use-issue-kanban-collapse.ts");
const { useCalendarDisplayFilterUpdate } =
  await import("../../apps/web/core/hooks/use-calendar-display-filter-update.ts");
const { useProjectView } = await import("../../apps/web/core/hooks/use-project-views.ts");
const { useIssueListMutationOwner } = await import("../../apps/web/core/hooks/use-issue-list-mutation-owner.ts");
const { ProjectViewIssuePreferencesProvider, useProjectViewIssuePreferences } =
  await import("../../apps/web/core/providers/project-view-issue-preferences.tsx");
const { ViewService } = await import("../../apps/web/core/services/view.service.ts");
const userService = (await import("../../apps/web/core/services/user.service.ts")).default;

const initialView = {
  id: "view-a",
  project: "p1",
  name: "View",
  owned_by: "alice",
  is_locked: false,
  updated_at: "revision-1",
  rich_filters: { priority__exact: "low" },
  display_filters: { layout: "list" },
  display_properties: {},
};

function createHarness(onPreferences, ProbeComponent = Probe) {
  return function Harness() {
    const query = useProjectView("acme", "p1", "view-a");
    if (!query.view) return null;
    return React.createElement(
      ProjectViewIssuePreferencesProvider,
      {
        view: query.view,
        workspaceSlug: "acme",
        projectId: "p1",
        accountId: "alice",
        isScopeCurrent: query.isScopeCurrent,
        isReadScopeCurrent: query.isReadScopeCurrent,
      },
      React.createElement(ProbeComponent, { onPreferences })
    );
  };
}

function LayoutPreferencesProbe({ onPreferences }) {
  const preferences = useProjectViewIssuePreferences();
  const selected = useIssueFilterPreferences(EIssuesStoreType.PROJECT_VIEW);
  const update = (projectId, type, values) => preferences.updateFilters("acme", projectId, type, values);
  const collapse = useIssueKanbanCollapse(
    EIssuesStoreType.PROJECT_VIEW,
    "acme",
    "p1",
    selected.issueFilters?.kanbanFilters,
    update
  );
  const updateCalendar = useCalendarDisplayFilterUpdate(
    EIssuesStoreType.PROJECT_VIEW,
    undefined,
    selected.issueFilters?.displayFilters?.calendar,
    update
  );
  React.useLayoutEffect(() => {
    onPreferences({ preferences, selected, collapse, updateCalendar });
  });
  return null;
}

function Probe({ onPreferences }) {
  onPreferences(useProjectViewIssuePreferences());
  return null;
}

function MutationProbe({ onPreferences }) {
  const preferences = useProjectViewIssuePreferences();
  const resolveMutationOwner = useIssueListMutationOwner(EIssuesStoreType.PROJECT_VIEW);
  onPreferences({ ...preferences, resolveMutationOwner });
  return null;
}

function ViewQueryProbe({ onQuery }) {
  const query = useProjectView("acme", "p1", "view-a");
  React.useLayoutEffect(() => {
    onQuery(query);
  }, [onQuery, query]);
  return null;
}

async function fixtureFor(t, route = "/acme/projects/p1/views/view-a") {
  const fixture = await createPermissionUiFixture(t, {
    path: route,
    workspaceRole: 15,
    projectRoles: { p1: 15 },
  });
  t.mock.method(ViewService.prototype, "getViewDetails", async () => initialView);
  return fixture;
}

await test("rapid filter edits publish the latest immutable request without waiting for list reads", async (t) => {
  const fixture = await fixtureFor(t);
  let current;
  await fixture.render(React.createElement(createHarness((value) => (current = value))));
  assert.ok(current);

  await React.act(async () => {
    const first = current.updateRichFilters({ priority__exact: "high" });
    const second = current.updateRichFilters({ priority__exact: "urgent" });
    await Promise.all([first, second]);
  });

  assert.deepEqual(current.requestInput.richFilters, { priority__exact: "urgent" });
  assert.equal(current.requestInput.isScopeCurrent(), true);
});

await test("metadata-only view updates retain a dirty local filter draft", async (t) => {
  const fixture = await fixtureFor(t);
  let current;
  const Harness = createHarness((value) => (current = value));
  await fixture.render(React.createElement(Harness));

  await React.act(async () => {
    await current.updateRichFilters({ priority__exact: "high" });
  });
  assert.deepEqual(current.requestInput.richFilters, { priority__exact: "high" });

  const savedMetadata = { ...initialView, name: "Renamed", updated_at: "revision-2" };
  const key = fixture.client
    .getQueryCache()
    .findAll({ queryKey: ["project-views"] })
    .find((query) => query.queryKey.at(-2) === "detail")?.queryKey;
  assert.ok(key);
  fixture.client.setQueryData(key, savedMetadata);
  await fixture.render(React.createElement(Harness));

  assert.deepEqual(current.requestInput.richFilters, { priority__exact: "high" });
  assert.equal(current.requestInput.isScopeCurrent(), true);
});

await test("route changes and unmount invalidate captured reads and mutation projections", async (t) => {
  const fixture = await fixtureFor(t);
  let current;
  const Harness = createHarness((value) => (current = value), MutationProbe);
  await fixture.render(React.createElement(Harness));
  const oldRead = current.requestInput;
  const oldMutation = await current.resolveMutationOwner("acme", "p1");
  assert.equal(oldRead.isReadScopeCurrent(), true);
  assert.ok(oldMutation);
  assert.equal(oldMutation.isScopeCurrent(), true);

  await act(async () => {
    fixture.router.history.push("/acme/projects/p2/views/view-a");
    await fixture.router.load();
  });
  assert.equal(oldRead.isScopeCurrent(), false);
  assert.equal(oldMutation.isScopeCurrent(), false);

  await fixture.render(null);
  assert.equal(oldRead.isScopeCurrent(), false);
  assert.equal(oldRead.isReadScopeCurrent(), false);
  assert.equal(oldMutation.isScopeCurrent(), false);
});

await test("a resolved missing project role is denied instead of pending forever", async (t) => {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/views/view-a",
    workspaceRole: 15,
    projectRoles: {},
  });
  const detail = t.mock.method(ViewService.prototype, "getViewDetails", async () => initialView);
  let query;
  await fixture.render(
    React.createElement(ViewQueryProbe, {
      onQuery: (value) => {
        query = value;
      },
    })
  );
  assert.equal(query.isDenied, true);
  assert.equal(query.isPending, false);
  assert.equal(detail.mock.callCount(), 0);
});

await test("retry stops after the committed route changes during identity refresh", async (t) => {
  const fixture = await fixtureFor(t);
  let query;
  await fixture.render(
    React.createElement(ViewQueryProbe, {
      onQuery: (value) => {
        query = value;
      },
    })
  );
  const detailReads = t.mock.method(ViewService.prototype, "getViewDetails", async () => initialView);
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  t.mock.method(userService, "currentUser", () => {
    started.resolve();
    return response.promise;
  });
  const retry = query.retry();
  await started.promise;
  await act(async () => {
    fixture.router.history.push("/acme/projects/p2/views/view-a");
    await fixture.router.load();
  });
  response.resolve({ id: "alice" });
  await retry;
  assert.equal(detailReads.mock.callCount(), 0);
  assert.equal(
    fixture.client
      .getQueryCache()
      .findAll({ queryKey: ["project-views"] })
      .some((entry) => entry.queryKey[5] === "p2"),
    false
  );
});

await test("native view layout settings preserve calendar fields, collapsed siblings, and identifier visibility", async (t) => {
  const fixture = await fixtureFor(t);
  let current;
  await fixture.render(React.createElement(createHarness((value) => (current = value), LayoutPreferencesProbe)));
  await act(async () => {
    await current.preferences.updateFilters("acme", "p1", EIssueFilterType.DISPLAY_FILTERS, {
      calendar: { layout: "month", show_weekends: true },
    });
    await current.preferences.updateFilters("acme", "p1", EIssueFilterType.DISPLAY_PROPERTIES, { key: false });
    await current.preferences.updateFilters("acme", "p1", EIssueFilterType.KANBAN_FILTERS, { group_by: ["first"] });
  });
  assert.equal(current.selected.issueFilters.displayProperties.key, false);
  assert.deepEqual(current.collapse.filters.group_by, ["first"]);
  await act(async () => {
    current.collapse.toggle("group_by", "second");
    await current.updateCalendar("p1", { calendar: { layout: "week" } });
  });
  assert.deepEqual(current.collapse.filters.group_by, ["first", "second"]);
  assert.deepEqual(current.selected.issueFilters.displayFilters.calendar, { layout: "week", show_weekends: true });
  await act(async () => current.collapse.toggle("group_by", "first"));
  assert.deepEqual(current.collapse.filters.group_by, ["second"]);
});
