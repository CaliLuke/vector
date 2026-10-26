/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { EIssueServiceType } from "@plane/types";
import type { TIssueServiceType } from "@plane/types";
import { captureItemReadScope } from "./work-item-read-scope";
import type { TProjectRosterScope } from "./member-rosters";
import { getIdentitySessionGeneration } from "./identity";
import { isAuthorizationError } from "./authorization-errors";

export type ItemAccessScope = {
  accountId: string | undefined;
  generation: number;
  workspaceSlug: string | undefined;
  projectId: string | undefined;
  issueId: string | undefined;
  serviceType?: TIssueServiceType;
};
export function itemAccessOptions(scope: ItemAccessScope) {
  return queryOptions({
    queryKey: [
      "workspace",
      scope.workspaceSlug,
      "item-access",
      scope.accountId,
      scope.generation,
      scope.projectId,
      scope.issueId,
      scope.serviceType ?? EIssueServiceType.ISSUES,
    ],
    queryFn: skipToken,
    initialData: false,
    staleTime: Infinity,
    gcTime: Infinity,
  });
}
export function isItemAccessDenied(client: QueryClient, scope: ItemAccessScope) {
  return client.getQueryData(itemAccessOptions(scope).queryKey) === true;
}
function matchesOwner(
  scope: ItemAccessScope,
  account: unknown,
  generation: unknown,
  project: unknown,
  item: unknown,
  service: unknown = EIssueServiceType.ISSUES
) {
  return (
    account === scope.accountId &&
    generation === scope.generation &&
    project === scope.projectId &&
    item === scope.issueId &&
    service === (scope.serviceType ?? EIssueServiceType.ISSUES)
  );
}
function matchesSchema(scope: ItemAccessScope, key: readonly unknown[]) {
  const input = key[4];
  return (
    !!input &&
    typeof input === "object" &&
    "issue_id" in input &&
    "project_id" in input &&
    matchesOwner(scope, key[3], key[10], input.project_id, input.issue_id)
  );
}
function matchesPreview(scope: ItemAccessScope, key: readonly unknown[]) {
  const input = key[3];
  return (
    !!input &&
    typeof input === "object" &&
    "accountId" in input &&
    "generation" in input &&
    "projectId" in input &&
    "issueId" in input &&
    matchesOwner(scope, input.accountId, input.generation, input.projectId, input.issueId)
  );
}
function matchesWorkspaceRead(scope: ItemAccessScope, key: readonly unknown[]) {
  switch (key[2]) {
    case "property-history":
    case "item-edit-source":
      return matchesOwner(scope, key[3], key[4], key[5], key[6]);
    case "work-item-schema":
      return matchesSchema(scope, key);
    case "inline-property-preview":
      return matchesPreview(scope, key);
    case "issue-discussion":
      return matchesOwner(scope, key[3], key[4], key[5], key[6], key[7]);
    default:
      return false;
  }
}
/** Purge only this confirmed actor/session/item/endpoint scope, including pending siblings. */
export function forgetItemReads(client: QueryClient, scope: ItemAccessScope) {
  if (!scope.accountId || !scope.workspaceSlug || !scope.projectId || !scope.issueId) return;
  client.setQueryData(itemAccessOptions(scope).queryKey, true);
  client.removeQueries({
    predicate: ({ queryKey: key }) => {
      if (key[0] === "issue-resources")
        return key[4] === scope.workspaceSlug && matchesOwner(scope, key[2], key[3], key[7], key[11], key[10]);
      if (key[0] === "issue-details") {
        const detailId = key[10] === "detail" || key[10] === "sub-items" ? key[11] : undefined;
        return key[3] === scope.workspaceSlug && matchesOwner(scope, key[1], key[2], key[6], detailId, key[9]);
      }
      return key[0] === "workspace" && key[1] === scope.workspaceSlug && matchesWorkspaceRead(scope, key);
    },
  });
}
/** Retire old query identities before opening a fresh explicit attempt. */
export function retryItemAccess(client: QueryClient, scope: ItemAccessScope) {
  if (isItemAccessDenied(client, scope)) forgetItemReads(client, scope);
  client.setQueryData(itemAccessOptions(scope).queryKey, false);
}
/** Only parent-authorized reads may establish a confirmed item denial. */
export async function readVisibleItem<T>(
  client: QueryClient,
  scope: ItemAccessScope,
  current: () => boolean,
  read: () => Promise<T>
): Promise<T> {
  const authorized = () => current() && getIdentitySessionGeneration(client) === scope.generation;
  const check = () => {
    if (!authorized() || isItemAccessDenied(client, scope)) throw new Error("Work item access changed.");
  };
  check();
  try {
    const value = await read();
    check();
    return value;
  } catch (cause) {
    if (authorized() && isAuthorizationError(cause)) client.setQueryData(itemAccessOptions(scope).queryKey, true);
    throw cause;
  }
}
export function captureVisibleItemRead(
  client: QueryClient,
  scope: ItemAccessScope & TProjectRosterScope,
  queryKey: readonly unknown[],
  signal: AbortSignal
) {
  const current = captureItemReadScope(client, scope, queryKey);
  return () => {
    if (
      !current() ||
      isItemAccessDenied(client, scope) ||
      signal.aborted ||
      getIdentitySessionGeneration(client) !== scope.generation
    )
      throw new Error("Work item access changed.");
  };
}
