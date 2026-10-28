import assert from "node:assert/strict";
import test from "node:test";
import { React, act, createPermissionUiFixture, settleQuery } from "./helpers/permission-ui-fixture.mjs";

const { useCycleDetails } = await import("../../apps/web/core/hooks/use-cycles.ts");
const { useModuleDetails } = await import("../../apps/web/core/hooks/use-modules.ts");
const { CycleService } = await import("../../apps/web/core/services/cycle.service.ts");
const { ModuleService } = await import("../../apps/web/core/services/module.service.ts");
const { EstimateService } = await import("../../apps/web/core/services/estimate.service.ts");
const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");
const { UserService } = await import("../../apps/web/core/services/user.service.ts");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");

async function fixture(t, useDetails, failedRead) {
  const view = await createPermissionUiFixture(t, { workspaceRole: 15, projectRoles: { p1: 20 } });
  let failing = true;
  let details;
  const record = { id: "parent", project_id: "p1", name: "Loaded", total_estimate_points: 3 };
  const readParent = t.mock.fn(async () => record);
  t.mock.method(CycleService.prototype, "retrieve", readParent);
  t.mock.method(ModuleService.prototype, "getModuleDetails", readParent);
  t.mock.method(UserService.prototype, "currentUser", async () => ({ id: "alice" }));
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => ({
    is_active: true,
    role: 15,
    updated_at: "2026-10-24T00:00:00Z",
  }));
  t.mock.method(ProjectService.prototype, "getProject", async () => {
    if (failing && failedRead === "project") throw new Error("Project unavailable");
    return { id: "p1", estimate: "effort" };
  });
  t.mock.method(EstimateService.prototype, "fetchProjectEstimates", async () => {
    if (failing && failedRead === "estimates") throw new Error("Estimates unavailable");
    return [
      {
        id: "effort",
        project: "p1",
        workspace: "w1",
        type: "points",
        last_used: true,
        points: [{ id: "point", estimate: "effort", project: "p1", workspace: "w1", key: 1, value: "3" }],
      },
    ];
  });
  function Probe() {
    const value = useDetails("acme", "p1", "parent");
    React.useLayoutEffect(() => {
      details = value;
    }, [value]);
    return React.createElement("output", null, value.isError ? "error" : value.isPending ? "pending" : "ready");
  }
  await view.render(React.createElement(Probe));
  return {
    ...view,
    readParent,
    details: () => details,
    status: () => view.environment.container.querySelector("output").textContent,
    recover: async () => {
      failing = false;
      await act(async () => details.retry());
      await settleQuery();
    },
  };
}

for (const { domain, useDetails } of [
  { domain: "cycle", useDetails: useCycleDetails },
  { domain: "module", useDetails: useModuleDetails },
]) {
  for (const prerequisite of ["estimates", "project"]) {
    void test(`${domain} detail exposes a failed ${prerequisite} prerequisite and recovers through its own Retry`, async (t) => {
      const view = await fixture(t, useDetails, prerequisite);
      assert.equal(view.status(), "error");
      assert.equal(view.details().isPending, false);
      assert.equal(view.readParent.mock.callCount(), 0, "do not request statistics without their revision");
      await view.recover();
      assert.equal(view.status(), "ready");
      assert.equal(view.details()[domain].name, "Loaded");
      assert.equal(view.details().estimates.points[0].value, "3");
      assert.equal(view.readParent.mock.callCount(), 1, "only fetch the recovered detail revision");
    });
  }
}
