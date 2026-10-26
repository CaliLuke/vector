import assert from "node:assert/strict";
import test from "node:test";
import { pathToFileURL } from "node:url";
import path from "node:path";
import { MutationObserver, QueryClient, QueryObserver } from "@tanstack/react-query";
const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { registerWebModuleHooks } = await fromRoot("tests/frontend/helpers/web-module-hooks.mjs");

registerWebModuleHooks();

const { identityKeys, advanceIdentitySession } = await fromRoot("apps/web/core/queries/identity.ts");
const { permissionKeys } = await fromRoot("apps/web/core/queries/permissions.ts");
const { workspaceViewKeys, workspaceViewsOptions, workspaceViewMutationOptions } = await fromRoot(
  "apps/web/core/queries/global-views.ts"
);
const { WorkspaceService } = await fromRoot("apps/web/core/services/workspace.service.ts");

const scope = {
  accountId: "alice",
  workspaceSlug: "acme",
  workspaceRole: 15,
  membershipRevision: "membership-1",
  sessionGeneration: 0,
};

function createViewsQueryFixture(t, initialViews = [{ id: "view-a", name: "A" }]) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(() => client.clear());
  const membershipKey = permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug);
  const viewKey = workspaceViewKeys.list(scope);
  client.setQueryData(identityKeys.currentUser(), { id: scope.accountId });
  client.setQueryData(membershipKey, {
    is_active: true,
    role: scope.workspaceRole,
    updated_at: scope.membershipRevision,
  });
  client.setQueryData(viewKey, initialViews);
  return { client, membershipKey, viewKey };
}

await test("workspace view list is scoped to membership revision and forwards cancellation", async (t) => {
  const { client } = createViewsQueryFixture(t);
  let requestSignal;
  t.mock.method(WorkspaceService.prototype, "getAllViews", async (_slug, signal) => {
    requestSignal = signal;
    return [{ id: "view-a", name: "A" }];
  });

  const options = workspaceViewsOptions(client, scope);
  const views = await client.fetchQuery({ ...options, staleTime: 0 });

  assert.deepEqual(options.queryKey, ["workspace-views", "alice", "acme", 15, "membership-1", 0, "list"]);
  assert.deepEqual(views, [{ id: "view-a", name: "A" }]);
  assert.ok(requestSignal instanceof AbortSignal);
});

async function verifyLateAcceptedUpdate(t, transition) {
  const { client, viewKey } = createViewsQueryFixture(t);
  const requestStarted = Promise.withResolvers();
  const acceptWrite = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "updateView", () => {
    requestStarted.resolve();
    return acceptWrite.promise;
  });
  const write = new MutationObserver(client, workspaceViewMutationOptions.update(client, scope, 0));
  const pending = write.mutate({ viewId: "view-a", data: { name: "Saved" } });
  await requestStarted.promise;

  const replacement = [{ id: "view-a", name: "Replacement" }];
  if (transition === "query replacement") {
    client.removeQueries({ queryKey: viewKey, exact: true });
    client.setQueryData(viewKey, replacement);
  } else {
    advanceIdentitySession(client);
    client.setQueryData(viewKey, replacement);
  }
  acceptWrite.resolve({ id: "view-a", name: "Saved" });
  await pending;

  assert.deepEqual(client.getQueryData(viewKey), replacement, transition);
}

await test("a late accepted update cannot write into a replacement list", (t) =>
  verifyLateAcceptedUpdate(t, "query replacement"));

await test("a late accepted update cannot write into a restarted session", (t) =>
  verifyLateAcceptedUpdate(t, "session replacement"));

