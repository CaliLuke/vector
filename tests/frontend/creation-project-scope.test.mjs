import {
  createJoinedProjectFixture,
  assertJoinedMembershipRevoked,
  waitForJoinedProjectText,
  workspace,
} from "./helpers/joined-project-fixture.mjs";
import { registerHooks } from "node:module";
import assert from "node:assert/strict";
import { notifyManager } from "@tanstack/react-query";
import test from "node:test";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, next) {
    if (specifier === "@tanstack/react-router")
      return {
        shortCircuit: true,
        url:
          "data:text/javascript," + encodeURIComponent('export const useParams = () => ({ workspaceSlug: "acme" });'),
      };
    return next(specifier, context);
  },
});
const { EUserProjectRoles } = webRequire("@plane/types");
const { IssueModalProvider } = await import("../../apps/web/core/components/issues/issue-modal/provider.tsx");
const { IssueModalContext } =
  await import("../../apps/web/core/components/issues/issue-modal/context/issue-modal-context.tsx");
const { useIssueModalSource } =
  await import("../../apps/web/core/components/issues/issue-modal/use-issue-modal-source.ts");

function useDeterministicQueryNotifications(t) {
  notifyManager.setScheduler(queueMicrotask);
  t.after(() => notifyManager.setScheduler((callback) => setTimeout(callback, 0)));
}

function Probe() {
  const { defaultProjectId } = React.useContext(IssueModalContext);
  const source = useIssueModalSource({
    defaultProjectId,
    workspaceSlug: workspace.slug,
    isOpen: true,
    fetchIssueDetails: false,
    getProjectByIdentifier: () => undefined,
  });
  const joinedProjectIds = source.activeProjectId ? [source.activeProjectId] : [];
  return React.createElement("output", { "data-testid": "joined-projects" }, joinedProjectIds.join(","));
}

function createFixture(t, { allowedProjectIds, ...options } = {}) {
  return createJoinedProjectFixture(
    t,
    () => React.createElement(IssueModalProvider, { allowedProjectIds }, React.createElement(Probe)),
    options
  );
}

await test("creation defaults contain only visible joined projects in picker order", async (t) => {
  const fixture = await createFixture(t);
  await fixture.render();

  assert.equal(
    fixture.environment.container.querySelector("[data-testid=joined-projects]").textContent,
    "joined-first"
  );
});

await test("inactive workspace membership removes cached joined project access", async (t) => {
  await assertJoinedMembershipRevoked(t, await createFixture(t), "joined-first", false);
});

await test("workspace permission read errors remove cached joined project access", async (t) => {
  await assertJoinedMembershipRevoked(t, await createFixture(t), "joined-first", true);
});

await test("read-only project roles do not grant work-item creation", async (t) => {
  const fixture = await createFixture(t, {
    roles: { "joined-first": EUserProjectRoles.GUEST, "joined-later": EUserProjectRoles.MEMBER },
  });
  await fixture.render();
  assert.equal(
    fixture.environment.container.querySelector("[data-testid=joined-projects]").textContent,
    "joined-later"
  );
});

await test("creation default becomes available when project metadata arrives after permissions", async (t) => {
  useDeterministicQueryNotifications(t);
  const fixture = await createFixture(t, { includeProjectMetadata: false });
  await fixture.render();
  assert.equal(fixture.environment.container.querySelector("[data-testid=joined-projects]").textContent, "");
  await act(async () =>
    fixture.setProjectDetails([
      { id: "joined-first", workspace: workspace.id, archived_at: null, sort_order: 4 },
      { id: "joined-later", workspace: workspace.id, archived_at: null, sort_order: 8 },
    ])
  );
  await act(async () => waitForJoinedProjectText(fixture.environment.container, "joined-first"));
  assert.equal(
    fixture.environment.container.querySelector("[data-testid=joined-projects]").textContent,
    "joined-first"
  );
});

await test("creation defaults stay within the caller's allowed project subset", async (t) => {
  const fixture = await createFixture(t, { allowedProjectIds: ["removed", "archived", "joined-later"] });
  await fixture.render();
  assert.equal(
    fixture.environment.container.querySelector("[data-testid=joined-projects]").textContent,
    "joined-later"
  );
});
await test("an explicit empty allowed-project list has no automatic default", async (t) => {
  const fixture = await createFixture(t, { allowedProjectIds: [] });
  await fixture.render();
  assert.equal(fixture.environment.container.querySelector("[data-testid=joined-projects]").textContent, "");
});
