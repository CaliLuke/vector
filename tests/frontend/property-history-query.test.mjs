import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { identityKeys, advanceIdentitySession, getIdentitySessionGeneration } =
  await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const {
  propertyHistoryCatalogOptions,
  propertyVersionPageOptions,
  propertyVersionDetailOptions,
  refreshPropertyHistory,
} = await import("../../apps/web/core/queries/work-item-property-history.ts");
const { workItemPropertyHistoryService: service } =
  await import("../../apps/web/core/services/work-item-property-history.service.ts");
function fixture(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  const scope = {
    accountId: "alice",
    workspaceSlug: "acme",
    projectId: "project",
    issueId: "item",
    membershipRole: 15,
    membershipRevision: "v1",
    projectRole: 5,
    generation: getIdentitySessionGeneration(client),
  };
  const memberKey = permissionKeys.workspaceMembership("alice", "acme");
  const rolesKey = permissionKeys.workspaceProjectRoles("alice", "acme", 15, "v1");
  client.setQueryData(memberKey, { is_active: true, role: 15, updated_at: "v1" });
  client.setQueryData(rolesKey, { project: 5 });
  t.after(() => client.clear());
  return { client, scope, memberKey, rolesKey };
}
await test("read-only history forwards exact item/field/revision scope and cursor with cancellation", async (t) => {
  const { client, scope } = fixture(t);
  const calls = [];
  t.mock.method(service, "catalog", async (...args) => {
    calls.push(args);
    return [{ property_id: "deleted-field", title: "Historical title" }];
  });
  t.mock.method(service, "versions", async (...args) => {
    calls.push(args);
    return { results: [{ id: "older-version" }], next_cursor: null };
  });
  t.mock.method(service, "detail", async (...args) => {
    calls.push(args);
    return { id: "older-version", value: null };
  });
  assert.equal((await client.fetchQuery(propertyHistoryCatalogOptions(scope)))[0].title, "Historical title");
  assert.equal(
    (await client.fetchQuery(propertyVersionPageOptions(scope, "deleted-field", "cursor-id"))).results[0].id,
    "older-version"
  );
  assert.equal(
    (await client.fetchQuery(propertyVersionDetailOptions(scope, "deleted-field", "older-version"))).value,
    null
  );
  assert.deepEqual(calls[1].slice(0, -1), [scope, "deleted-field", "cursor-id"]);
  assert.deepEqual(calls[2].slice(0, -1), [scope, "deleted-field", "older-version"]);
  assert.ok(calls.every((args) => args.at(-1) instanceof AbortSignal));
});
for (const method of ["catalog", "versions", "detail"]) {
  for (const change of ["membership", "role", "session", "query"]) {
    void test(`${method} rejects late content after ${change} replacement`, async (t) => {
      const { client, scope, memberKey, rolesKey } = fixture(t),
        started = Promise.withResolvers(),
        response = Promise.withResolvers();
      t.mock.method(service, method, () => {
        started.resolve();
        return response.promise;
      });
      const options = {
        catalog: () => propertyHistoryCatalogOptions(scope),
        versions: () => propertyVersionPageOptions(scope, "field", null),
        detail: () => propertyVersionDetailOptions(scope, "field", "revision"),
      }[method]();
      const reading = client.fetchQuery(options),
        rejected = assert.rejects(reading);
      await started.promise;
      if (change === "membership") client.setQueryData(memberKey, { is_active: false, role: 15, updated_at: "v2" });
      if (change === "role") client.setQueryData(rolesKey, {});
      if (change === "session") advanceIdentitySession(client);
      if (change === "query") client.removeQueries({ queryKey: options.queryKey, exact: true });
      response.resolve({ secret: "historical content" });
      await rejected;
      assert.equal(client.getQueryData(options.queryKey), undefined);
    });
  }
}
await test("revision cache identity and refresh are scoped to item, field, cursor and account generation", async (t) => {
  const { client, scope } = fixture(t);
  const current = propertyVersionDetailOptions(scope, "field", "v1");
  const other = propertyVersionDetailOptions({ ...scope, issueId: "other" }, "field", "v1");
  assert.notDeepEqual(current.queryKey, propertyVersionDetailOptions(scope, "field", "v2").queryKey);
  assert.notDeepEqual(
    current.queryKey,
    propertyVersionDetailOptions({ ...scope, generation: scope.generation + 1 }, "field", "v1").queryKey
  );
  client.setQueryData(current.queryKey, { id: "v1" });
  client.setQueryData(other.queryKey, { id: "v1" });
  await refreshPropertyHistory(client, scope);
  assert.equal(client.getQueryState(current.queryKey).isInvalidated, true);
  assert.equal(client.getQueryState(other.queryKey).isInvalidated, false);
});
