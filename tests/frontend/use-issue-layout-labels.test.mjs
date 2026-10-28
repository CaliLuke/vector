import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { waitForDom } from "./helpers/wait-for-dom.mjs";

registerWebModuleHooks();
const { createReactDomTestEnvironment } = await import("./helpers/react-dom.mjs");
const { React, webRequire } = await import("./helpers/react.mjs");
const { QueryClient, QueryClientProvider } = await import("@tanstack/react-query");
const { EUserPermissions } = webRequire("@plane/constants");
const { EUserWorkspaceRoles } = webRequire("@plane/types");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { IssueLabelService } = await import("../../apps/web/core/services/issue/issue_label.service.ts");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { useIssueLayoutLabels } = await import("../../apps/web/core/hooks/use-issue-layout-labels.ts");

const ACCOUNT = "layout-label-account";
const WORKSPACE = "layout-label-workspace";
const PROJECT = "layout-label-project";
const REVISION = "membership-revision";

function createClient(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  client.setQueryData(identityKeys.currentUser(), { id: ACCOUNT });
  client.setQueryData(permissionKeys.workspaceMembership(ACCOUNT, WORKSPACE), {
    is_active: true,
    role: EUserWorkspaceRoles.MEMBER,
    updated_at: REVISION,
  });
  t.after(() => client.clear());
  return client;
}

function Probe({ workspaceLevel }) {
  const result = useIssueLayoutLabels(WORKSPACE, workspaceLevel ? undefined : PROJECT, workspaceLevel);
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(
      "output",
      { "data-testid": "status" },
      JSON.stringify({
        labels: result.labels?.map((label) => label.id),
        names: result.labels?.map((label) => label.name),
        byId: [...(result.labelsById?.keys() ?? [])],
        scopeKey: result.scopeKey,
        idsRevision: result.idsRevision,
        isPending: result.isPending,
        isError: result.isError,
        isAccessDenied: result.isAccessDenied,
        scopeIsReady: result.scopeIsReady,
        scopeIsError: result.scopeIsError,
      })
    ),
    React.createElement("button", { type: "button", onClick: () => void result.retry() }, "Retry")
  );
}

function readStatus(environment) {
  const output = environment.container.querySelector('[data-testid="status"]')?.textContent;
  return output ? JSON.parse(output) : undefined;
}

function waitForStatus(environment, predicate) {
  return waitForDom(
    environment.container,
    () => predicate(readStatus(environment)),
    "Timed out waiting for label scope"
  );
}

function assertCatalogueUnavailableBeforeRetry(environment) {
  const state = readStatus(environment);
  assert.equal(state.labels, undefined);
  assert.equal(state.idsRevision, undefined);
  assert.equal(state.isPending, false);
  return state;
}

async function renderWorkspaceProbe(environment, client) {
  await environment.render(
    React.createElement(QueryClientProvider, { client }, React.createElement(Probe, { workspaceLevel: true }))
  );
  await React.act(async () => waitForStatus(environment, (status) => status?.scopeIsError));
}

async function retryUntilCanonicalEmpty(environment) {
  await React.act(async () => {
    environment.container.querySelector("button").click();
    await waitForStatus(environment, (status) => status?.idsRevision === "[]" && !status.isError);
  });
}

function mockWorkspaceCatalogueReads(t, { membership, roles }) {
  const reads = { membership: 0, roles: 0, labels: 0 };
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => {
    reads.membership += 1;
    return membership(reads.membership);
  });
  t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => {
    reads.roles += 1;
    return roles(reads.roles);
  });
  t.mock.method(IssueLabelService.prototype, "getWorkspaceIssueLabels", async () => {
    reads.labels += 1;
    return [];
  });
  return reads;
}

void test("project layout exposes only successful canonical labels and their sorted ID revision", async (t) => {
  const client = createClient(t);
  const environment = await createReactDomTestEnvironment();
  let labelReads = 0;
  const refreshStarted = Promise.withResolvers();
  const completeRefresh = Promise.withResolvers();
  t.after(async () => {
    await environment.unmount();
  });
  t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => ({
    [PROJECT]: EUserPermissions.ADMIN,
  }));
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async () => [
    ...(labelReads++ === 0
      ? [
          { id: "label-z", name: "Z", sort_order: 2 },
          { id: "label-a", name: "A", sort_order: 1 },
        ]
      : await (async () => {
          refreshStarted.resolve();
          return completeRefresh.promise;
        })()),
  ]);

  await environment.render(
    React.createElement(QueryClientProvider, { client }, React.createElement(Probe, { workspaceLevel: false }))
  );
  await React.act(async () => waitForStatus(environment, (status) => status?.idsRevision === '["label-a","label-z"]'));

  let state = readStatus(environment);
  assert.deepEqual(state.labels, ["label-a", "label-z"]);
  assert.deepEqual(state.byId, ["label-a", "label-z"]);
  assert.equal(state.idsRevision, '["label-a","label-z"]');
  assert.match(state.scopeKey, new RegExp(PROJECT));
  assert.equal(state.isPending, false);
  assert.equal(state.isError, false);
  assert.equal(state.scopeIsReady, true);
  assert.equal(state.scopeIsError, false);

  await React.act(async () => {
    environment.container.querySelector("button").click();
    await refreshStarted.promise;
  });
  state = readStatus(environment);
  assert.equal(state.idsRevision, '["label-a","label-z"]');
  assert.equal(state.isPending, false);
  completeRefresh.resolve([
    { id: "label-z", name: "Z updated", sort_order: 0 },
    { id: "label-a", name: "A updated", sort_order: 1 },
  ]);
  await React.act(async () => waitForStatus(environment, (current) => current?.names.includes("A updated")));
  state = readStatus(environment);
  assert.deepEqual(state.labels, ["label-z", "label-a"]);
  assert.deepEqual(state.byId, ["label-z", "label-a"]);
});