await test("a pending old roster GET cannot overwrite an accepted canonical update", async (t) => {
  const { client, viewKey } = createViewsQueryFixture(t);
  const staleRead = Promise.withResolvers();
  const requestStarted = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "getAllViews", () => {
    requestStarted.resolve();
    return staleRead.promise;
  });
  const options = workspaceViewsOptions(client, scope);
  const oldFetch = client.fetchQuery({ ...options, staleTime: 0 }).catch(() => undefined);
  await requestStarted.promise;

  t.mock.method(WorkspaceService.prototype, "updateView", async () => ({ id: "view-a", name: "Accepted" }));
  await new MutationObserver(client, workspaceViewMutationOptions.update(client, scope, 0)).mutate({
    viewId: "view-a",
    data: { name: "Accepted" },
  });
  staleRead.resolve([{ id: "view-a", name: "Stale" }]);
  await oldFetch;

  assert.deepEqual(client.getQueryData(viewKey), [{ id: "view-a", name: "Accepted" }]);
});

await test("rejected writes preserve cached rows and a retry sends the write again", async (t) => {
  const { client, viewKey } = createViewsQueryFixture(t);
  let rejectWrite = true;
  const update = t.mock.method(WorkspaceService.prototype, "updateView", async () => {
    if (rejectWrite) throw new Error("write rejected");
    return { id: "view-a", name: "Accepted" };
  });
  const mutation = new MutationObserver(client, workspaceViewMutationOptions.update(client, scope, 0));

  await assert.rejects(mutation.mutate({ viewId: "view-a", data: { name: "Accepted" } }), /write rejected/);
  assert.deepEqual(client.getQueryData(viewKey), [{ id: "view-a", name: "A" }]);
  rejectWrite = false;
  await mutation.mutate({ viewId: "view-a", data: { name: "Accepted" } });

  assert.equal(update.mock.callCount(), 2);
  assert.deepEqual(client.getQueryData(viewKey), [{ id: "view-a", name: "Accepted" }]);
});

await test("a failed refresh does not turn an accepted create into a repeatable write", async (t) => {
  const { client, viewKey } = createViewsQueryFixture(t);
  t.mock.method(WorkspaceService.prototype, "getAllViews", async () => {
    throw new Error("refresh offline");
  });
  const observer = new QueryObserver(client, workspaceViewsOptions(client, scope));
  const unsubscribe = observer.subscribe(() => {});
  t.after(unsubscribe);
  const create = t.mock.method(WorkspaceService.prototype, "createView", async () => ({ id: "view-b", name: "B" }));

  await new MutationObserver(client, workspaceViewMutationOptions.create(client, scope, 0)).mutate({ name: "B" });
  await new Promise((resolve) => setImmediate(resolve));

  assert.equal(create.mock.callCount(), 1);
  assert.equal(
    client.getQueryData(viewKey).some((view) => view.id === "view-b"),
    true
  );
  assert.equal(client.getQueryState(viewKey).status, "error");
});

await test("accepted deletion removes only its captured view", async (t) => {
  const { client, viewKey } = createViewsQueryFixture(t, [
    { id: "view-a", name: "A" },
    { id: "view-b", name: "B" },
  ]);
  t.mock.method(WorkspaceService.prototype, "deleteView", async () => undefined);

  await new MutationObserver(client, workspaceViewMutationOptions.delete(client, scope, 0)).mutate("view-a");

  assert.deepEqual(client.getQueryData(viewKey), [{ id: "view-b", name: "B" }]);
});

const { React, act, createPermissionUiFixture, registerPermissionUiStubs, settleQuery } = await fromRoot(
  "tests/frontend/helpers/permission-ui-fixture.mjs"
);
const { createQueryRouterFixture } = await fromRoot("tests/frontend/helpers/query-router.mjs");
const { useParams } = await import("@tanstack/react-router");
const { WorkspaceIssuePreferencesProvider, useWorkspaceIssuePreferences } = await fromRoot(
  "apps/web/core/providers/workspace-issue-preferences.tsx"
);
const { CalendarStateProvider } = await fromRoot("apps/web/core/providers/issue-layout-ui-state.tsx");
const { useWorkspaceViews } = await fromRoot("apps/web/core/hooks/use-workspace-views.ts");
const { default: userService } = await fromRoot("apps/web/core/services/user.service.ts");

async function renderViewsProbe(fixture) {
  let current;
  function Probe() {
    const result = useWorkspaceViews("acme");
    React.useLayoutEffect(() => {
      current = result;
    });
    return React.createElement("span", null, `${result.currentUserQuery.status}/${result.membershipQuery.status}`);
  }
  await fixture.render(React.createElement(Probe));
  return () => current;
}

