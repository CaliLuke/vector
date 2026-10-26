import { seedPermissionQueries } from "./permission-query-fixture.mjs";
import { React } from "./react.mjs";
import { createQueryRouterFixture } from "./query-router.mjs";

export const catalogueUser = { id: "current-user" };
export const catalogueSettings = {
  id: catalogueUser.id,
  workspace: { last_workspace_slug: "removed", fallback_workspace_slug: "fallback" },
};
export const removedWorkspace = {
  id: "removed-id",
  name: "Removed workspace",
  slug: "removed",
  organization_size: "2-10",
  timezone: "UTC",
};
export const fallbackWorkspace = {
  id: "fallback-id",
  name: "Fallback workspace",
  slug: "fallback",
  organization_size: "2-10",
  timezone: "UTC",
};

export async function renderWorkspaceCatalogueRoute(t, { path, loadComponent }) {
  const fixture = await createQueryRouterFixture(t, path);
  const component = await loadComponent();
  const { currentUserOptions, identityKeys } = await import("../../../apps/web/core/queries/identity.ts");
  const { workspaceKeys } = await import("../../../apps/web/core/queries/workspaces.ts");
  const { client, environment, router } = fixture;
  const defaultOptions = client.getDefaultOptions();
  client.setDefaultOptions({
    ...defaultOptions,
    mutations: { ...defaultOptions.mutations, gcTime: Infinity },
  });
  client.setQueryData(currentUserOptions().queryKey, catalogueUser);
  client.setQueryData(identityKeys.settings(catalogueUser.id), catalogueSettings);
  client.setQueryData(workspaceKeys.list(catalogueUser.id), [removedWorkspace, fallbackWorkspace]);
  await seedPermissionQueries(client, { userId: catalogueUser.id, workspaceSlug: removedWorkspace.slug });
  await seedPermissionQueries(client, { userId: catalogueUser.id, workspaceSlug: fallbackWorkspace.slug });
  await fixture.render(React.createElement(component));
  return { client, environment, router };
}
