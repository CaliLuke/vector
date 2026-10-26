import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedIssueReadAccess } from "./helpers/issue-query-fixtures.mjs";
registerWebModuleHooks();
const { tablePropertyValuesOptions, tablePropertyBatches, PropertyCatalogChanged } =
  await import("../../apps/web/core/queries/work-item-table-values.ts");
const { IssueService } = await import("../../apps/web/core/services/issue/issue.service.ts");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const scope = {
  accountId: "alice",
  workspaceSlug: "acme",
  projectId: "p1",
  generation: 0,
  membershipRole: 15,
  membershipRevision: "read-scope-1",
  projectRole: 20,
  membershipProjectRole: 20,
};
const item = {
  id: "one",
  project_id: "p1",
  updated_at: "v1",
  property_values: { number: { status: "value", value: 0 } },
  properties_evaluated_at: "2026-10-25T10:00:00Z",
  properties_revision: 7,
};
function fixture(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  seedIssueReadAccess(client);
  t.after(() => client.clear());
  return client;
}
await test("table values are scoped batched reads and do not seed from shared item projections", async (t) => {
  const client = fixture(t);
  const read = t.mock.method(IssueService.prototype, "retrieveIssues", async (_w, _p, _ids, options) => {
    assert.equal(options.includeProperties, true);
    assert.ok(options.signal);
    return [item];
  });
  const options = tablePropertyValuesOptions(scope, 7, [item, { id: "missing", updated_at: "v1" }]);
  assert.equal(options.initialData, undefined);
  assert.deepEqual(await client.fetchQuery(options), {
    one: { property_values: item.property_values, properties_evaluated_at: item.properties_evaluated_at },
  });
  assert.equal(read.mock.callCount(), 1);
  assert.notDeepEqual(options.queryKey, tablePropertyValuesOptions(scope, 7, [{ ...item, updated_at: "v2" }]).queryKey);
  assert.notDeepEqual(options.queryKey, tablePropertyValuesOptions(scope, 8, [item]).queryKey);
});
await test("late table responses cannot survive account or permission changes", async (t) => {
  await Promise.all(
    ["account", "permission"].map((scenario) =>
      t.test(scenario, async (subtest) => {
        const client = fixture(subtest);
        const pending = Promise.withResolvers();
        subtest.mock.method(IssueService.prototype, "retrieveIssues", () => pending.promise);
        const options = tablePropertyValuesOptions(scope, 7, [item]);
        const read = client.fetchQuery(options);
        if (scenario === "account") client.setQueryData(identityKeys.currentUser(), { id: "bob" });
        else client.setQueryData(permissionKeys.workspaceProjectRoles("alice", "acme", 15, "read-scope-1"), { p1: 5 });
        pending.resolve([item]);
        await assert.rejects(read, /access changed/);
        assert.equal(client.getQueryData(options.queryKey), undefined);
      })
    )
  );
});
await test("table responses reject catalog drift and rows outside requested project/IDs", async (t) => {
  await Promise.all(
    [
      [{ ...item, properties_revision: 8 }, PropertyCatalogChanged],
      [{ ...item, project_id: "other" }, /Invalid/],
      [{ ...item, id: "other" }, /Invalid/],
      [{ ...item, property_values: undefined }, /unavailable/],
    ].map(([row, error], index) =>
      t.test(String(index), async (subtest) => {
        const client = fixture(subtest);
        subtest.mock.method(IssueService.prototype, "retrieveIssues", async () => [row]);
        await assert.rejects(client.fetchQuery(tablePropertyValuesOptions(scope, 7, [item])), error);
      })
    )
  );
});
await test("batches partition projects, deduplicate IDs and bound request sizes without drafts or archives", () => {
  const items = Array.from({ length: 205 }, (_, n) => ({ id: String(n), project_id: "p1", updated_at: "v1" }));
  items.push(
    items[0],
    { id: "cross", project_id: "p2" },
    { id: "draft", project_id: "p1", is_draft: true },
    { id: "archive", project_id: "p1", archived_at: "now" }
  );
  const batches = tablePropertyBatches(items);
  assert.deepEqual(
    batches.map((batch) => [batch.projectId, batch.items.length]),
    [
      ["p1", 100],
      ["p1", 100],
      ["p1", 5],
      ["p2", 1],
    ]
  );
});

await test("workspace admin table reads retain raw project role in their key and guard", async (t) => {
  const client = fixture(t);
  client.setQueryData(permissionKeys.workspaceMembership("alice", "acme"), {
    is_active: true,
    role: 20,
    updated_at: "admin-rev",
  });
  const roles = permissionKeys.workspaceProjectRoles("alice", "acme", 20, "admin-rev");
  client.setQueryData(roles, { p1: 15 });
  const admin = { ...scope, membershipRole: 20, membershipRevision: "admin-rev", membershipProjectRole: 15 };
  const pending = Promise.withResolvers();
  t.mock.method(IssueService.prototype, "retrieveIssues", () => pending.promise);
  const options = tablePropertyValuesOptions(admin, 7, [item]);
  const read = client.fetchQuery(options);
  client.setQueryData(roles, { p1: 5 });
  pending.resolve([item]);
  await assert.rejects(read, /access changed/);
  assert.equal(client.getQueryData(options.queryKey), undefined);
  assert.notDeepEqual(
    options.queryKey,
    tablePropertyValuesOptions({ ...admin, membershipProjectRole: 5 }, 7, [item]).queryKey
  );
});
