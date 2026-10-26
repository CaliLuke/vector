import assert from "node:assert/strict";
import test from "node:test";
import { pathToFileURL } from "node:url";
import path from "node:path";

const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { React, act, createPermissionUiFixture, settleQuery } = await fromRoot(
  "tests/frontend/helpers/permission-ui-fixture.mjs"
);
const { EIssuesStoreType, EUserPermissions, EUserWorkspaceRoles } = await fromRoot("packages/types/dist/index.mjs");
const { permissionKeys } = await fromRoot("apps/web/core/queries/permissions.ts");
const { IssueService } = await fromRoot("apps/web/core/services/issue/index.ts");
const { useIssueListQuery } = await fromRoot("apps/web/core/hooks/use-issue-list-query.ts");
const { useIssueListRead } = await fromRoot("apps/web/core/hooks/use-issue-list-read.ts");
const { ProjectIssuePreferencesContext } = await fromRoot(
  "apps/web/core/hooks/use-project-issue-preferences-context.tsx"
);
const { currentParentCatalogueScope } = await fromRoot("apps/web/core/queries/cycle-module-scope.ts");
const { issueListKeys } = await fromRoot("apps/web/core/queries/issue-lists.ts");
const { useIssuesActions } = await fromRoot("apps/web/core/hooks/use-issues-actions.tsx");
const response = (issue) => ({
  grouped_by: null,
  next_cursor: "",
  prev_cursor: "",
  next_page_results: false,
  prev_page_results: false,
  total_count: 1,
  count: 1,
  total_pages: 1,
  extra_stats: null,
  total_results: 1,
  results: [issue],
});

async function mountActions(fixture, onRender) {
  function Probe() {
    onRender(useIssuesActions(EIssuesStoreType.PROJECT));
    return null;
  }
  await fixture.render(React.createElement(Probe));
}

await test("workspace header can create in another authorized project through native issue actions", async (t) => {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/",
    userId: "alice",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { target: EUserPermissions.MEMBER },
  });
  let actions;
  await mountActions(fixture, (value) => (actions = value));
  const receipt = { id: "created-1", project_id: "target", name: "Created from workspace" };
  let request;
  t.mock.method(IssueService.prototype, "createIssue", async (workspaceSlug, projectId, data) => {
    request = { workspaceSlug, projectId, data };
    return receipt;
  });

  assert.deepEqual(await actions.createIssue("target", { name: "Created from workspace" }), receipt);
  assert.deepEqual(request, {
    workspaceSlug: "acme",
    projectId: "target",
    data: { name: "Created from workspace" },
  });
  assert.equal(fixture.router.state.matches.at(-1).routeId, "/_authenticated/$workspaceSlug/_projects/");
});

await test("native create rejects when the target project permission query fails", async (t) => {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/",
    userId: "alice",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { target: EUserPermissions.MEMBER },
    rolesRead: async () => {
      throw new Error("project roles unavailable");
    },
  });
  fixture.client.removeQueries({
    queryKey: permissionKeys.workspaceProjectRoles("alice", "acme", EUserWorkspaceRoles.MEMBER, "2026-10-24T00:00:00Z"),
    exact: true,
  });
  let actions;
  await mountActions(fixture, (value) => (actions = value));
  await settleQuery();
  let createCalls = 0;
  t.mock.method(IssueService.prototype, "createIssue", async () => {
    createCalls++;
    return { id: "created-1", project_id: "target" };
  });

  await assert.rejects(actions.createIssue("target", { name: "Must be denied" }), /access is not ready/i);
  assert.equal(createCalls, 0);
  assert.ok(
    fixture.client
      .getQueryCache()
      .findAll({ queryKey: ["permissions", "alice", "acme", "project-roles"] })
      .some((query) => query.state.status === "error"),
    "the target-project role read must be in an error state"
  );
});

await test("native create rejects a target project outside the verified roles response", async (t) => {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/",
    userId: "alice",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { allowed: EUserPermissions.MEMBER },
  });
  let actions;
  await mountActions(fixture, (value) => (actions = value));
  let createCalls = 0;
  t.mock.method(IssueService.prototype, "createIssue", async () => {
    createCalls++;
    return { id: "created-1", project_id: "denied" };
  });

  await assert.rejects(actions.createIssue("denied", { name: "Must be denied" }), /access is not ready/i);
  assert.equal(createCalls, 0);
});

