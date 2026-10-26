import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./react-dom.mjs";
import { React, act } from "./react.mjs";
import { createRouterFixture } from "./router.mjs";

export async function settleQuery() {
  await act(async () => new Promise((resolve) => setTimeout(resolve, 20)));
}

export async function createQueryRouterFixture(t, path = "/acme/projects/p1/issues/") {
  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const router = createRouterFixture(path);
  t.after(async () => {
    await environment.render(null);
    await client.cancelQueries();
    client.clear();
    await environment.unmount();
  });
  return {
    environment,
    client,
    router: router.router,
    render: async (element) => {
      await router.render(environment, React.createElement(QueryClientProvider, { client }, element));
      await settleQuery();
    },
  };
}