async function retryViewsOnce(current) {
  await act(async () => {
    await current().retry();
  });
  await settleQuery();
}

await test("one retry recovers failed membership and loads the newly enabled view query", async (t) => {
  const fixture = await createQueryRouterFixture(t, "/acme/workspace-views/");
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  let offline = true;
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => {
    if (offline) throw new Error("offline");
    return { is_active: true, role: 15, updated_at: "recovered-membership" };
  });
  t.mock.method(WorkspaceService.prototype, "getAllViews", async () => [{ id: "recovered-view", name: "Recovered" }]);
  const current = await renderViewsProbe(fixture);
  assert.equal(current().membershipQuery.isError, true);
  assert.equal(current().views, undefined);
  offline = false;

  await retryViewsOnce(current);

  assert.deepEqual(current().views, [{ id: "recovered-view", name: "Recovered" }]);
});

await test("one retry recovers failed identity, fresh account membership and views", async (t) => {
  const fixture = await createQueryRouterFixture(t, "/acme/workspace-views/");
  let offline = true;
  t.mock.method(userService, "currentUser", async () => {
    if (offline) throw new Error("offline");
    return { id: "recovered-account" };
  });
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => ({
    is_active: true,
    role: 15,
    updated_at: "recovered-membership",
  }));
  t.mock.method(WorkspaceService.prototype, "getAllViews", async () => [{ id: "recovered-view", name: "Recovered" }]);
  const current = await renderViewsProbe(fixture);
  await settleQuery();
  assert.equal(current().currentUserQuery.isError, true);
  assert.equal(current().views, undefined);
  offline = false;

  await retryViewsOnce(current);

  assert.deepEqual(current().views, [{ id: "recovered-view", name: "Recovered" }]);
  assert.deepEqual(fixture.client.getQueryData(permissionKeys.workspaceMembership("recovered-account", "acme")), {
    is_active: true,
    role: 15,
    updated_at: "recovered-membership",
  });
});

const ui = { React, toasts: [] };
globalThis.workspaceViewUi = ui;
const modalStubs = {
  "@plane/propel/toast": `export const TOAST_TYPE = { SUCCESS: 'success', ERROR: 'error' };
    export const setToast = (toast) => globalThis.workspaceViewUi.toasts.push(toast);`,
  "@/components/views/form": `export function ViewForm(props) { globalThis.workspaceViewUi.form = props; return null; }`,
  "@plane/ui": `export function AlertModalCore(props) { globalThis.workspaceViewUi.alert = props; return null; }`,
  "@plane/i18n": `export const useTranslation = () => ({ t: (key) => key });`,
};
registerPermissionUiStubs("/workspace/views/modal.tsx", modalStubs);
registerPermissionUiStubs("/workspace/views/delete-view-modal.tsx", modalStubs);
const { CreateUpdateWorkspaceViewModal } = await fromRoot("apps/web/core/components/workspace/views/modal.tsx");
const { DeleteGlobalViewModal } = await fromRoot("apps/web/core/components/workspace/views/delete-view-modal.tsx");

async function modalFixture(t, Component, data) {
  // Drain native mutation observer notifications before the DOM fixture is disposed.
  t.after(settleQuery);
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/workspace-views/",
    workspaceRole: 15,
  });
  ui.toasts = [];
  let closes = 0;
  const render = (props = {}) =>
    fixture.render(
      React.createElement(Component, {
        isOpen: true,
        data,
        onClose: () => {
          closes++;
        },
        ...props,
      })
    );
  await render();
  return { ...fixture, render, closes: () => closes };
}

await test("accepted create closes the current modal and navigates once", async (t) => {
  t.mock.method(WorkspaceService.prototype, "createView", async () => ({ id: "created-view", name: "Created" }));
  const fixture = await modalFixture(t, CreateUpdateWorkspaceViewModal);

  await act(async () => {
    await ui.form.handleFormSubmit({ name: "Created" });
  });

  assert.equal(fixture.closes(), 1);
  assert.equal(fixture.router.state.location.pathname, "/acme/workspace-views/created-view");
  assert.deepEqual(
    ui.toasts.map((toast) => toast.type),
    ["success"]
  );
});

