import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { webRequire } = await import("./helpers/react.mjs");
const { QueryClient, MutationObserver } = webRequire("@tanstack/react-query");
const { EstimateService } = await import("../../apps/web/core/services/estimate.service.ts");
const { estimateKeys, estimateListOptions, estimateMutationOptions, activeProjectEstimate, orderedEstimatePoints } =
  await import("../../apps/web/core/queries/estimates.ts");
const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const scope = {
  kind: "project",
  accountId: "alice",
  sessionGeneration: 0,
  workspaceSlug: "acme",
  membershipRole: 15,
  membershipRevision: "m1",
  projectId: "p1",
  projectMembershipRole: 20,
  projectRole: 20,
};
const estimate = {
  id: "e1",
  project: "p1",
  workspace: "w1",
  name: "Effort",
  type: "POINTS",
  last_used: true,
  points: [
    { id: "high", estimate: "e1", project: "p1", key: 2, value: "8" },
    { id: "low", estimate: "e1", project: "p1", key: 1, value: "3" },
  ],
};
function clientFixture(t, activeScope = scope) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  client.setQueryData(identityKeys.currentUser(), { id: activeScope.accountId });
  const membershipKey = permissionKeys.workspaceMembership(activeScope.accountId, activeScope.workspaceSlug);
  const rolesKey = permissionKeys.workspaceProjectRoles(
    activeScope.accountId,
    activeScope.workspaceSlug,
    activeScope.membershipRole,
    activeScope.membershipRevision
  );
  client.setQueryData(membershipKey, {
    is_active: true,
    role: activeScope.membershipRole,
    updated_at: activeScope.membershipRevision,
  });
  client.setQueryData(rolesKey, { [activeScope.projectId]: activeScope.projectMembershipRole });
  t.after(() => client.clear());
  return { client, membershipKey, rolesKey };
}
function deferred() {
  let resolve;
  const promise = new Promise((yes) => {
    resolve = yes;
  });
  return { promise, resolve };
}

await test("native project query normalizes plain records, orders points without mutation, and accepts canonical empty", async (t) => {
  const { client } = clientFixture(t);
  let rows = [estimate];
  t.mock.method(EstimateService.prototype, "fetchProjectEstimates", async () => rows);
  const result = await client.fetchQuery(estimateListOptions(scope));
  assert.equal(result[0].type, "points");
  assert.deepEqual(
    orderedEstimatePoints(result[0]).map((point) => point.id),
    ["low", "high"]
  );
  assert.deepEqual(
    result[0].points.map((point) => point.id),
    ["high", "low"]
  );
  assert.equal(activeProjectEstimate(result, "p1", "e1"), result[0]);
  rows = [];
  await client.fetchQuery({ ...estimateListOptions(scope), staleTime: 0 });
  assert.deepEqual(client.getQueryData(estimateKeys.list(scope)), []);
});

await test("late project response is rejected after permission query replacement", async (t) => {
  const { client, rolesKey } = clientFixture(t);
  const read = deferred();
  t.mock.method(EstimateService.prototype, "fetchProjectEstimates", () => read.promise);
  const pending = client.fetchQuery(estimateListOptions(scope));
  client.removeQueries({ queryKey: rolesKey, exact: true });
  client.setQueryData(rolesKey, { p1: 20 });
  read.resolve([estimate]);
  await assert.rejects(pending, /access changed/);
  assert.equal(client.getQueryData(estimateKeys.list(scope)), undefined);
});

await test("cached identity error prevents new reads and writes while an accepted write still reconciles", async (t) => {
  const { client } = clientFixture(t);
  client.setQueryData(estimateKeys.list(scope), []);
  const write = deferred();
  let calls = 0;
  t.mock.method(EstimateService.prototype, "createEstimate", () => {
    calls++;
    return write.promise;
  });
  const observer = new MutationObserver(client, estimateMutationOptions(client, scope));
  const accepted = observer.mutate({ type: "create", data: { estimate_points: [] } });
  await new Promise((resolve) => setImmediate(resolve));
  const identity = client.getQueryCache().find({ queryKey: identityKeys.currentUser(), exact: true });
  identity.setState({ status: "error", error: new Error("verification unavailable") });
  write.resolve(estimate);
  await accepted;
  assert.equal(client.getQueryData(estimateKeys.list(scope))[0].id, "e1");
  await assert.rejects(observer.mutate({ type: "create", data: { estimate_points: [] } }), /member access/);
  await assert.rejects(client.fetchQuery({ ...estimateListOptions(scope), staleTime: 0 }), /access changed/);
  assert.equal(calls, 1);
});

