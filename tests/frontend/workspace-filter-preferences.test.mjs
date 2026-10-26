import assert from "node:assert/strict";
import test from "node:test";
import { pathToFileURL } from "node:url";
import path from "node:path";
const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { registerWebModuleHooks } = await fromRoot("tests/frontend/helpers/web-module-hooks.mjs");
const { React, act, createPermissionUiFixture, registerPermissionUiStubs, settleQuery } = await fromRoot(
  "tests/frontend/helpers/permission-ui-fixture.mjs"
);

registerWebModuleHooks();
const { WorkspaceIssuePreferencesProvider, useWorkspaceIssuePreferences, areWorkspaceRichFiltersEquivalent } =
  await fromRoot("apps/web/core/providers/workspace-issue-preferences.tsx");
const { EIssuesStoreType } = await fromRoot("packages/types/dist/index.mjs");
const { workspaceViewKeys } = await fromRoot("apps/web/core/queries/global-views.ts");
const { permissionKeys } = await fromRoot("apps/web/core/queries/permissions.ts");
const { identityKeys, advanceIdentitySession, currentUserOptions } = await fromRoot(
  "apps/web/core/queries/identity.ts"
);
const { default: userService } = await fromRoot("apps/web/core/services/user.service.ts");
const { WorkspaceService } = await fromRoot("apps/web/core/services/workspace.service.ts");
const { WorkItemFilterUiProvider } = await fromRoot("apps/web/core/providers/work-item-filter-ui.tsx");
const { EIssueFilterType } = await fromRoot("packages/constants/dist/index.mjs");
const savedView = {
  id: "saved-view",
  name: "Saved",
  rich_filters: { priority__in: "low" },
  display_filters: { layout: "spreadsheet", order_by: "-created_at" },
  display_properties: { priority: false },
};

async function preferencesFixture(
  t,
  viewId = "saved-view",
  { local, legacy, editor = false, modal = false, saved = savedView } = {}
) {
  t.after(settleQuery);
  const fixture = await createPermissionUiFixture(t, { path: `/acme/workspace-views/${viewId}`, workspaceRole: 15 });
  if (local !== undefined) window.localStorage.setItem("workspace_issue_preferences", local);
  if (legacy !== undefined) window.localStorage.setItem("issue_local_filters", legacy);
  const membership = fixture.client.getQueryData(permissionKeys.workspaceMembership("alice", "acme"));
  const key = workspaceViewKeys.list({
    accountId: "alice",
    workspaceSlug: "acme",
    workspaceRole: 15,
    membershipRevision: membership.updated_at,
    sessionGeneration: 0,
  });
  fixture.client.setQueryData(key, [saved]);
  t.mock.method(WorkspaceService.prototype, "getAllViews", async () => [saved]);
  let current;
  let currentEditor;
  let modalCloses = 0;
  function FilterProbe({ editor: selectedEditor }) {
    React.useLayoutEffect(() => {
      currentEditor = selectedEditor;
    });
    return null;
  }
  function Probe() {
    const preferences = useWorkspaceIssuePreferences();
    React.useLayoutEffect(() => {
      current = preferences;
    });
    const editorNode =
      editor && preferences?.issueFilters
        ? React.createElement(
            WorkspaceLevelWorkItemFiltersHOC,
            {
              entityType: EIssuesStoreType.GLOBAL,
              entityId: preferences.viewId,
              workspaceSlug: preferences.workspaceSlug,
              enableSaveView: true,
              enableUpdateView: true,
              initialWorkItemFilters: preferences.issueFilters,
              updateFilters: (expression) => preferences.updateExpression(expression),
            },
            ({ editor: filterEditor }) => React.createElement(FilterProbe, { editor: filterEditor })
          )
        : React.createElement("span", null, preferences?.metadata.membershipQuery.status);
    return React.createElement(
      React.Fragment,
      null,
      editorNode,
      modal && preferences?.metadata.view
        ? React.createElement(CreateUpdateWorkspaceViewModal, {
            isOpen: true,
            data: preferences.metadata.view,
            onClose: () => {
              modalCloses++;
            },
          })
        : null
    );
  }
  await fixture.render(
    React.createElement(
      WorkItemFilterUiProvider,
      { scopeKey: `alice:${fixture.router.state.location.pathname}` },
      React.createElement(WorkspaceIssuePreferencesProvider, null, React.createElement(Probe))
    )
  );
  return {
    ...fixture,
    current: () => current,
    key,
    filter: () => currentEditor,
    closes: () => modalCloses,
  };
}

