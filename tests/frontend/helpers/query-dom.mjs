import { seedPermissionQueries } from "./permission-query-fixture.mjs";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./react-dom.mjs";
import { React } from "./react.mjs";

/** Query observers unmount before cancellation and cache teardown. */
export async function createQueryDomFixture(t, options = { defaultOptions: { queries: { retry: false } } }) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient(options);
  t.after(async () => {
    await env.unmount();
    await client.cancelQueries();
    client.clear();
  });
  return {
    env,
    client,
    render: (element) => env.render(React.createElement(QueryClientProvider, { client }, element)),
  };
}

/** The common member scope used by item/catalog observer tests. */
export async function createProjectQueryDomFixture(t, projectRole) {
  const fixture = await createQueryDomFixture(t);
  const { identityKeys } = await import("../../../apps/web/core/queries/identity.ts");
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  await seedPermissionQueries(fixture.client, {
    userId: "alice",
    workspaceSlug: "acme",
    workspaceRole: 15,
    projectRoles: { p: projectRole },
    revision: "v1",
  });
  return fixture;
}
