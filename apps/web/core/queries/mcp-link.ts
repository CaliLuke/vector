import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { mcpLinkService } from "@/services/mcp-link.service";
import { captureIdentityScope, isIdentityScopeCurrent, isIdentityScopeVerified } from "./identity";

export const mcpLinkKey = (userId: string | undefined) => ["identity", userId, "mcp-link"] as const;

export function mcpLinkOptions(userId: string | undefined) {
  return queryOptions({
    queryKey: mcpLinkKey(userId),
    retry: false,
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
    queryFn: userId
      ? async ({ client, signal }) => {
          const scope = captureIdentityScope(client, userId);
          if (!isIdentityScopeVerified(client, scope)) throw new Error("Sign in again to manage MCP access.");
          const status = await mcpLinkService.status(signal);
          if (!isIdentityScopeVerified(client, scope)) throw new Error("The account session changed.");
          return status;
        }
      : skipToken,
  });
}

export function mcpLinkChangeOptions<T>(
  client: QueryClient,
  userId: string | undefined,
  operation: (input: T, assertCurrent: () => void) => Promise<void>
) {
  const scope = captureIdentityScope(client, userId);
  const query = client.getQueryCache().find({ queryKey: mcpLinkKey(userId), exact: true });
  const assertCurrent = () => {
    if (
      !isIdentityScopeVerified(client, scope) ||
      !query ||
      client.getQueryCache().find({ queryKey: mcpLinkKey(userId), exact: true }) !== query
    )
      throw new Error("The account session changed.");
  };
  return mutationOptions({
    scope: { id: `identity/${userId}/mcp-link` },
    retry: false,
    gcTime: 0,
    mutationFn: async (input: T) => {
      if (
        !isIdentityScopeVerified(client, scope) ||
        !query ||
        query.state.status !== "success" ||
        query.state.fetchStatus !== "idle"
      ) {
        throw new Error("Reload your connection status before continuing.");
      }
      assertCurrent();
      await operation(input, assertCurrent);
    },
    onSettled: async () => {
      if (
        !isIdentityScopeCurrent(client, scope) ||
        client.getQueryCache().find({ queryKey: mcpLinkKey(userId), exact: true }) !== query
      )
        return;
      await client.invalidateQueries({ predicate: (candidate) => candidate === query });
    },
  });
}