await test("workspace request separates read verification from mutation lineage", async (t) => {
  const f = await preferencesFixture(t);
  const request = f.current().requestInput;
  assert.equal(request.isScopeCurrent(), true);
  assert.equal(request.isReadScopeCurrent(), true);
  t.mock.method(userService, "currentUser", async () => {
    throw new Error("identity refresh failed");
  });
  await assert.rejects(f.client.fetchQuery({ ...currentUserOptions(), staleTime: 0 }));
  assert.equal(request.isScopeCurrent(), true);
  assert.equal(request.isReadScopeCurrent(), false);
});

await test("custom view preferences derive directly from the canonical native list without a detail GET", async (t) => {
  const fixture = await preferencesFixture(t);

  assert.deepEqual(fixture.current().issueFilters.richFilters, { priority__in: "low" });
  assert.equal(fixture.current().issueFilters.displayProperties.priority, false);
  assert.equal(fixture.current().requestInput.isScopeCurrent(), true);
});

await test("working edits preserve the canonical view and only request-affecting fields change query identity", async (t) => {
  const fixture = await preferencesFixture(t);
  const before = fixture.current().requestInput;
  await act(async () => {
    fixture.current().updateFilters(EIssueFilterType.DISPLAY_PROPERTIES, { priority: true });
  });
  assert.deepEqual(fixture.current().requestInput.displayFilters, before.displayFilters);
  assert.equal(fixture.client.getQueryData(fixture.key)[0].display_properties.priority, false);

  await act(async () => {
    fixture.current().updateExpression({ priority__in: "high" });
  });

  assert.notDeepEqual(fixture.current().requestInput.richFilters, before.richFilters);
  assert.deepEqual(fixture.client.getQueryData(fixture.key)[0].rich_filters, { priority__in: "low" });
  assert.deepEqual(fixture.current().issueFilters.richFilters, { priority__in: "high" });
});

await test("a canonical refresh cannot overwrite a newer working rich-filter draft", async (t) => {
  const fixture = await preferencesFixture(t);
  await act(async () => {
    fixture.current().updateExpression({ priority__in: "high" });
  });
  const submitted = fixture.current().revision;
  await act(async () => {
    fixture.current().updateExpression({ priority__in: "urgent" });
  });
  await act(async () => {
    fixture.client.setQueryData(fixture.key, [{ ...savedView, rich_filters: { priority__in: "none" } }]);
  });

  assert.equal(fixture.current().acknowledgeSave("saved-view", submitted), false);
  assert.deepEqual(fixture.current().issueFilters.richFilters, { priority__in: "urgent" });
});

await test("accepted save clears only the submitted draft while canonical data remains the default", async (t) => {
  const fixture = await preferencesFixture(t);
  await act(async () => {
    fixture.current().updateExpression({ priority__in: "high" });
  });
  const submitted = fixture.current().revision;
  await act(async () => {
    fixture.client.setQueryData(fixture.key, [{ ...savedView, rich_filters: { priority__in: "high" } }]);
    assert.equal(fixture.current().acknowledgeSave("saved-view", submitted), true);
  });

  assert.equal(fixture.current().isExpressionDirty, false);
  assert.deepEqual(fixture.current().issueFilters.richFilters, { priority__in: "high" });
});

async function failMembershipRefresh(t, fixture) {
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => {
    throw new Error("offline");
  });
  await act(async () => {
    await fixture.client.refetchQueries({ queryKey: permissionKeys.workspaceMembership("alice", "acme"), exact: true });
  });
  await settleQuery();
}

await test("a failed membership refresh invalidates a captured request despite cached permission data", async (t) => {
  const fixture = await preferencesFixture(t);
  const input = fixture.current().requestInput;
  await failMembershipRefresh(t, fixture);

  assert.equal(input.isScopeCurrent(), false);
  assert.equal(fixture.current().issueFilters, undefined);
});

await test("a same-account session replacement clears working drafts and invalidates captured request input", async (t) => {
  const fixture = await preferencesFixture(t);
  await act(async () => {
    fixture.current().updateExpression({ priority__in: "high" });
  });
  const input = fixture.current().requestInput;

  await act(async () => {
    advanceIdentitySession(fixture.client);
  });

  assert.equal(input.isScopeCurrent(), false);
  assert.equal(fixture.current().isExpressionDirty, false);
});

