import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { identityKeys, advanceIdentitySession, getIdentitySessionGeneration } =
  await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { workspaceSearchOptions } = await import("../../apps/web/core/queries/workspace-search.ts");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const params = { search: "current field", workspace_search: true };
function fixture(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  const membershipKey = permissionKeys.workspaceMembership("alice", "acme");
  const rolesKey = permissionKeys.workspaceProjectRoles("alice", "acme", 15, "v1");
  client.setQueryData(membershipKey, { is_active: true, role: 15, updated_at: "v1" });
  client.setQueryData(rolesKey, { project: 15 });
  const scope = {
    accountId: "alice",
    workspaceSlug: "acme",
    membershipRole: 15,
    membershipRevision: "v1",
    generation: getIdentitySessionGeneration(client),
    roles: client.getQueryData(rolesKey),
    rolesUpdatedAt: client.getQueryState(rolesKey).dataUpdatedAt,
  };
  t.after(() => client.clear());
  return { client, scope, membershipKey, rolesKey };
}
await test("workspace search forwards scope, literal term and cancellation and keeps a plain-text match", async (t) => {
  const { client, scope } = fixture(t);
  const result = {
    results: {
      issue: [
        {
          id: "item",
          custom_property_match: { property_id: "field", property_title: "Details", text: "<img onerror=bad>" },
        },
      ],
    },
  };
  t.mock.method(WorkspaceService.prototype, "searchWorkspace", async (slug, query, signal) => {
    assert.equal(slug, "acme");
    assert.deepEqual(query, params);
    assert.ok(signal instanceof AbortSignal);
    return result;
  });
  assert.deepEqual(await client.fetchQuery(workspaceSearchOptions(scope, params)), result);
});
for (const change of ["membership", "roles", "roles-error", "roles-replaced", "session", "query", "cancel"]) {
  void test(`search rejects late snippets after ${change} changes`, async (t) => {
    const { client, scope, membershipKey, rolesKey } = fixture(t);
    const started = Promise.withResolvers(),
      response = Promise.withResolvers();
    let signal;
    t.mock.method(WorkspaceService.prototype, "searchWorkspace", (...args) => {
      signal = args.at(-1);
      started.resolve();
      return response.promise;
    });
    const options = workspaceSearchOptions(scope, params),
      read = client.fetchQuery(options);
    const rejected = assert.rejects(read);
    await started.promise;
    if (change === "membership") client.setQueryData(membershipKey, { is_active: false, role: 15, updated_at: "v2" });
    if (change === "roles") client.setQueryData(rolesKey, {});
    if (change === "roles-error") client.getQueryCache().find({ queryKey: rolesKey }).setState({ status: "error" });
    if (change === "roles-replaced") {
      client.removeQueries({ queryKey: rolesKey, exact: true });
      client.setQueryData(rolesKey, { project: 15 });
    }
    if (change === "session") advanceIdentitySession(client);
    if (change === "query") client.removeQueries({ queryKey: options.queryKey, exact: true });
    if (change === "cancel") {
      await client.cancelQueries({ queryKey: options.queryKey, exact: true });
      assert.equal(signal.aborted, true);
    }
    response.resolve({ results: { issue: [{ secret: "late content" }] } });
    await rejected;
    assert.equal(client.getQueryData(options.queryKey), undefined);
  });
}
