import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { itemDetailSchemaOptions, refreshItemSchema } =
  await import("../../apps/web/core/queries/work-item-detail-schema.ts");
const { workItemSchemaService } = await import("../../apps/web/core/services/work-item-schema.service.ts");
const scope = {
  accountId: "alice",
  generation: 0,
  workspaceSlug: "acme",
  projectId: "p",
  issueId: "i",
  membershipRole: 15,
  membershipRevision: "v1",
  projectRole: 15,
  refreshVersion: "optimistic",
};
function setup(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  const memberKey = permissionKeys.workspaceMembership("alice", "acme");
  const rolesKey = permissionKeys.workspaceProjectRoles("alice", "acme", 15, "v1");
  client.setQueryData(memberKey, { is_active: true, role: 15, updated_at: "v1" });
  client.setQueryData(rolesKey, { p: 15 });
  t.after(() => client.clear());
  return { client, memberKey, rolesKey, options: itemDetailSchemaOptions(scope) };
}
await test("detail reads canonical type and exposes a coherent server response", async (t) => {
  const f = setup(t),
    snapshot = {
      stored_values: { effort: 0, verified: false },
      computed: { formula: { status: "value", value: 1 } },
      item_updated_at: "server-version",
    };
  let request;
  t.mock.method(workItemSchemaService, "schema", async (...args) => {
    request = args;
    return snapshot;
  });
  assert.deepEqual(await f.client.fetchQuery(f.options), snapshot);
  assert.deepEqual(request.slice(0, 2), ["acme", { mode: "edit", project_id: "p", issue_id: "i" }]);
  assert.ok(request[2] instanceof AbortSignal);
});
for (const change of ["membership", "role", "session", "query"]) {
  void test(`late schema cannot survive ${change} replacement`, async (sub) => {
    const f = setup(sub),
      started = Promise.withResolvers(),
      response = Promise.withResolvers();
    sub.mock.method(workItemSchemaService, "schema", () => {
      started.resolve();
      return response.promise;
    });
    const reading = f.client.fetchQuery(f.options);
    const rejected = assert.rejects(reading);
    await started.promise;
    if (change === "membership") f.client.setQueryData(f.memberKey, { is_active: false, role: 15, updated_at: "v2" });
    if (change === "role") f.client.setQueryData(f.rolesKey, {});
    if (change === "session") advanceIdentitySession(f.client);
    if (change === "query") f.client.removeQueries({ queryKey: f.options.queryKey, exact: true });
    response.resolve({ stored_values: { secret: "private" } });
    await rejected;
    assert.equal(f.client.getQueryData(f.options.queryKey), undefined);
  });
}
await test("accepted edits invalidate the earlier optimistic read only for the owning item", async (t) => {
  const f = setup(t);
  const other = itemDetailSchemaOptions({ ...scope, issueId: "other" });
  f.client.setQueryData(f.options.queryKey, { computed: { value: 1 } });
  f.client.setQueryData(other.queryKey, { computed: { value: 2 } });
  await refreshItemSchema(f.client, "acme", "i");
  assert.equal(f.client.getQueryState(f.options.queryKey).isInvalidated, true);
  assert.equal(f.client.getQueryState(other.queryKey).isInvalidated, false);
});
await test("clock formulas refresh at minute boundaries; ordinary formulas do not poll", (t) => {
  t.mock.timers.enable({ apis: ["Date"], now: new Date("2026-10-26T23:59:30Z") });
  const { refetchInterval } = itemDetailSchemaOptions(scope);
  assert.equal(refetchInterval(field("NOW()")), 30000);
  assert.equal(refetchInterval(field("today () + 2")), 30000);
  t.mock.timers.tick(30000);
  assert.equal(refetchInterval(field("NOW()")), 60000);
  assert.equal(refetchInterval(field("today () + 2")), 60000);
  assert.equal(refetchInterval(field("{{effort}} + 2")), false);
});

function field(expression) {
  return { state: { data: { properties: [{ kind: "Formula", configuration: { expression } }] } } };
}