await test("static display choices persist within the account while rich filters remain transient", async (t) => {
  const fixture = await preferencesFixture(t, "assigned", {
    local: JSON.stringify([
      {
        accountId: "alice",
        workspaceSlug: "acme",
        viewId: "assigned",
        filters: { display_filters: { layout: "spreadsheet", order_by: "priority" } },
      },
    ]),
  });
  await act(async () => {
    fixture.current().updateFilters(EIssueFilterType.DISPLAY_FILTERS, { order_by: "-priority" });
    fixture.current().updateExpression({ priority__in: "high" });
  });
  const records = JSON.parse(window.localStorage.getItem("workspace_issue_preferences"));

  assert.equal(fixture.current().issueFilters.displayFilters.order_by, "-priority");
  assert.equal(records[0].filters.rich_filters, undefined);
  assert.equal(records[0].filters.display_filters.order_by, "-priority");
});

await test("invalid local storage does not prevent static view defaults", async (t) => {
  const fixture = await preferencesFixture(t, "all-issues", { local: "[null]" });

  assert.equal(fixture.current().issueFilters.displayFilters.layout, "spreadsheet");
  assert.equal(fixture.current().issueFilters.displayFilters.order_by, "-created_at");
});

await test("batched display changes compose without mutating the caller payload or canonical response", async (t) => {
  const fixture = await preferencesFixture(t);
  const patch = Object.freeze({ order_by: "priority" });
  await act(async () => {
    fixture.current().updateFilters(EIssueFilterType.DISPLAY_FILTERS, patch);
    fixture.current().updateFilters(EIssueFilterType.DISPLAY_FILTERS, { sub_issue: true });
  });

  assert.equal(fixture.current().issueFilters.displayFilters.order_by, "priority");
  assert.equal(fixture.current().issueFilters.displayFilters.sub_issue, true);
  assert.deepEqual(patch, { order_by: "priority" });
  assert.equal(fixture.client.getQueryData(fixture.key)[0].display_filters.order_by, "-created_at");
});

await test("route A to B to A resets working drafts and never revives an old request guard", async (t) => {
  const fixture = await preferencesFixture(t);
  await act(async () => {
    fixture.current().updateExpression({ priority__in: "high" });
  });
  const captured = fixture.current().requestInput;
  await act(async () => {
    await fixture.router.navigate({ href: "/acme/workspace-views/assigned" });
  });
  await act(async () => {
    await fixture.router.navigate({ href: "/acme/workspace-views/saved-view" });
  });
  await settleQuery();

  assert.equal(captured.isScopeCurrent(), false);
  assert.equal(fixture.current().isExpressionDirty, false);
  assert.deepEqual(fixture.current().issueFilters.richFilters, { priority__in: "low" });
});

await test("canonical view-list replacement invalidates a captured custom-view request", async (t) => {
  const fixture = await preferencesFixture(t);
  const captured = fixture.current().requestInput;
  await act(async () => {
    fixture.client.removeQueries({ queryKey: fixture.key, exact: true });
    fixture.client.setQueryData(fixture.key, [savedView]);
  });

  assert.equal(captured.isScopeCurrent(), false);
});

await test("rich-only metadata acknowledgement preserves independent display changes", async (t) => {
  const fixture = await preferencesFixture(t);
  await act(async () => {
    fixture.current().updateFilters(EIssueFilterType.DISPLAY_PROPERTIES, { priority: true });
    fixture.current().updateExpression({ priority__in: "high" });
  });
  const revision = fixture.current().revision;
  await act(async () => {
    fixture.client.setQueryData(fixture.key, [{ ...savedView, rich_filters: { priority__in: "high" } }]);
    assert.equal(fixture.current().acknowledgeSave("saved-view", revision, true), true);
  });

  assert.equal(fixture.current().isExpressionDirty, false);
  assert.equal(fixture.current().issueFilters.displayProperties.priority, true);
});

await test("persisted display choices stay isolated across account A to B to A", async (t) => {
  const fixture = await preferencesFixture(t, "assigned");
  await act(async () => {
    fixture.current().updateFilters(EIssueFilterType.DISPLAY_FILTERS, { order_by: "priority" });
  });
  const membership = fixture.client.getQueryData(permissionKeys.workspaceMembership("alice", "acme"));
  fixture.client.setQueryData(permissionKeys.workspaceMembership("bob", "acme"), membership);
  await act(async () => {
    fixture.client.setQueryData(identityKeys.currentUser(), { id: "bob" });
  });
  await settleQuery();
  assert.equal(fixture.current().issueFilters.displayFilters.order_by, "-created_at");
  await act(async () => {
    fixture.current().updateFilters(EIssueFilterType.DISPLAY_FILTERS, { order_by: "-priority" });
  });

  await act(async () => {
    fixture.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  });
  await settleQuery();

  assert.equal(fixture.current().issueFilters.displayFilters.order_by, "priority");
  const records = JSON.parse(window.localStorage.getItem("workspace_issue_preferences"));
  assert.equal(records.find((record) => record.accountId === "alice").filters.display_filters.order_by, "priority");
  assert.equal(records.find((record) => record.accountId === "bob").filters.display_filters.order_by, "-priority");
});

