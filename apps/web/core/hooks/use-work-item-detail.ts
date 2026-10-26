/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
import { useCurrentUser } from "./use-current-user";
import { useIdentityGeneration } from "./use-identity-generation";
import { useProjectPermissions } from "./use-permissions";
import { useItemAccess } from "./use-item-access";
import type { TIssueDetailScope } from "@/queries/issue-details";
import {
  allSubItems,
  issueByIdentifierOptions,
  issueDetailOptions,
  issueSubItemsOptions,
} from "@/queries/issue-details";

function hasDetailPermissions(scope: TIssueDetailScope) {
  return Boolean(
    scope.accountId &&
    scope.membershipRole !== undefined &&
    scope.membershipRevision &&
    scope.projectId &&
    scope.projectRole !== undefined &&
    scope.projectMembershipRole !== undefined
  );
}

function getProjectMembershipRole(permissions: ReturnType<typeof useProjectPermissions>, projectId?: string) {
  return permissions.rolesQuery.isSuccess && projectId ? permissions.rolesQuery.data?.[projectId] : undefined;
}

export function useWorkItemDetailScope(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  serviceType: TIssueServiceType = EIssueServiceType.ISSUES,
  issueId?: string
) {
  const client = useQueryClient();
  const identity = useCurrentUser();
  const generation = useIdentityGeneration(client);
  const permissions = useProjectPermissions(workspaceSlug, projectId);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const projectMembershipRole = getProjectMembershipRole(permissions, projectId);
  const scope = useMemo<TIssueDetailScope>(
    () => ({
      accountId: identity.isSuccess ? identity.data?.id : undefined,
      sessionGeneration: generation,
      workspaceSlug,
      membershipRole: permissions.workspaceRole,
      membershipRevision: membership?.updated_at,
      projectId,
      projectMembershipRole,
      projectRole: permissions.projectRole,
      serviceType,
    }),
    [
      generation,
      identity.data?.id,
      identity.isSuccess,
      membership?.updated_at,
      permissions.projectRole,
      permissions.workspaceRole,
      projectId,
      projectMembershipRole,
      serviceType,
      workspaceSlug,
    ]
  );
  const itemAccess = useItemAccess({
    accountId: scope.accountId,
    generation,
    workspaceSlug,
    projectId,
    issueId,
    serviceType,
  });
  const isError = identity.isError || permissions.isError;
  const isPending = !isError && (!identity.isSuccess || permissions.isPending);
  const isReady =
    !isPending && !isError && membership?.is_active === true && hasDetailPermissions(scope) && !itemAccess.isDenied;

  return {
    client,
    scope,
    isReady,
    isError,
    isPending,
    isDenied: !isError && !isPending && !isReady,
    retry: async () => {
      if (identity.isError) await identity.refetch();
      else if (permissions.membershipQuery.isError) await permissions.membershipQuery.refetch();
      else if (permissions.rolesQuery.isError) await permissions.rolesQuery.refetch();
      else itemAccess.retry();
    },
  };
}

export function useWorkItemDetail(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  issueId: string | undefined,
  serviceType: TIssueServiceType = EIssueServiceType.ISSUES,
  enabled = true
) {
  const access = useWorkItemDetailScope(workspaceSlug, projectId, serviceType, issueId);
  const query = useQuery(issueDetailOptions(access.client, access.scope, issueId, enabled && access.isReady));
  return {
    ...access,
    query,
    issue: access.isReady && query.isSuccess ? query.data : undefined,
    isPending: access.isPending || (access.isReady && query.isPending),
    isError: access.isError || (access.isReady && query.isError),
    retry: access.isError || access.isDenied ? access.retry : query.refetch,
  };
}

export function useWorkItemByIdentifier(
  workspaceSlug: string | undefined,
  projectIdentifier: string | undefined,
  sequenceId: string | undefined,
  projectId: string | undefined,
  serviceType: TIssueServiceType = EIssueServiceType.ISSUES
) {
  const access = useWorkItemDetailScope(workspaceSlug, projectId, serviceType);
  const query = useQuery(
    issueByIdentifierOptions(access.client, access.scope, projectIdentifier, sequenceId, access.isReady)
  );
  const returnedItem = access.isReady && query.isSuccess ? query.data : undefined;
  const itemAccess = useItemAccess({
    accountId: access.scope.accountId,
    generation: access.scope.sessionGeneration,
    workspaceSlug,
    projectId,
    issueId: returnedItem?.id,
    serviceType,
  });
  const isDenied = access.isDenied || itemAccess.isDenied;
  return {
    ...access,
    query,
    issue: !isDenied ? returnedItem : undefined,
    isPending: access.isPending || (access.isReady && query.isPending),
    isError: access.isError || (access.isReady && query.isError),
    isDenied,
    retry:
      access.isError || access.isDenied
        ? access.retry
        : itemAccess.isDenied
          ? async () => {
              itemAccess.retry();
              await query.refetch();
            }
          : query.refetch,
  };
}

export function useWorkItemSubItems(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  parentIssueId: string | undefined,
  serviceType: TIssueServiceType = EIssueServiceType.ISSUES,
  enabled = true
) {
  const access = useWorkItemDetailScope(workspaceSlug, projectId, serviceType, parentIssueId);
  const query = useQuery(issueSubItemsOptions(access.client, access.scope, parentIssueId, enabled && access.isReady));
  return {
    ...access,
    query,
    subItems: access.isReady && query.isSuccess ? allSubItems(query.data) : undefined,
    isSuccess: access.isReady && query.isSuccess,
    stateDistribution: access.isReady && query.isSuccess ? query.data.state_distribution : undefined,
    isPending: access.isPending || (access.isReady && query.isPending),
    isError: access.isError || (access.isReady && query.isError),
  };
}
