import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { parentCatalogueFixture } = await import("./helpers/parent-catalogue-fixture.mjs");
const { ProjectService } = await import("../../apps/web/core/services/project/index.ts");
const { projectIssueSearchOptions } = await import("../../apps/web/core/queries/project-issue-search.ts");
const { currentParentCatalogueScope } = await import("../../apps/web/core/queries/cycle-module-scope.ts");
const { advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const row = { id: "issue", project_id: "p1", name: "Private item" };
const options = (client, params = {}) => {
  const { project, workspace } = currentParentCatalogueScope(client, "acme", "p1");
  return projectIssueSearchOptions(client, project, workspace, {
    search: "private",
    workspace_search: false,
    ...params,
  });
};

await test("project search validates canonical project ownership", async (t) => {
  const { client } = await parentCatalogueFixture(t);
  t.mock.method(ProjectService.prototype, "projectIssuesSearch", async () => [row]);
  assert.deepEqual(await client.fetchQuery(options(client)), [row]);
  t.mock.method(ProjectService.prototype, "projectIssuesSearch", async () => [{ ...row, project_id: "foreign" }]);
  await assert.rejects(client.fetchQuery({ ...options(client), staleTime: 0 }), /unexpected project/);
});

await test("replacing permission queries invalidates a search already in flight", async (t) => {
  const { client, rolesKey } = await parentCatalogueFixture(t);
  const deferred = Promise.withResolvers();
  t.mock.method(ProjectService.prototype, "projectIssuesSearch", () => deferred.promise);
  const requestOptions = options(client);
  const pending = client.fetchQuery(requestOptions);
  client.removeQueries({ queryKey: rolesKey, exact: true });
  client.setQueryData(rolesKey, { p1: 20 });
  deferred.resolve([row]);
  await assert.rejects(pending);
  assert.equal(client.getQueryData(requestOptions.queryKey), undefined);
});

await test("changing another project role rejects a workspace search response", async (t) => {
  const { client, rolesKey } = await parentCatalogueFixture(t);
  client.setQueryData(rolesKey, { p1: 20, p2: 20 });
  const deferred = Promise.withResolvers();
  t.mock.method(ProjectService.prototype, "projectIssuesSearch", () => deferred.promise);
  const requestOptions = options(client, { workspace_search: true });
  const pending = client.fetchQuery(requestOptions);
  client.setQueryData(rolesKey, { p1: 20 });
  deferred.resolve([{ ...row, project_id: "p2" }]);
  await assert.rejects(pending);
  assert.equal(client.getQueryData(requestOptions.queryKey), undefined);
});

await test("session changes and cancellation cannot repopulate search results", async (t) => {
  const { client } = await parentCatalogueFixture(t);
  const deferred = Promise.withResolvers();
  let signal;
  t.mock.method(ProjectService.prototype, "projectIssuesSearch", (_workspace, _project, _params, requestSignal) => {
    signal = requestSignal;
    return deferred.promise;
  });
  const requestOptions = options(client);
  const pending = client.fetchQuery(requestOptions);
  const rejected = assert.rejects(pending);
  advanceIdentitySession(client);
  await client.cancelQueries({ queryKey: requestOptions.queryKey, exact: true });
  assert.equal(signal.aborted, true);
  deferred.resolve([row]);
  await rejected;
  assert.equal(client.getQueryData(requestOptions.queryKey), undefined);
});