await test("unowned legacy preferences are not inferred into an account or rewritten", async (t) => {
  const legacy = JSON.stringify([
    {
      key: "GLOBAL",
      workspaceSlug: "acme",
      userId: "assigned",
      filters: { display_filters: { layout: "spreadsheet", order_by: "priority" } },
    },
    {
      key: "PROJECT",
      workspaceSlug: "acme",
      viewId: "project-a",
      filters: { display_filters: { order_by: "-priority" } },
    },
  ]);
  const fixture = await preferencesFixture(t, "assigned", { legacy });

  assert.equal(fixture.current().issueFilters.displayFilters.order_by, "-created_at");
  await act(async () => {
    fixture.current().updateFilters(EIssueFilterType.DISPLAY_FILTERS, { order_by: "name" });
  });

  assert.equal(window.localStorage.getItem("issue_local_filters"), legacy);
  assert.equal(JSON.parse(window.localStorage.getItem("workspace_issue_preferences"))[0].accountId, "alice");
});

// Keep the actual HOC and React expression editor. Only catalogue/config I/O is isolated.
registerPermissionUiStubs("/work-item-filters/filters-hoc/workspace-level.tsx", {
  "@/components/issues/issue-layouts/utils": "export const removeNillKeys = (value) => value ?? {};",
  "@/hooks/store/use-label": "export const useLabel = () => ({getWorkspaceLabelIds: () => []});",
  "@/hooks/use-workspaces": "export const useCurrentWorkspace = () => ({data: {id: 'workspace'}});",
  "@/hooks/use-joined-project-ids": "export const useJoinedProjectIds = () => [];",
  "@/hooks/use-member-rosters": "export const useWorkspaceMemberRoster = () => ({members: []});",
  "@/components/workspace/views/modal": "export const CreateUpdateWorkspaceViewModal = () => null;",
  "@plane/propel/toast":
    "export const TOAST_TYPE = {SUCCESS:'success',ERROR:'error'}; export const setToast = () => {};",
});
registerPermissionUiStubs("/work-item-filters/filters-hoc/use-view-filter-actions.ts", {
  "@/components/issues/issue-layouts/utils":
    "export const removeNillKeys = value => Object.fromEntries(Object.entries(value ?? {}).filter(([,entry]) => entry != null));",
});
registerPermissionUiStubs("/work-item-filters/filters-hoc/base.tsx", {
  "@/hooks/work-item-filters/use-work-item-filters-config":
    "const configs = []; export const useWorkItemFiltersConfig = () => ({configs, areAllConfigsInitialized: true});",
});
const editorUi = {};
globalThis.workspaceFilterEditorUi = editorUi;
registerPermissionUiStubs("/workspace/views/modal.tsx", {
  "@/components/views/form":
    "export function ViewForm(props) { globalThis.workspaceFilterEditorUi.form = props; return null; }",
  "@plane/propel/toast":
    "export const TOAST_TYPE = {SUCCESS:'success',ERROR:'error'}; export const setToast = () => {};",
});
const { CreateUpdateWorkspaceViewModal } = await fromRoot("apps/web/core/components/workspace/views/modal.tsx");
const { WorkspaceLevelWorkItemFiltersHOC } = await fromRoot(
  "apps/web/core/components/work-item-filters/filters-hoc/workspace-level.tsx"
);

await test("filter-row saves await persistence and retain retry state after rejection", async (t) => {
  const fixture = await preferencesFixture(t, "saved-view", {
    editor: true,
    saved: { ...savedView, owned_by: "alice" },
  });
  await settleQuery();
  await act(async () => {
    fixture.filter().onUpdateConditionValue(fixture.filter().conditions[0].id, "high");
  });
  assert.equal(fixture.filter().canUpdate, true);
  const deferred = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "updateView", () => deferred.promise);
  let rejected;
  await act(async () => {
    rejected = assert.rejects(fixture.filter().onUpdate(), /offline/);
  });
  assert.equal(fixture.filter().hasChanges, true);
  await act(async () => {
    deferred.reject(new Error("offline"));
    await rejected;
  });
  assert.equal(fixture.filter().hasChanges, true);
  assert.equal(fixture.filter().canUpdate, true);
  assert.deepEqual(fixture.client.getQueryData(fixture.key)[0].rich_filters, savedView.rich_filters);
});

