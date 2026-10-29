import assert from "node:assert/strict";
import test from "node:test";
import { QueryClientProvider } from "@tanstack/react-query";
import { createQueryDomFixture } from "./helpers/query-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { registerProjectCardModuleHooks } from "./helpers/project-card-module-hooks.mjs";

registerProjectCardModuleHooks({ React, state: {}, showModalContents: false });

// Load the shared module graph before a test owns DOM globals and mocks. A test
// timeout cannot cancel imports, so late setup must not overlap the next case.
const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
const { FavoriteService } = await import("../../apps/web/core/services/favorite/favorite.service.ts");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { EUserPermissions, EUserWorkspaceRoles } = await import("../../packages/types/dist/index.mjs");
const { ProjectCard } = await import("../../apps/web/core/components/project/card.tsx");

const project = { id: "project-id", name: "Go Plane", identifier: "GOPLANE", members: [], network: 2 };

async function renderProjectCard(t, projectRoles) {
  const { env: environment, client } = await createQueryDomFixture(t, {
    defaultOptions: { queries: { retry: false, gcTime: Infinity } },
  });
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
  t.after(() => assert.deepEqual(errors, []));
  const fixture = createRouterFixture("/default/projects/");
  t.mock.method(FavoriteService.prototype, "getFavorites", async () => []);
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", async () => []);
  client.setQueryData(currentUserOptions().queryKey, { id: "alice" });
  await seedPermissionQueries(client, {
    userId: "alice",
    workspaceSlug: "default",
    workspaceRole: EUserWorkspaceRoles.ADMIN,
    projectRoles,
  });
  await fixture.render(
    environment,
    React.createElement(QueryClientProvider, { client }, React.createElement(ProjectCard, { project }))
  );
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  return environment.container;
}

await test("workspace admin without project membership cannot favorite a project card", async (t) => {
  const container = await renderProjectCard(t, {});
  assert.equal(
    container.querySelectorAll('button[class*="bg-white/10"]').length,
    1,
    "only the copy-link action is visible"
  );
});

await test("a canonical project member can favorite the project card", async (t) => {
  const container = await renderProjectCard(t, { "project-id": EUserPermissions.MEMBER });
  assert.equal(
    container.querySelectorAll('button[class*="bg-white/10"]').length,
    2,
    "the member can see the favorite action"
  );
});
