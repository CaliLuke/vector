import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedIssueReadAccess } from "./helpers/issue-query-fixtures.mjs";
registerWebModuleHooks();
import { QueryClient, QueryObserver, MutationObserver } from "@tanstack/react-query";
const { captureParentStatistics } = await import("../../apps/web/core/queries/parent-statistics.ts");
const { currentParentCatalogueScope } = await import("../../apps/web/core/queries/cycle-module-scope.ts");
const { cycleDetailOptions, moduleDetailOptions } = await import("../../apps/web/core/queries/cycle-modules.ts");
const { CycleService } = await import("../../apps/web/core/services/cycle.service.ts");
const { ModuleService } = await import("../../apps/web/core/services/module.service.ts");
const { IssueService } = await import("../../apps/web/core/services/issue/issue.service.ts");
const { currentIssueListMutationOwner, issueListMutationOptions } =
  await import("../../apps/web/core/queries/issue-list-mutations.ts");
const { projectIssueIdsByDisplayOrder, projectGroupedIssueIdsByDisplayOrder } =
  await import("../../apps/web/core/components/issues/issue-layouts/issue-display-ordering.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const scope = {
  workspaceSlug: "acme",
  scopeKey: "alice/acme/p1",
  pointsByProject: {
    p1: {
      small: { id: "small", project: "p1", estimate: "e1", key: 1, value: "3" },
      large: { id: "large", project: "p1", estimate: "e1", key: 2, value: "8" },
    },
  },
  isCurrent: () => true,
  isReadCurrent: () => true,
};
const base = {
  project_id: "p1",
  state_id: "started",
  assignee_ids: [],
  label_ids: [],
  module_ids: ["m1"],
  cycle_id: "c1",
  created_at: "2026-10-01T00:00:00Z",
  completed_at: null,
};
function deferred() {
  let resolve;
  const promise = new Promise((yes) => {
    resolve = yes;
  });
  return { promise, resolve };
}
function queryFixture(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  seedIssueReadAccess(client);
  t.after(() => client.clear());
  return client;
}
await test("estimate order is project-scoped, preserves empty-last and reacts to immutable point-key replacement", () => {
  const issues = {
    a: { ...base, id: "a", estimate_point: "large" },
    b: { ...base, id: "b", estimate_point: "small" },
    empty: { ...base, id: "empty", estimate_point: null },
    foreign: { ...base, id: "foreign", project_id: "p2", estimate_point: "large" },
  };
  assert.deepEqual(
    projectIssueIdsByDisplayOrder(
      ["a", "b", "empty"],
      (id) => issues[id],
      undefined,
      "estimate_point__key",
      undefined,
      scope
    ),
    ["b", "a", "empty"]
  );
  assert.deepEqual(
    projectIssueIdsByDisplayOrder(
      ["empty", "a", "b"],
      (id) => issues[id],
      undefined,
      "-estimate_point__key",
      undefined,
      scope
    ),
    ["a", "b", "empty"]
  );
  const next = {
    ...scope,
    pointsByProject: { p1: { ...scope.pointsByProject.p1, large: { ...scope.pointsByProject.p1.large, key: 0 } } },
  };
  assert.deepEqual(
    projectGroupedIssueIdsByDisplayOrder(
      { group: { sub: ["b", "a"] } },
      (id) => issues[id],
      undefined,
      "estimate_point__key",
      undefined,
      next
    ),
    { group: { sub: ["a", "b"] } }
  );
  assert.deepEqual(
    projectIssueIdsByDisplayOrder(["b", "a"], (id) => issues[id], undefined, "estimate_point__key", undefined, {
      ...scope,
      isCurrent: () => false,
    }),
    ["b", "a"]
  );
  assert.deepEqual(
    projectIssueIdsByDisplayOrder(
      ["foreign", "b"],
      (id) => issues[id],
      undefined,
      "estimate_point__key",
      undefined,
      scope
    ),
    ["foreign", "b"]
  );
});
async function testParentStatistics(domain) {
  function parentFixture(t) {
    const client = queryFixture(t);
    const id = domain === "cycle" ? "c1" : "m1";
    const access = currentParentCatalogueScope(client, "acme", "p1").project;
    const options = (domain === "cycle" ? cycleDetailOptions : moduleDetailOptions)(access, id);
    client.setQueryData(options.queryKey, {
      id,
      project_id: "p1",
      total_issues: 1,
      total_estimate_points: 3,
      started_estimate_points: 3,
    });
    return {
      client,
      id,
      access,
      options,
      effects: captureParentStatistics(client, domain, access, id, options.queryKey),
      service: domain === "cycle" ? CycleService.prototype : ModuleService.prototype,
      method: domain === "cycle" ? "retrieve" : "getModuleDetails",
      parent: () => client.getQueryData(options.queryKey),
    };
  }
  await test(`${domain} accepted estimate PATCH refreshes canonical statistics without requiring an estimate catalogue`, async (t) => {
    const { client, id, access, effects, options, service, method, parent } = parentFixture(t);
    const pending = deferred();
    t.mock.method(IssueService.prototype, "patchIssue", () => pending.promise);
    t.mock.method(service, method, async () => ({
      id,
      project_id: "p1",
      total_issues: 1,
      total_estimate_points: 8,
      started_estimate_points: 8,
    }));
    const observer = new QueryObserver(client, { ...options, staleTime: Infinity });
    const unsubscribe = observer.subscribe(() => {});
    t.after(unsubscribe);
    const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true, effects);
    assert.equal(owner.scope.projectId, access.projectId);
    const saving = new MutationObserver(client, issueListMutationOptions(client)).mutate({
      owner,
      mutation: { kind: "patch", issueId: "i1", data: { estimate_point: "large" } },
    });
    const canonical = { ...base, id: "i1", estimate_point: "large" };
    pending.resolve(canonical);
    assert.deepEqual(await saving, canonical);
    await new Promise((resolve) => setImmediate(resolve));
    assert.equal(parent().total_estimate_points, 8);
    assert.equal(parent().started_estimate_points, 8);
  });
  await test(`${domain} canonical detail response cannot commit after permission or identity scope replacement`, async (t) => {
    const { client, id, effects, options, service, method, parent } = parentFixture(t);
    const read = deferred();
    let calls = 0;
    t.mock.method(service, method, () => {
      calls++;
      return read.promise;
    });
    const observer = new QueryObserver(client, options);
    const unsubscribe = observer.subscribe(() => {});
    t.after(unsubscribe);
    assert.equal(calls, 1);
    client
      .getQueryCache()
      .find({ queryKey: permissionKeys.workspaceMembership("alice", "acme"), exact: true })
      .setState({ status: "error", error: { status: 403 } });
    read.resolve({ id, project_id: "p1", total_estimate_points: 99 });
    await new Promise((resolve) => setImmediate(resolve));
    assert.equal(parent().total_estimate_points, 3);
    assert.equal(observer.getCurrentResult().isError, true);
    client.setQueryData(identityKeys.currentUser(), { id: "bob" });
    await effects.refresh();
    assert.equal(calls, 1);
  });
  await test(`${domain} parent detail cancellation rejects an older response after a newer read commits`, async (t) => {
    const { client, id, options, service, method, parent } = parentFixture(t);
    const a = deferred(),
      b = deferred(),
      signals = [];
    t.mock.method(service, method, (...args) => {
      signals.push(args.at(-1));
      return signals.length === 1 ? a.promise : b.promise;
    });
    const older = client.fetchQuery({ ...options, staleTime: 0 }).catch(() => undefined);
    await client.cancelQueries({ queryKey: options.queryKey, exact: true });
    assert.equal(signals[0].aborted, true);
    const newer = client.fetchQuery({ ...options, staleTime: 0 });
    b.resolve({ id, project_id: "p1", total_estimate_points: 8 });
    await newer;
    a.resolve({ id, project_id: "p1", total_estimate_points: 3 });
    await older;
    assert.equal(parent().total_estimate_points, 8);
  });
}

await testParentStatistics("cycle");
await testParentStatistics("module");