await test("real editor canonical initialization and refresh remain pristine despite reset notifications", async (t) => {
  const fixture = await preferencesFixture(t, "saved-view", { editor: true });
  await settleQuery();
  assert.ok(fixture.filter());
  assert.equal(fixture.current().isExpressionDirty, false);
  assert.equal(fixture.current().revision, 0);

  await act(async () => {
    fixture.client.setQueryData(fixture.key, [{ ...savedView, rich_filters: { and: [{ priority__in: "high" }] } }]);
  });
  await settleQuery();

  assert.equal(fixture.current().isExpressionDirty, false);
  assert.equal(fixture.current().revision, 0);
  assert.equal(fixture.filter().conditions[0].value, "high");
});

await test("real static editor is replaced across workspaces and old callbacks cannot edit the new draft", async (t) => {
  const fixture = await preferencesFixture(t, "assigned", { editor: true });
  await settleQuery();
  let oldFilter;
  await act(async () => {
    await fixture.current().updateExpression({ priority__in: "high" });
  });
  await settleQuery();
  oldFilter = fixture.filter();
  assert.equal(oldFilter.conditions.length, 1, "the edit callback is captured after the editor commits its render");
  await act(async () => {
    oldFilter.onToggleVisibility(true);
  });
  assert.equal(fixture.current().isExpressionDirty, true);
  const membership = fixture.client.getQueryData(permissionKeys.workspaceMembership("alice", "acme"));
  fixture.client.setQueryData(permissionKeys.workspaceMembership("alice", "beta"), membership);

  await act(async () => {
    await fixture.router.navigate({ href: "/beta/workspace-views/assigned" });
  });
  await settleQuery();

  assert.notEqual(fixture.filter(), oldFilter);
  assert.equal(fixture.filter().conditions.length, 0);
  assert.equal(fixture.current().workspaceSlug, "beta");
  assert.equal(fixture.current().isExpressionDirty, false);
  await act(async () => {
    oldFilter.onUpdateConditionValue(oldFilter.conditions[0].id, "urgent");
  });
  assert.equal(fixture.current().isExpressionDirty, false);
  assert.equal(fixture.filter().conditions.length, 0);
});

async function realEditorWithDraft(t) {
  const fixture = await preferencesFixture(t, "saved-view", { editor: true, modal: true });
  await settleQuery();
  await act(async () => {
    await fixture.current().updateExpression({ priority__in: "high" });
  });
  let canonical = savedView;
  t.mock.method(WorkspaceService.prototype, "getAllViews", async () => [canonical]);
  t.mock.method(WorkspaceService.prototype, "updateView", async (_workspace, _view, payload) => {
    canonical = { ...canonical, ...payload };
    return canonical;
  });
  return fixture;
}

await test("name-only quick-action editing preserves the unrelated active real-editor draft", async (t) => {
  const fixture = await realEditorWithDraft(t);

  await act(async () => {
    await editorUi.form.handleFormSubmit({ ...savedView, name: "Renamed" });
  });
  await settleQuery();

  assert.equal(fixture.closes(), 1);
  assert.equal(fixture.client.getQueryData(fixture.key)[0].name, "Renamed");
  assert.deepEqual(fixture.current().issueFilters.richFilters, { priority__in: "high" });
  assert.equal(fixture.current().isExpressionDirty, true);
  assert.equal(fixture.filter().conditions[0].value, "high");
});

await test("editing that submits and accepts the current real-editor expression acknowledges its draft", async (t) => {
  const fixture = await realEditorWithDraft(t);

  await act(async () => {
    await editorUi.form.handleFormSubmit({ ...savedView, rich_filters: { priority__in: "high" } });
  });
  await settleQuery();

  assert.equal(fixture.closes(), 1);
  assert.equal(fixture.current().isExpressionDirty, false);
  assert.equal(fixture.filter().hasChanges, false);
  assert.equal(fixture.filter().conditions[0].value, "high");
});

await test("saved predicate comparison retains invalid, null and empty selections", () => {
  assert.equal(areWorkspaceRichFiltersEquivalent({ unknown__exact: "a" }, {}), false);
  assert.equal(areWorkspaceRichFiltersEquivalent({ unknown__exact: "a" }, { unknown__exact: "b" }), false);
  assert.equal(areWorkspaceRichFiltersEquivalent({ state_id__exact: null }, {}), false);
  assert.equal(areWorkspaceRichFiltersEquivalent({ label_id__in: [] }, {}), false);
  assert.equal(
    areWorkspaceRichFiltersEquivalent({ and: [{ state_id__exact: null }] }, { state_id__exact: null }),
    true
  );
});
