import { registerHooks } from "node:module";
import { React, act } from "./react.mjs";
import { createQueryRouterFixture, settleQuery } from "./query-router.mjs";
import { resolvePackageStub } from "./module-stubs.mjs";
import { registerWebModuleHooks } from "./web-module-hooks.mjs";
import { seedPermissionQueries } from "./permission-query-fixture.mjs";

registerWebModuleHooks();

export { React, act, settleQuery };

export function registerPermissionUiStubs(parentSuffix, stubs) {
  registerHooks({
    resolve(specifier, context, nextResolve) {
      if (context.parentURL?.endsWith(parentSuffix)) return resolvePackageStub(specifier, stubs, context, nextResolve);
      return nextResolve(specifier, context);
    },
  });
}

export async function createPermissionUiFixture(
  t,
  { path, userId = "alice", workspaceSlug = "acme", workspaceRole, projectRoles = {}, rolesRead }
) {
  const fixture = await createQueryRouterFixture(t, path);
  const { identityKeys } = await import("../../../apps/web/core/queries/identity.ts");
  const { WorkspaceService } = await import("../../../apps/web/core/services/workspace.service.ts");
  fixture.client.setQueryData(identityKeys.currentUser(), { id: userId });
  await seedPermissionQueries(fixture.client, { userId, workspaceSlug, workspaceRole, projectRoles });
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => ({
    is_active: true,
    role: workspaceRole,
    updated_at: "revision-1",
  }));
  if (rolesRead) t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", rolesRead);
  else t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => projectRoles);
  return fixture;
}
