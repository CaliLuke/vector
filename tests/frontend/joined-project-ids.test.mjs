import {
  createJoinedProjectFixture,
  assertJoinedMembershipRevoked,
  workspace,
} from "./helpers/joined-project-fixture.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { useJoinedProjectIds } = await import("../../apps/web/core/hooks/use-joined-project-ids.ts");

function Probe({ workspaceSlug }) {
  const joinedProjectIds = useJoinedProjectIds(workspaceSlug);
  return React.createElement("output", { "data-testid": "joined-projects" }, joinedProjectIds.join(","));
}

function createFixture(t, options) {
  return createJoinedProjectFixture(t, () => React.createElement(Probe, { workspaceSlug: workspace.slug }), options);
}

await test("joined project IDs use role-query membership and preserve metadata ordering and scope", async (t) => {
  const fixture = await createFixture(t);
  await fixture.render();

  assert.equal(
    fixture.environment.container.querySelector("[data-testid=joined-projects]").textContent,
    "joined-first,joined-later"
  );
});

await test("inactive workspace membership removes cached joined project access", async (t) => {
  await assertJoinedMembershipRevoked(t, await createFixture(t), "joined-first,joined-later", false);
});

await test("workspace permission read errors remove cached joined project access", async (t) => {
  await assertJoinedMembershipRevoked(t, await createFixture(t), "joined-first,joined-later", true);
});