await test("queued mutation rechecks session before transport and cannot populate replacement account", async (t) => {
  const { client } = clientFixture(t);
  client.setQueryData(estimateKeys.list(scope), []);
  const write = deferred();
  let calls = 0;
  t.mock.method(EstimateService.prototype, "createEstimate", () => {
    calls++;
    return write.promise;
  });
  const options = estimateMutationOptions(client, scope);
  const a = new MutationObserver(client, options).mutate({ type: "create", data: { estimate_points: [] } });
  const b = new MutationObserver(client, options).mutate({ type: "create", data: { estimate_points: [] } });
  const rejected = assert.rejects(b, /member access/);
  await new Promise((resolve) => setImmediate(resolve));
  advanceIdentitySession(client);
  client.setQueryData(identityKeys.currentUser(), { id: "bob" });
  write.resolve(estimate);
  await a;
  await rejected;
  assert.equal(calls, 1);
  assert.deepEqual(client.getQueryData(estimateKeys.list(scope)), []);
});

await Promise.all(
  [401, 403, 404].map((status) =>
    test(`accepted estimate write cannot reconcile a cached identity authorization error ${status}`, async (t) => {
      const { client } = clientFixture(t);
      client.setQueryData(estimateKeys.list(scope), []);
      const write = deferred();
      t.mock.method(EstimateService.prototype, "createEstimate", () => write.promise);
      const observer = new MutationObserver(client, estimateMutationOptions(client, scope));
      const saving = observer.mutate({ type: "create", data: { estimate_points: [] } });
      await new Promise((resolve) => setImmediate(resolve));
      client
        .getQueryCache()
        .find({ queryKey: identityKeys.currentUser(), exact: true })
        .setState({ status: "error", error: { response: { status } } });
      write.resolve(estimate);
      const result = await saving;
      assert.equal(result.captured.isCurrent(), false);
      assert.deepEqual(client.getQueryData(estimateKeys.list(scope)), []);
    })
  )
);

await test("workspace admin effective project access is verified independently of a guest membership identity", async (t) => {
  const admin = { ...scope, membershipRole: 20, projectMembershipRole: 5, projectRole: 20 };
  const { client } = clientFixture(t, admin);
  client.setQueryData(estimateKeys.list(admin), []);
  t.mock.method(EstimateService.prototype, "createEstimate", async () => estimate);
  await new MutationObserver(client, estimateMutationOptions(client, admin)).mutate({
    type: "create",
    data: { estimate_points: [] },
  });
  assert.equal(client.getQueryData(estimateKeys.list(admin))[0].id, "e1");
});

await test("canonical create and project selection responses cannot cross project or selected-ID scope", async (t) => {
  const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");
  const { client } = clientFixture(t);
  client.setQueryData(estimateKeys.list(scope), []);
  t.mock.method(EstimateService.prototype, "createEstimate", async () => ({ ...estimate, project: "p2" }));
  const options = estimateMutationOptions(client, scope);
  await assert.rejects(
    new MutationObserver(client, options).mutate({ type: "create", data: { estimate_points: [] } }),
    /outside the requested project/
  );
  assert.deepEqual(client.getQueryData(estimateKeys.list(scope)), []);
  t.mock.method(ProjectService.prototype, "updateProject", async () => ({ id: "p2", estimate: "e1" }));
  await assert.rejects(
    new MutationObserver(client, options).mutate({ type: "select", estimateId: "e1", expectedId: null }),
    /different project estimate selection/
  );
  assert.deepEqual(client.getQueryData(estimateKeys.list(scope)), []);
});
