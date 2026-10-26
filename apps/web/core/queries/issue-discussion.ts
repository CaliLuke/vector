/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { effectiveProjectRole } from "@/helpers/permission-policy";
import { readVisibleItem, isItemAccessDenied } from "./work-item-access";
import { isAuthorizationError } from "./authorization-errors";
import { queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient, Query } from "@tanstack/react-query";
import { EIssueServiceType } from "@plane/types";
import type { TIssueServiceType, TUserPermissions, IWorkspaceMemberMe } from "@plane/types";
import type { TProjectRosterScope } from "./member-rosters";
import { captureItemReadScope, captureItemLineageScope } from "./work-item-read-scope";
import { getIdentitySessionGeneration, getCurrentUserId } from "./identity";
import { captureProjectAccessScope } from "./project-access-scope";
import { permissionKeys } from "./permissions";
import { IssueActivityService } from "@/services/issue/issue_activity.service";
import { IssueCommentService } from "@/services/issue/issue_comment.service";
import { IssueReactionService } from "@/services/issue/issue_reaction.service";

export type DiscussionTarget = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  serviceType?: TIssueServiceType;
};
export type DiscussionScope = TProjectRosterScope &
  DiscussionTarget & {
    generation: number;
    projectMembershipRole: TUserPermissions | undefined;
  };
export type DiscussionCollection = "activity" | "comments" | "reactions";
export function currentDiscussionScope(client: QueryClient, target: DiscussionTarget): DiscussionScope {
  const accountId = getCurrentUserId(client);
  const membership = client.getQueryData<IWorkspaceMemberMe>(
    permissionKeys.workspaceMembership(accountId, target.workspaceSlug)
  );
  const roles = client.getQueryData<Record<string, TUserPermissions>>(
    permissionKeys.workspaceProjectRoles(
      accountId,
      target.workspaceSlug,
      membership?.role ?? undefined,
      membership?.updated_at
    )
  );
  return {
    ...target,
    accountId,
    generation: getIdentitySessionGeneration(client),
    membershipRole: membership?.role ?? undefined,
    membershipRevision: membership?.updated_at,
    projectRole: effectiveProjectRole(membership?.role ?? null, roles, target.projectId) ?? undefined,
    projectMembershipRole: roles?.[target.projectId],
  };
}

/** Local editors belong to the actor/session/item, independently of a refresh's readiness. */
export const discussionOwnerKey = (scope: DiscussionScope) =>
  JSON.stringify([
    scope.accountId,
    scope.generation,
    scope.workspaceSlug,
    scope.projectId,
    scope.issueId,
    scope.serviceType ?? EIssueServiceType.ISSUES,
  ]);
export const discussionKey = (scope: DiscussionScope, collection: DiscussionCollection) =>
  [
    "workspace",
    scope.workspaceSlug,
    "issue-discussion",
    scope.accountId,
    scope.generation,
    scope.projectId,
    scope.issueId,
    scope.serviceType ?? EIssueServiceType.ISSUES,
    scope.membershipRole,
    scope.membershipRevision,
    scope.projectRole,
    scope.projectMembershipRole,
    collection,
  ] as const;
