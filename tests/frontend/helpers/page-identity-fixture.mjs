import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { React } from "./react.mjs";

export async function createPageIdentityFixture(t) {
  const { currentUserOptions } = await import("../../../apps/web/core/queries/identity.ts");
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  client.setQueryData(currentUserOptions().queryKey, { id: "owner" });
  t.after(() => client.clear());

  return {
    client,
    wrap(element) {
      return React.createElement(QueryClientProvider, { client }, element);
    },
    async render(environment, element) {
      await environment.render(React.createElement(QueryClientProvider, { client }, element));
    },
  };
}