await test("a query mounted after a pending write is cancelled and refetched without direct receipt projection", async (t) => {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/",
    userId: "alice",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { target: EUserPermissions.MEMBER },
  });
  let actions;
  let listState;
  let showList = false;
  const existing = { id: "item-1", project_id: "target", name: "Before" };
  const workspaceScope = currentParentCatalogueScope(fixture.client, "acme", "target").workspace;
  const projectScope = currentParentCatalogueScope(fixture.client, "acme", "target").project;
  const workspaceRead = {
    scope: workspaceScope,
    pagination: { canGroup: false, perPageCount: 20 },
    request: {
      family: "workspace",
      input: {
        kind: "workspace",
        workspaceSlug: "acme",
        viewId: "all-issues",
        accountId: "alice",
        richFilters: {},
        displayFilters: { layout: "list", order_by: "created_at" },
        isScopeCurrent: () => true,
        isReadScopeCurrent: () => true,
      },
      isReadCurrent: () => true,
    },
  };
  const workspaceKey = issueListKeys.list(workspaceRead);
  fixture.client.setQueryData(workspaceKey, { generation: "workspace-g1", response: response(existing) });
  const projectRead = {
    scope: projectScope,
    pagination: { canGroup: false, perPageCount: 20 },
    request: {
      family: "project",
      input: {
        workspaceSlug: "acme",
        projectId: "target",
        richFilters: {},
        displayFilters: { layout: "list", order_by: "created_at" },
      },
      isReadCurrent: () => true,
    },
  };
  const writeStarted = Promise.withResolvers();
  const acceptWrite = Promise.withResolvers();
  const firstReadStarted = Promise.withResolvers();
  const firstRead = Promise.withResolvers();
  const secondReadStarted = Promise.withResolvers();
  const secondRead = Promise.withResolvers();
  const readSignals = [];
  let readCalls = 0;
  t.mock.method(IssueService.prototype, "patchIssue", () => {
    writeStarted.resolve();
    return acceptWrite.promise;
  });
  t.mock.method(IssueService.prototype, "getIssues", (_workspace, _project, _params, options) => {
    readCalls++;
    readSignals.push(options.signal);
    if (readCalls === 1) {
      firstReadStarted.resolve();
      return firstRead.promise;
    }
    secondReadStarted.resolve();
    return secondRead.promise;
  });
  function ActionProbe() {
    const value = useIssuesActions(EIssuesStoreType.PROJECT);
    React.useEffect(() => {
      actions = value;
    }, [value]);
    return null;
  }
  function ListProbe() {
    const value = useIssueListQuery(projectRead);
    React.useEffect(() => {
      listState = value;
    }, [value]);
    return null;
  }
  function Tree() {
    return React.createElement(
      React.Fragment,
      null,
      React.createElement(ActionProbe),
      showList ? React.createElement(ListProbe) : null
    );
  }
  await fixture.render(React.createElement(Tree));
  const pendingWrite = actions.updateIssue("target", existing.id, { name: "After" });
  await writeStarted.promise;
  await act(async () => {
    await fixture.router.navigate({ href: "/acme/projects/target/issues/" });
  });
  showList = true;
  await fixture.render(React.createElement(Tree));
  await firstReadStarted.promise;
  const projectKey = issueListKeys.list(projectRead);
  assert.ok(fixture.client.getQueryCache().find({ queryKey: projectKey, exact: true }));

  const accepted = { ...existing, name: "After" };
  acceptWrite.resolve(accepted);
  await secondReadStarted.promise;
  assert.equal(readSignals[0].aborted, true, "the late target GET is cancelled after the accepted write");
  assert.equal(fixture.client.getQueryData(workspaceKey).response.results[0].name, "After");
  assert.equal(
    fixture.client.getQueryData(projectKey),
    undefined,
    "the newly mounted Query receives no direct receipt write"
  );

  secondRead.resolve(response(accepted));
  await pendingWrite;
  await settleQuery();
  firstRead.resolve(response(existing));
  await settleQuery();
  assert.equal(readCalls, 2, "the active target Query refetches after cancellation");
  assert.equal(fixture.client.getQueryData(projectKey).response.results[0].name, "After");
  assert.deepEqual(Object.keys(listState.issuesById), [existing.id]);
});

await test("same-project preference failure hides cached issue rows and exposes list retry", async (t) => {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/",
    userId: "alice",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { p1: EUserPermissions.MEMBER },
  });
  const requestInput = {
    workspaceSlug: "acme",
    projectId: "p1",
    accountId: "alice",
    richFilters: {},
    displayFilters: { layout: "list", order_by: "created_at" },
    isCurrent: () => true,
    isReadScopeCurrent: () => true,
  };
  const cachedIssue = { id: "cached-issue", project_id: "p1", name: "Must disappear" };
  const cachedResponse = {
    grouped_by: null,
    next_cursor: "",
    prev_cursor: "",
    next_page_results: false,
    prev_page_results: false,
    total_count: 1,
    count: 1,
    total_pages: 1,
    extra_stats: null,
    total_results: 1,
    results: [cachedIssue],
  };
  const scope = currentParentCatalogueScope(fixture.client, "acme", "p1").project;
  const pagination = { canGroup: false, perPageCount: 20 };
  const listKey = issueListKeys.list({
    scope,
    pagination,
    request: { family: "project", input: requestInput, isReadCurrent: () => true },
  });
  fixture.client.setQueryData(listKey, { generation: "cached-generation", response: cachedResponse });
  let reads = 0;
  t.mock.method(IssueService.prototype, "getIssues", async () => {
    reads++;
    return cachedResponse;
  });
  const preferenceContext = {
    workspaceSlug: "acme",
    projectId: "p1",
    properties: undefined,
    issueFilters: { richFilters: {}, displayFilters: requestInput.displayFilters, displayProperties: {} },
    requestInput,
    isPending: false,
    isError: true,
    refetch: async () => {},
    save: async () => {},
  };
  let latest;
  function Probe() {
    const read = useIssueListRead(EIssuesStoreType.PROJECT, pagination);
    const value = useIssueListQuery(read);
    React.useEffect(() => {
      latest = value;
    }, [value]);
    return null;
  }
  await fixture.render(
    React.createElement(
      ProjectIssuePreferencesContext.Provider,
      { value: preferenceContext },
      React.createElement(Probe)
    )
  );
  await settleQuery();

  assert.equal(
    fixture.client.getQueryData(listKey).response.results[0].name,
    "Must disappear",
    "the Query cache is retained"
  );
  assert.deepEqual(latest.issuesById, {}, "failed preference verification hides cached rows from the list projection");
  assert.equal(latest.isError, true);
  assert.equal(typeof latest.retry, "function", "the list surface can expose Retry");
  assert.equal(reads, 0, "the failed preference scope is rejected before transport");
});