export function discussionScopeReady(scope: DiscussionScope) {
  return Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.projectId &&
    scope.issueId &&
    scope.membershipRevision &&
    scope.projectMembershipRole !== undefined &&
    scope.projectRole !== undefined &&
    [5, 15, 20].includes(scope.projectRole)
  );
}
function captureDiscussionScope(client: QueryClient, scope: DiscussionScope, collection: DiscussionCollection) {
  const key = discussionKey(scope, collection);
  const verified = captureItemReadScope(client, scope, key);
  const lineage = captureItemLineageScope(client, scope, key);
  const rolesKey = permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    scope.membershipRole,
    scope.membershipRevision
  );
  const roleMatches = () =>
    client.getQueryData<Record<string, TUserPermissions>>(rolesKey)?.[scope.projectId] === scope.projectMembershipRole;
  const same = () => getIdentitySessionGeneration(client) === scope.generation && roleMatches();
  return {
    isVerified: () => discussionScopeReady(scope) && same() && verified(),
    isCurrent: () => same() && lineage(),
  };
}
/** New writes need a successful item collection; accepted effects retain lineage through transient errors. */
export function captureDiscussionWrite(client: QueryClient, scope: DiscussionScope, collection: DiscussionCollection) {
  const captured = captureDiscussionScope(client, scope, collection);
  const query = client.getQueryCache().find({ queryKey: discussionKey(scope, collection), exact: true });
  return {
    isVerified: () => captured.isVerified() && !isItemAccessDenied(client, scope) && query?.state.status === "success",
    isCurrent: () =>
      captured.isCurrent() && !isItemAccessDenied(client, scope) && !isAuthorizationError(query?.state.error),
  };
}
function collectionOptions<T>(
  scope: DiscussionScope,
  collection: DiscussionCollection,
  read: (signal: AbortSignal) => Promise<T>
) {
  const queryKey = discussionKey(scope, collection);
  return queryOptions({
    queryKey,
    queryFn: discussionScopeReady(scope)
      ? async ({ client, signal }) => {
          const captured = captureDiscussionScope(client, scope, collection);
          return readVisibleItem(
            client,
            scope,
            () => !signal.aborted && captured.isVerified(),
            () => read(signal)
          );
        }
      : skipToken,
    retry: false,
    refetchOnMount: true,
    staleTime: 0,
  });
}
export function issueActivityOptions(scope: DiscussionScope) {
  return collectionOptions(scope, "activity", (signal) =>
    new IssueActivityService(scope.serviceType).getIssueActivities(
      scope.workspaceSlug,
      scope.projectId,
      scope.issueId,
      signal
    )
  );
}
export function issueCommentsOptions(scope: DiscussionScope) {
  return collectionOptions(scope, "comments", (signal) =>
    new IssueCommentService(scope.serviceType).getIssueComments(
      scope.workspaceSlug,
      scope.projectId,
      scope.issueId,
      signal
    )
  );
}
export function issueReactionsOptions(scope: DiscussionScope) {
  return collectionOptions(scope, "reactions", (signal) =>
    new IssueReactionService(scope.serviceType).listIssueReactions(
      scope.workspaceSlug,
      scope.projectId,
      scope.issueId,
      signal
    )
  );
}
export type IssueActivityRefreshOwner = {
  accountId: string | undefined;
  generation: number;
  isCurrent: () => boolean;
};
/** Capture at the primary action's start, never from a replacement actor after its await. */
export function captureIssueActivityRefreshOwner(
  client: QueryClient,
  target: DiscussionTarget
): IssueActivityRefreshOwner {
  const accountId = getCurrentUserId(client),
    generation = getIdentitySessionGeneration(client);
  const scope = currentDiscussionScope(client, target);
  const access = captureProjectAccessScope(client, scope);
  return {
    accountId,
    generation,
    isCurrent: () =>
      access.isCurrent() &&
      currentDiscussionScope(client, target).projectMembershipRole === scope.projectMembershipRole &&
      !isItemAccessDenied(client, { ...target, accountId, generation }),
  };
}
/** Carry a native refresh effect across an entity action’s primary request. */
export function captureIssueActivityRefresh(client: QueryClient, target: DiscussionTarget) {
  const owner = captureIssueActivityRefreshOwner(client, target);
  return () => refreshIssueActivity(client, target, owner);
}
export async function refreshIssueActivity(
  client: QueryClient,
  target: DiscussionTarget,
  owner: IssueActivityRefreshOwner
) {
  const current = () =>
    Boolean(
      owner.accountId &&
      getCurrentUserId(client) === owner.accountId &&
      getIdentitySessionGeneration(client) === owner.generation &&
      owner.isCurrent()
    );
  if (!current()) return;
  const queries = new Set(
    client.getQueryCache().findAll({
      predicate: ({ queryKey: key }) =>
        key[0] === "workspace" &&
        key[1] === target.workspaceSlug &&
        key[2] === "issue-discussion" &&
        key[3] === owner.accountId &&
        key[4] === owner.generation &&
        key[5] === target.projectId &&
        key[6] === target.issueId &&
        key[7] === (target.serviceType ?? EIssueServiceType.ISSUES) &&
        key[12] === "activity",
    })
  );
  const filters = { predicate: (query: Query) => queries.has(query) };
  await client.cancelQueries(filters);
  if (current()) await client.invalidateQueries(filters);
}