async function lateCreateFixture(t) {
  const started = Promise.withResolvers();
  const accepted = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "createView", () => {
    started.resolve();
    return accepted.promise;
  });
  const fixture = await modalFixture(t, CreateUpdateWorkspaceViewModal);
  const submitted = Promise.withResolvers();
  await act(async () => {
    ui.form.handleFormSubmit({ name: "Created" }).then(submitted.resolve, submitted.reject);
    await started.promise;
  });
  return { ...fixture, accepted, pending: submitted.promise };
}

await test("accepted create cannot close or navigate a closed and reopened modal", async (t) => {
  const fixture = await lateCreateFixture(t);
  await fixture.render({ isOpen: false });
  await fixture.render({ isOpen: true });

  await act(async () => {
    fixture.accepted.resolve({ id: "created-view", name: "Created" });
    await fixture.pending;
  });

  assert.equal(fixture.closes(), 0);
  assert.equal(fixture.router.state.location.pathname, "/acme/workspace-views/");
  assert.deepEqual(ui.toasts, []);
});

await test("accepted create cannot navigate after the route changes", async (t) => {
  const fixture = await lateCreateFixture(t);
  await act(async () => {
    await fixture.router.navigate({ href: "/acme/workspace-views/all-issues" });
  });

  await act(async () => {
    fixture.accepted.resolve({ id: "created-view", name: "Created" });
    await fixture.pending;
  });

  assert.equal(fixture.closes(), 0);
  assert.equal(fixture.router.state.location.pathname, "/acme/workspace-views/all-issues");
  assert.deepEqual(ui.toasts, []);
});

await test("accepted update cannot close or show feedback after account-session replacement", async (t) => {
  const started = Promise.withResolvers();
  const accepted = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "updateView", () => {
    started.resolve();
    return accepted.promise;
  });
  const fixture = await modalFixture(t, CreateUpdateWorkspaceViewModal, { id: "view-a", rich_filters: {} });
  let pending;
  await act(async () => {
    pending = ui.form.handleFormSubmit({ name: "Saved" });
    await started.promise;
  });
  await act(async () => {
    advanceIdentitySession(fixture.client);
  });

  await act(async () => {
    accepted.resolve({ id: "view-a", rich_filters: { changed: true } });
    await pending;
  });

  assert.equal(fixture.closes(), 0);
  assert.deepEqual(ui.toasts, []);
});

await test("accepted deletion cannot close the replacement view dialog", async (t) => {
  const started = Promise.withResolvers();
  const accepted = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "deleteView", () => {
    started.resolve();
    return accepted.promise;
  });
  const fixture = await modalFixture(t, DeleteGlobalViewModal, { id: "view-a" });
  let pending;
  await act(async () => {
    pending = ui.alert.handleSubmit();
    await started.promise;
  });
  await fixture.render({ data: { id: "view-b" } });

  await act(async () => {
    accepted.resolve();
    await pending;
  });

  assert.equal(fixture.closes(), 0);
  assert.deepEqual(ui.toasts, []);
});

await test("accepted metadata updates reach the canonical native cache before modal success", async (t) => {
  t.mock.method(WorkspaceService.prototype, "updateView", async () => ({
    id: "view-a",
    rich_filters: { changed: true },
  }));
  const fixture = await modalFixture(t, CreateUpdateWorkspaceViewModal, { id: "view-a", rich_filters: {} });
  const membership = fixture.client.getQueryData(permissionKeys.workspaceMembership("alice", "acme"));
  const key = workspaceViewKeys.list({
    accountId: "alice",
    workspaceSlug: "acme",
    workspaceRole: 15,
    membershipRevision: membership.updated_at,
    sessionGeneration: 0,
  });
  fixture.client.setQueryData(key, [{ id: "view-a", rich_filters: {} }]);

  await act(async () => {
    await ui.form.handleFormSubmit({ name: "Saved" });
  });
  assert.deepEqual(fixture.client.getQueryData(key)[0].rich_filters, { changed: true });

  assert.equal(fixture.closes(), 1);
  assert.deepEqual(
    ui.toasts.map((toast) => toast.type),
    ["success"]
  );
});

