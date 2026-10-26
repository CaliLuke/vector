import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createQueryRouterFixture } from "./helpers/query-router.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";

registerWebModuleHooks();
globalThis.pageTitleProjection = {
  scope: {
    accountId: "alice",
    workspaceSlug: "acme",
    sessionGeneration: 0,
    membershipRole: 20,
    membershipRevision: "revision-1",
    projectId: "project-1",
    projectMembershipRole: 20,
    projectRole: 20,
    rolesRevision: JSON.stringify([["project-1", 20]]),
  },
};
const stubs = {
  "@/components/pages/page-record-context": "export const usePageRecord=()=>globalThis.pageTitleProjection;",
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const { useRealtimePageEvents } = await import("../../apps/web/core/hooks/use-realtime-page-events.tsx");
const { pageKeys } = await import("../../apps/web/core/queries/pages.ts");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { ProjectPageService } = await import("../../apps/web/core/services/page/project-page.service.ts");

await test("collaborative title receipts update visible queries without echoing a REST save on locked pages", async (t) => {
  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-1/pages/page-1/");
  const scope = globalThis.pageTitleProjection.scope;
  fixture.client.setQueryData(identityKeys.currentUser(), { id: scope.accountId });
  await seedPermissionQueries(fixture.client, {
    userId: scope.accountId,
    workspaceSlug: scope.workspaceSlug,
    workspaceRole: scope.membershipRole,
    revision: scope.membershipRevision,
    projectRoles: { [scope.projectId]: scope.projectMembershipRole },
  });
  const page = { id: "page-1", project_ids: [scope.projectId], name: "Original", is_locked: true };
  const detailKey = pageKeys.detail(scope, page.id);
  const listKey = pageKeys.project(scope);
  fixture.client.setQueryData(detailKey, page);
  fixture.client.setQueryData(listKey, [page]);
  const writes = t.mock.method(ProjectPageService.prototype, "update", async () => {
    throw new Error("A collaborative receipt must not issue a second write");
  });
  function Harness() {
    const { updatePageProperties } = useRealtimePageEvents({
      page,
      getMemberDisplayName: () => undefined,
      handlers: { getRedirectionLink: () => "/acme/" },
    });
    return React.createElement(
      "button",
      {
        onClick: () => updatePageProperties(page.id, "property_updated", { name: "Collaborative title" }),
      },
      "Receive title"
    );
  }
  await fixture.render(React.createElement(Harness));
  t.mock.timers.enable({ apis: ["setTimeout"] });
  await act(async () => fixture.environment.container.querySelector("button").click());
  assert.equal(fixture.client.getQueryData(detailKey).name, "Collaborative title");
  assert.equal(fixture.client.getQueryData(listKey)[0].name, "Collaborative title");
  await act(async () => t.mock.timers.tick(2500));
  assert.equal(writes.mock.callCount(), 0);
  assert.equal(fixture.client.getQueryData(detailKey).name, "Collaborative title");
});
