import type { QueryClient } from "@tanstack/react-query";
import type { TProjectRosterScope } from "./member-rosters";
import { captureProjectAccessScope } from "./project-access-scope";

function captureItemScope(
  client: QueryClient,
  scope: TProjectRosterScope,
  queryKey: readonly unknown[],
  verified: boolean
) {
  const access = captureProjectAccessScope(client, scope);
  const query = client.getQueryCache().find({ queryKey, exact: true });
  return () =>
    Boolean(
      query &&
      client.getQueryCache().find({ queryKey, exact: true }) === query &&
      (verified ? access.isVerified() : access.isCurrent())
    );
}

export function captureItemReadScope(client: QueryClient, scope: TProjectRosterScope, queryKey: readonly unknown[]) {
  return captureItemScope(client, scope, queryKey, true);
}

/** Keep acknowledged item writes reconcilable during a transient identity refresh error. */
export function captureItemLineageScope(client: QueryClient, scope: TProjectRosterScope, queryKey: readonly unknown[]) {
  return captureItemScope(client, scope, queryKey, false);
}
