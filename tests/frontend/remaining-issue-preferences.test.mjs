import assert from "node:assert/strict";
import test from "node:test";
import { React, act, createPermissionUiFixture } from "./helpers/permission-ui-fixture.mjs";

const { LocalIssuePreferencesProvider, useLocalIssuePreferences } =
  await import("../../apps/web/core/providers/local-issue-preferences.tsx");
const { getProfileIssueRequestParams, getArchivedIssueRequestParams } =
  await import("../../apps/web/helpers/issue-list/request.ts");
const { EIssuesStoreType } = await import("../../packages/types/dist/index.mjs");
const { EIssueFilterType } = await import("../../packages/constants/dist/index.mjs");
const { readLocalIssueFilters } = await import("../../apps/web/helpers/issue-list/local-preferences.ts");
const { advanceIdentitySession, identityKeys } = await import("../../apps/web/core/queries/identity.ts");

function Probe({ onRead }) {
  const preferences = useLocalIssuePreferences();
  React.useLayoutEffect(() => onRead(preferences), [onRead, preferences]);
  return null;
}

function profilePreferencesFor(userId, filters) {
  return { key: EIssuesStoreType.PROFILE, workspaceSlug: "acme", viewId: "alice", userId, filters };
}

await test("profile preferences persist locally and preserve the static profile view filter", async (t) => {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/profile/alice/assigned",
    workspaceRole: 15,
  });
  window.localStorage.setItem(
    "issue_local_filters",
    JSON.stringify([
      profilePreferencesFor("alice", {
        rich_filters: { priority__in: "high" },
        display_filters: { layout: "list" },
        display_properties: { priority: false },
      }),
    ])
  );
  let current;
  const onRead = (value) => {
    current = value;
  };
  await fixture.render(
    React.createElement(
      LocalIssuePreferencesProvider,
      { domain: EIssuesStoreType.PROFILE, workspaceSlug: "acme", entityId: "alice" },
      React.createElement(Probe, { onRead })
    )
  );

  assert.deepEqual(current.issueFilters.richFilters, { priority__in: "high" });
  assert.equal(current.issueFilters.displayProperties.priority, false);
  const params = getProfileIssueRequestParams(
    current.profileRequestInput("assigned"),
    { canGroup: true, perPageCount: 50 },
    undefined
  );
  assert.equal(params.assignees, "alice");
  assert.deepEqual(JSON.parse(params.filters), { priority__in: "high" });
  assert.equal(
    getProfileIssueRequestParams(
      current.profileRequestInput("created"),
      { canGroup: true, perPageCount: 50 },
      undefined
    ).created_by,
    "alice"
  );
  assert.equal(
    getProfileIssueRequestParams(
      current.profileRequestInput("subscribed"),
      { canGroup: true, perPageCount: 50 },
      undefined
    ).subscriber,
    "alice"
  );

  await act(async () => {
    await current.updateFilters(EIssueFilterType.DISPLAY_FILTERS, { layout: "kanban" });
  });
  assert.equal(
    readLocalIssueFilters(EIssuesStoreType.PROFILE, "acme", "alice", "alice").display_filters.layout,
    "kanban"
  );
  const staleInput = current.profileRequestInput("assigned");
  await act(async () => fixture.router.navigate({ href: "/acme/projects/p1/issues/" }));
  await fixture.render(React.createElement("main", null, "another route"));
  assert.equal(staleInput.isCurrent(), false);
});

await test("archived preferences keep the archived sub-issue request invariant", async (t) => {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/archives/issues/",
    workspaceRole: 15,
  });
  window.localStorage.setItem(
    "issue_local_filters",
    JSON.stringify([
      {
        key: EIssuesStoreType.ARCHIVED,
        workspaceSlug: "acme",
        viewId: "p1",
        userId: "alice",
        filters: { rich_filters: { priority__in: "urgent" }, display_filters: { layout: "list" } },
      },
    ])
  );
  let current;
  const onRead = (value) => {
    current = value;
  };
  await fixture.render(
    React.createElement(
      LocalIssuePreferencesProvider,
      { domain: EIssuesStoreType.ARCHIVED, workspaceSlug: "acme", entityId: "p1" },
      React.createElement(Probe, { onRead })
    )
  );

  const params = getArchivedIssueRequestParams(
    current.archivedRequestInput(),
    { canGroup: true, perPageCount: 50 },
    undefined
  );
  assert.equal(params.sub_issue, true);
  assert.deepEqual(JSON.parse(params.filters), { priority__in: "urgent" });
});

await test("profile local preferences remain bound to the authenticated account", async (t) => {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/profile/alice/assigned",
    workspaceRole: 15,
  });
  window.localStorage.setItem(
    "issue_local_filters",
    JSON.stringify([
      profilePreferencesFor("alice", { rich_filters: { priority__in: "high" } }),
      {
        key: EIssuesStoreType.PROFILE,
        workspaceSlug: "acme",
        viewId: "alice",
        filters: { rich_filters: { priority__in: "legacy-unowned" } },
      },
    ])
  );
  let current;
  const onRead = (value) => {
    current = value;
  };
  const provider = () =>
    React.createElement(
      LocalIssuePreferencesProvider,
      { domain: EIssuesStoreType.PROFILE, workspaceSlug: "acme", entityId: "alice" },
      React.createElement(Probe, { onRead })
    );
  await fixture.render(provider());

  assert.deepEqual(current.issueFilters.richFilters, { priority__in: "high" });
  const oldInput = current.profileRequestInput("assigned");
  await act(async () => {
    advanceIdentitySession(fixture.client);
    fixture.client.setQueryData(identityKeys.currentUser(), { id: "bob" });
  });
  await fixture.render(provider());

  assert.equal(current.accountId, "bob");
  assert.deepEqual(current.issueFilters.richFilters, {});
  assert.equal(oldInput.isCurrent(), false);
  await act(async () => {
    await current.updateRichFilters({ priority__in: "low" });
  });
  assert.deepEqual(readLocalIssueFilters(EIssuesStoreType.PROFILE, "acme", "alice", "bob").rich_filters, {
    priority__in: "low",
  });
  assert.deepEqual(readLocalIssueFilters(EIssuesStoreType.PROFILE, "acme", "alice", "alice").rich_filters, {
    priority__in: "high",
  });
  assert.deepEqual(readLocalIssueFilters(EIssuesStoreType.PROFILE, "acme", "alice", undefined).rich_filters, {
    priority__in: "legacy-unowned",
  });
});
