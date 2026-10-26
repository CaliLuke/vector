/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCommittedScope } from "./use-committed-scope";
import { useItemAccess } from "./use-item-access";
import { retryItemAccess } from "@/queries/work-item-access";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { EIssueServiceType } from "@plane/types";
import { useCurrentUser } from "./use-current-user";
import { useProjectPermissions } from "./use-permissions";
import { useIdentityGeneration } from "./use-identity-generation";
import { getCurrentUserId, getIdentitySessionGeneration, identityKeys } from "@/queries/identity";
import { workspaceMembershipOptions, workspaceProjectRolesOptions } from "@/queries/permissions";
import {
  issueActivityOptions,
  issueCommentsOptions,
  issueReactionsOptions,
  discussionScopeReady,
  captureDiscussionWrite,
  currentDiscussionScope,
} from "@/queries/issue-discussion";
import type { DiscussionTarget, DiscussionScope } from "@/queries/issue-discussion";
import { commentMutationOptions, reactionMutationOptions } from "@/queries/issue-discussion-mutations";

function useDiscussionScope(target: DiscussionTarget) {
  const client = useQueryClient();
  const identity = useCurrentUser();
  const generation = useIdentityGeneration(client);
  const permissions = useProjectPermissions(target.workspaceSlug, target.projectId);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const scope: DiscussionScope = {
    ...target,
    serviceType: target.serviceType ?? EIssueServiceType.ISSUES,
    accountId: identity.data?.id,
    generation,
    membershipRole: permissions.workspaceRole,
    membershipRevision: membership?.updated_at,
    projectRole: permissions.projectRole,
    projectMembershipRole: permissions.roles?.[target.projectId],
  };
  const itemAccess = useItemAccess(scope);
  const retryOwner = useCommittedScope(
    JSON.stringify([generation, target.workspaceSlug, target.projectId, target.issueId, target.serviceType])
  );
  const identityQuery = client.getQueryCache().find({ queryKey: identityKeys.currentUser(), exact: true });
  const retryPrerequisites = async () => {
    const check = () => {
      if (
        !retryOwner() ||
        !identityQuery ||
        getIdentitySessionGeneration(client) !== generation ||
        client.getQueryCache().find({ queryKey: identityKeys.currentUser(), exact: true }) !== identityQuery ||
        (scope.accountId && getCurrentUserId(client) !== scope.accountId)
      )
        throw new Error("The discussion retry owner changed.");
    };
    check();
    if (identity.isError) await identity.refetch({ throwOnError: true });
    check();
    const force = itemAccess.isDenied || !discussionScopeReady(scope);
    let next = currentDiscussionScope(client, target);
    const membershipOptions = workspaceMembershipOptions(next.accountId, next.workspaceSlug);
    if (force || client.getQueryState(membershipOptions.queryKey)?.status !== "success")
      await client.fetchQuery({ ...membershipOptions, staleTime: 0 });
    check();
    next = currentDiscussionScope(client, target);
    const rolesOptions = workspaceProjectRolesOptions(
      next.accountId,
      next.workspaceSlug,
      next.membershipRole,
      next.membershipRevision
    );
    if (force || client.getQueryState(rolesOptions.queryKey)?.status !== "success")
      await client.fetchQuery({ ...rolesOptions, staleTime: 0 });
    check();
    next = currentDiscussionScope(client, target);
    retryItemAccess(client, next);
    return next;
  };
  return {
    scope,
    isError:
      itemAccess.isDenied ||
      identity.isError ||
      permissions.isError ||
      (!identity.isPending && !permissions.isPending && !discussionScopeReady(scope)),
    isPending: identity.isPending || permissions.isPending,
    retryPrerequisites,
  };
}

export function useIssueDiscussion(target: DiscussionTarget) {
  const client = useQueryClient();
  const access = useDiscussionScope(target);
  const activity = useQuery(issueActivityOptions(access.scope));
  const comments = useQuery(issueCommentsOptions(access.scope));
  const commentMutation = useMutation(commentMutationOptions(client, access.scope));
  const reactions = useMutation(reactionMutationOptions(client, access.scope, "comments"));
  return {
    ...access,
    ...captureDiscussionWrite(client, access.scope, "comments"),
    activity,
    comments,
    commentMutation,
    reactions,
    isError: access.isError || activity.isError || comments.isError,
    isPending: access.isPending || activity.isPending || comments.isPending,
    retry: async () => {
      const next = await access.retryPrerequisites();
      await Promise.all([client.fetchQuery(issueActivityOptions(next)), client.fetchQuery(issueCommentsOptions(next))]);
    },
  };
}

export function useIssueReactions(target: DiscussionTarget) {
  const client = useQueryClient();
  const access = useDiscussionScope(target);
  const query = useQuery(issueReactionsOptions(access.scope));
  const mutation = useMutation(reactionMutationOptions(client, access.scope, "reactions"));
  return {
    ...access,
    ...captureDiscussionWrite(client, access.scope, "reactions"),
    query,
    mutation,
    isError: access.isError || query.isError,
    isPending: access.isPending || query.isPending,
    retry: async () => client.fetchQuery(issueReactionsOptions(await access.retryPrerequisites())),
  };
}