await test("a rejected create preserves the active modal and reports the primary write failure", async (t) => {
  t.mock.method(WorkspaceService.prototype, "createView", async () => {
    throw new Error("write rejected");
  });
  const fixture = await modalFixture(t, CreateUpdateWorkspaceViewModal);

  await act(async () => {
    await assert.rejects(ui.form.handleFormSubmit({ name: "Draft" }), /write rejected/);
  });

  assert.equal(fixture.closes(), 0);
  assert.deepEqual(
    ui.toasts.map((toast) => toast.type),
    ["error"]
  );
});

await test("a failed create from a closed dialog cannot show feedback in its replacement", async (t) => {
  const fixture = await lateCreateFixture(t);
  await fixture.render({ isOpen: false });
  await fixture.render({ isOpen: true });

  await act(async () => {
    fixture.accepted.reject(new Error("write rejected"));
    await assert.rejects(fixture.pending, /write rejected/);
  });

  assert.equal(fixture.closes(), 0);
  assert.deepEqual(ui.toasts, []);
});

registerPermissionUiStubs("/issues/issue-layouts/roots/all-issue-layout-root.tsx", {
  "@plane/propel/empty-state": `export function EmptyStateDetailed({title}) {
    return globalThis.workspaceViewUi.React.createElement('div', {role: 'alert'}, title);
  }`,
  "@/components/issues/peek-overview": `export const IssuePeekOverview = () => null;`,
  "@/components/views/helper": `export function WorkspaceActiveLayout({globalViewsLoading}) {
    return globalThis.workspaceViewUi.React.createElement('div', {'data-testid': 'workspace-layout'}, globalViewsLoading ? 'Loading' : 'Work items');
  }`,
  "@/components/work-item-filters/filters-hoc/workspace-level": `export const WorkspaceLevelWorkItemFiltersHOC = ({children}) => children({editor: undefined});`,
  "@/components/work-item-filters/filters-row": `export const WorkItemFiltersRow = () => null;`,
  "@/hooks/use-workspace-issue-properties": `export const useWorkspaceIssueProperties = () => {};`,
});
const { AllIssueLayoutRoot } = await fromRoot(
  "apps/web/core/components/issues/issue-layouts/roots/all-issue-layout-root.tsx"
);

async function renderIssueLayout(
  t,
  viewId,
  isDefaultView,
  viewsRead,
  { readIssues = async () => [], dynamic = false } = {}
) {
  ui.rootRequests = [];
  t.mock.method(WorkspaceService.prototype, "getViewIssues", async (workspaceSlug, params, options) => {
    ui.rootRequests.push({ workspaceSlug, params, signal: options.signal });
    const result = await readIssues(workspaceSlug, params, options.signal);
    return Array.isArray(result)
      ? { results: result, total_count: result.length, next_page_results: false, next_cursor: "", prev_cursor: "" }
      : result;
  });
  function Root() {
    const params = useParams({ strict: false });
    const preferences = useWorkspaceIssuePreferences();
    React.useLayoutEffect(() => {
      ui.rootPreferences = preferences;
    });
    const defaultView = dynamic
      ? ["all-issues", "assigned", "created", "subscribed"].includes(params.globalViewId)
      : isDefaultView;
    return React.createElement(AllIssueLayoutRoot, { isDefaultView: defaultView });
  }
  t.after(settleQuery);
  const fixture = await createPermissionUiFixture(t, {
    path: `/acme/workspace-views/${viewId}`,
    workspaceRole: 15,
  });
  t.mock.method(WorkspaceService.prototype, "getAllViews", viewsRead);
  await fixture.render(
    React.createElement(
      CalendarStateProvider,
      null,
      React.createElement(WorkspaceIssuePreferencesProvider, null, React.createElement(Root))
    )
  );
  await settleQuery();
  return fixture;
}

