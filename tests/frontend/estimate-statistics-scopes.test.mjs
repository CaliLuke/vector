import assert from "node:assert/strict";
import test from "node:test";
import { React, act, createPermissionUiFixture, settleQuery } from "./helpers/permission-ui-fixture.mjs";
const { useEstimateStatistics } = await import("../../apps/web/core/hooks/use-estimate-statistics.ts");
const { useCycleDetails } = await import("../../apps/web/core/hooks/use-cycles.ts");
const { CycleService } = await import("../../apps/web/core/services/cycle.service.ts");
const { estimateKeys } = await import("../../apps/web/core/queries/estimates.ts");
const { projectKeys } = await import("../../apps/web/core/queries/projects.ts");
const { cycleDetailOptions } = await import("../../apps/web/core/queries/cycle-modules.ts");
const { estimateStatisticsRevision } = await import("../../apps/web/core/queries/parent-statistics.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const scope = {
  kind: "project",
  accountId: "alice",
  sessionGeneration: 0,
  workspaceSlug: "acme",
  membershipRole: 15,
  membershipRevision: "2026-10-24T00:00:00Z",
  projectId: "p1",
  projectMembershipRole: 20,
  projectRole: 20,
};
const parentScope = { ...scope, workspaceRole: 15 };
const projectScope = { ...parentScope, projectRoles: [["p1", 20]] };
function catalogue(value, key = 1) {
  return [
    {
      id: "effort",
      project: "p1",
      workspace: "w1",
      type: "points",
      last_used: true,
      points: [{ id: "point", estimate: "effort", project: "p1", workspace: "w1", key, value }],
    },
  ];
}
function options(rows = catalogue("3"), revision = scope.membershipRevision) {
  return cycleDetailOptions(
    { ...parentScope, membershipRevision: revision },
    "c1",
    false,
    estimateStatisticsRevision("effort", "p1", rows)
  );
}
function Probe({ id }) {
  const result = useEstimateStatistics("cycle", "acme", "p1", "c1"),
    { cycle } = useCycleDetails("acme", "p1", "c1");
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(
      "output",
      { "data-consumer": id },
      result.statistics.isError ? "error" : result.statistics.isPending ? "pending" : "ready",
      ":",
      cycle?.total_estimate_points ?? "none"
    ),
    React.createElement("button", { onClick: () => void result.statistics.retry() }, "Retry statistics")
  );
}
async function fixture(t, rows = catalogue("3"), total = 3) {
  const view = await createPermissionUiFixture(t, { workspaceRole: 15, projectRoles: { p1: 20 } });
  view.client.setDefaultOptions({ queries: { retry: false, gcTime: Infinity, staleTime: Infinity } });
  view.client.setQueryData(projectKeys.detail(projectScope), { id: "p1", estimate: "effort" });
  view.client.setQueryData(estimateKeys.list(scope), rows);
  view.client.setQueryData(options(rows).queryKey, { id: "c1", project_id: "p1", total_estimate_points: total });
  const reads = [];
  t.mock.method(CycleService.prototype, "retrieve", () => {
    const read = Promise.withResolvers();
    reads.push(read);
    return read.promise;
  });
  t.after(() => {
    for (const read of reads) read.resolve({ id: "c1", project_id: "p1", total_estimate_points: 8 });
  });
  return {
    ...view,
    reads,
    render: (ids) =>
      view.render(
        React.createElement(React.Fragment, null, ...ids.map((id) => React.createElement(Probe, { key: id, id })))
      ),
    outputs: () => [...view.environment.container.querySelectorAll("output")].map((node) => node.textContent),
  };
}
await test("permission revision requires a native canonical parent read before changed totals are ready", async (t) => {
  const view = await fixture(t);
  await view.render(["one"]);
  await act(async () => {
    view.client.setQueryData(permissionKeys.workspaceMembership("alice", "acme"), {
      is_active: true,
      role: 15,
      updated_at: "revision-2",
    });
    view.client.setQueryData(permissionKeys.workspaceProjectRoles("alice", "acme", 15, "revision-2"), { p1: 20 });
    view.client.setQueryData(projectKeys.detail({ ...projectScope, membershipRevision: "revision-2" }), {
      id: "p1",
      estimate: "effort",
    });
    view.client.setQueryData(estimateKeys.list({ ...scope, membershipRevision: "revision-2" }), catalogue("8"));
  });
  await settleQuery();
  assert.equal(view.reads.length, 1);
  assert.deepEqual(view.outputs(), ["pending:none"]);
  await act(async () => view.reads[0].resolve({ id: "c1", project_id: "p1", total_estimate_points: 8 }));
  await settleQuery();
  assert.deepEqual(view.outputs(), ["ready:8"]);
});
await test("two statistics consumers share one Query read; key-only changes and unmount do not restart it", async (t) => {
  const view = await fixture(t);
  await view.render(["one", "two"]);
  await act(async () => view.client.setQueryData(estimateKeys.list(scope), catalogue("8")));
  await settleQuery();
  assert.equal(view.reads.length, 1);
  await act(async () => view.client.setQueryData(estimateKeys.list(scope), catalogue("8", 2)));
  await view.render(["two"]);
  assert.equal(view.reads.length, 1);
  await act(async () => view.reads[0].resolve({ id: "c1", project_id: "p1", total_estimate_points: 8 }));
  await settleQuery();
  assert.deepEqual(view.outputs(), ["ready:8"]);
});
await test("deleting an inactive numeric system refreshes totals while the canonical active pointer stays unchanged", async (t) => {
  const inactive = {
    id: "inactive",
    project: "p1",
    workspace: "w1",
    type: "points",
    last_used: false,
    points: [{ id: "inactive-point", estimate: "inactive", project: "p1", workspace: "w1", key: 1, value: "8" }],
  };
  const view = await fixture(t, [...catalogue("3"), inactive], 11);
  await view.render(["one"]);
  await act(async () => view.client.setQueryData(estimateKeys.list(scope), catalogue("3")));
  await settleQuery();
  assert.equal(view.client.getQueryData(projectKeys.detail(projectScope)).estimate, "effort");
  assert.equal(view.reads.length, 1);
  assert.deepEqual(view.outputs(), ["pending:none"]);
  await act(async () => view.reads[0].resolve({ id: "c1", project_id: "p1", total_estimate_points: 3 }));
  await settleQuery();
  assert.deepEqual(view.outputs(), ["ready:3"]);
});
await test("retired native transport failure cannot replace a newer successful canonical read", async (t) => {
  const view = await fixture(t);
  await view.render(["one", "two"]);
  await act(async () => view.client.setQueryData(estimateKeys.list(scope), catalogue("8")));
  await settleQuery();
  assert.equal(view.reads.length, 1);
  const current = options(catalogue("8"));
  let replacement;
  await act(async () => {
    await view.client.cancelQueries({ queryKey: current.queryKey, exact: true });
    replacement = view.client.fetchQuery({ ...current, staleTime: 0 });
  });
  assert.equal(view.reads.length, 2);
  await act(async () => {
    view.reads[1].resolve({ id: "c1", project_id: "p1", total_estimate_points: 8 });
    await replacement;
  });
  await act(async () => view.reads[0].reject(new Error("obsolete transport failure")));
  await settleQuery();
  assert.deepEqual(view.outputs(), ["ready:8", "ready:8"]);
});

await test("failed numeric refresh exposes Retry and a fresh canonical native read restores readiness", async (t) => {
  const view = await fixture(t);
  await view.render(["one"]);
  await act(async () => view.client.setQueryData(estimateKeys.list(scope), catalogue("8")));
  await settleQuery();
  await act(async () => view.reads[0].reject(new Error("statistics offline")));
  await settleQuery();
  assert.deepEqual(view.outputs(), ["error:none"]);
  await act(async () => view.environment.container.querySelector("button").click());
  await settleQuery();
  assert.equal(view.reads.length, 2);
  await act(async () => view.reads[1].resolve({ id: "c1", project_id: "p1", total_estimate_points: 8 }));
  await settleQuery();
  assert.deepEqual(view.outputs(), ["ready:8"]);
});