void test("workspace layout retries a failed role prerequisite and then exposes canonical empty labels", async (t) => {
  const client = createClient(t);
  const environment = await createReactDomTestEnvironment();
  const reads = mockWorkspaceCatalogueReads(t, {
    membership: async () => ({ is_active: true, role: EUserWorkspaceRoles.MEMBER, updated_at: REVISION }),
    roles: async (read) => {
      if (read === 1) throw new Error("roles unavailable");
      return { [PROJECT]: EUserPermissions.MEMBER };
    },
  });
  t.after(async () => {
    await environment.unmount();
  });

  await renderWorkspaceProbe(environment, client);
  assertCatalogueUnavailableBeforeRetry(environment);
  assert.equal(reads.roles, 1);
  assert.equal(reads.labels, 0);

  await retryUntilCanonicalEmpty(environment);
  const state = readStatus(environment);
  assert.deepEqual(state.labels, []);
  assert.deepEqual(state.byId, []);
  assert.equal(state.idsRevision, "[]");
  assert.match(state.scopeKey, new RegExp(WORKSPACE));
  assert.equal(state.isPending, false);
  assert.equal(state.isError, false);
  assert.equal(state.scopeIsReady, true);
  assert.equal(state.scopeIsError, false);
  assert.equal(reads.roles, 2);
  assert.equal(reads.labels, 1);
});

void test("workspace layout rebuilds its scope after membership retry", async (t) => {
  const client = createClient(t);
  client.removeQueries({ queryKey: permissionKeys.workspaceMembership(ACCOUNT, WORKSPACE), exact: true });
  const environment = await createReactDomTestEnvironment();
  const reads = mockWorkspaceCatalogueReads(t, {
    membership: async (read) => {
      if (read === 1) throw new Error("membership unavailable");
      return { is_active: true, role: EUserWorkspaceRoles.MEMBER, updated_at: REVISION };
    },
    roles: async () => ({ [PROJECT]: EUserPermissions.MEMBER }),
  });
  t.after(() => environment.unmount());

  await renderWorkspaceProbe(environment, client);
  assertCatalogueUnavailableBeforeRetry(environment);
  assert.equal(reads.membership, 1);
  assert.equal(reads.roles, 0);
  assert.equal(reads.labels, 0);

  await retryUntilCanonicalEmpty(environment);
  const state = readStatus(environment);
  assert.equal(state.scopeIsReady, true);
  assert.equal(state.scopeIsError, false);
  assert.equal(reads.membership, 2);
  assert.equal(reads.roles, 1);
  assert.equal(reads.labels, 1);
});

void test("a resolved inactive membership is denied instead of left in a loading state", async (t) => {
  const client = createClient(t);
  const environment = await createReactDomTestEnvironment();
  client.setQueryData(permissionKeys.workspaceMembership(ACCOUNT, WORKSPACE), {
    is_active: false,
    role: EUserWorkspaceRoles.MEMBER,
    updated_at: REVISION,
  });
  t.after(() => environment.unmount());
  t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => {
    throw new Error("inactive memberships must not load project roles");
  });
  t.mock.method(IssueLabelService.prototype, "getProjectLabels", async () => {
    throw new Error("inactive memberships must not load labels");
  });

  await environment.render(
    React.createElement(QueryClientProvider, { client }, React.createElement(Probe, { workspaceLevel: false }))
  );
  await React.act(async () => waitForStatus(environment, (status) => status?.isAccessDenied));

  const state = readStatus(environment);
  assert.equal(state.labels, undefined);
  assert.equal(state.idsRevision, undefined);
  assert.equal(state.isPending, false);
  assert.equal(state.isError, false);
  assert.equal(state.isAccessDenied, true);
  assert.equal(state.scopeIsReady, false);
  assert.equal(state.scopeIsError, true);
});