await Promise.all(
  ["all-issues", "assigned", "created", "subscribed"].map((viewId) =>
    test(`static ${viewId} work items remain visible when custom view metadata fails`, async (t) => {
      const fixture = await renderIssueLayout(t, viewId, true, async () => {
        throw new Error("metadata offline");
      });
      assert.equal(
        fixture.environment.container.querySelector('[data-testid="workspace-layout"]').textContent,
        "Work items"
      );
      assert.equal(fixture.environment.container.querySelector('[role="alert"]'), null);
    })
  )
);

await test("a custom view remains blocked when its metadata cannot be loaded", async (t) => {
  const fixture = await renderIssueLayout(t, "custom-view", false, async () => {
    throw new Error("metadata offline");
  });

  assert.equal(fixture.environment.container.querySelector('[data-testid="workspace-layout"]'), null);
  assert.match(fixture.environment.container.querySelector('[role="alert"]').textContent, /unavailable/);
});

await test("a missing custom view stays unavailable after a successful empty metadata response", async (t) => {
  const fixture = await renderIssueLayout(t, "missing-view", false, async () => []);

  assert.equal(
    Boolean(fixture.environment.container.querySelector('[data-testid="workspace-layout"]')),
    false,
    "the missing custom view must not render a work-item layout"
  );
  assert.match(fixture.environment.container.querySelector('[role="alert"]').textContent, /does not exist/);
});

await test("static work items remain blocked after canonical membership refresh fails", async (t) => {
  const fixture = await renderIssueLayout(t, "all-issues", true, async () => []);
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => {
    throw new Error("permissions offline");
  });

  await act(async () => {
    await fixture.client.refetchQueries({ queryKey: permissionKeys.workspaceMembership("alice", "acme"), exact: true });
  });
  await settleQuery();

  assert.equal(fixture.environment.container.querySelector('[data-testid="workspace-layout"]'), null);
  assert.match(fixture.environment.container.querySelector('[role="alert"]').textContent, /unavailable/);
});

await test("native root reloads request-affecting drafts but column choices do not refetch rows", async (t) => {
  await renderIssueLayout(t, "saved-view", false, async () => [
    { id: "saved-view", rich_filters: {}, display_filters: { layout: "spreadsheet" } },
  ]);
  assert.equal(ui.rootRequests.length, 1);
  await act(async () => {
    ui.rootPreferences.updateFilters("display_properties", { priority: false });
  });
  await settleQuery();
  assert.equal(ui.rootRequests.length, 1);
  assert.equal(ui.rootPreferences.issueFilters.displayProperties.priority, false);

  await act(async () => {
    ui.rootPreferences.updateExpression({ priority__in: "high" });
  });
  await settleQuery();

  assert.equal(ui.rootRequests.length, 2);
  assert.deepEqual(JSON.parse(ui.rootRequests[1].params.filters), { priority__in: "high" });
  assert.ok(ui.rootRequests[1].signal instanceof AbortSignal);
});

await test("a held old issue read cannot hide a replacement workspace metadata failure behind loading", async (t) => {
  const pending = Promise.withResolvers();
  const fixture = await renderIssueLayout(
    t,
    "all-issues",
    true,
    async (slug) => {
      if (slug === "beta") throw new Error("metadata offline");
      return [];
    },
    { readIssues: () => pending.promise, dynamic: true }
  );
  const oldSignal = ui.rootRequests[0].signal;
  const membership = fixture.client.getQueryData(permissionKeys.workspaceMembership("alice", "acme"));
  fixture.client.setQueryData(permissionKeys.workspaceMembership("alice", "beta"), membership);

  await act(async () => {
    await fixture.router.navigate({ href: "/beta/workspace-views/custom-b" });
  });
  await settleQuery();
  await act(async () => {
    pending.resolve([]);
  });
  await settleQuery();

  assert.equal(oldSignal.aborted, true);
  assert.equal(fixture.environment.container.querySelector('[data-testid="workspace-layout"]'), null);
  assert.match(fixture.environment.container.querySelector('[role="alert"]').textContent, /unavailable/);
});
