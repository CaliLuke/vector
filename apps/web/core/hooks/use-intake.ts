/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import sortBy from "lodash-es/sortBy.js";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCurrentTime } from "./use-current-time";
import { useCurrentUser } from "./use-current-user";
import { useIdentityGeneration } from "./use-identity-generation";
import { useProjectPermissions } from "./use-permissions";
import { intakeDetailOptions, intakeListOptions, intakeMutationOptions } from "@/queries/intake";
import type { IntakeScope } from "@/queries/intake";
import { captureProjectCatalogueCount } from "@/queries/projects";
import type { TProjectCatalogueScope } from "@/queries/projects";
import { EInboxIssueStatus, EIssuesStoreType } from "@plane/types";
import type { EUserWorkspaceRoles, TUserPermissions } from "@plane/types";
import { isAuthorizationError } from "@/queries/authorization-errors";
import { selectIntakeRows } from "@/helpers/intake-preferences";
import { useIntakePreferences } from "@/providers/intake-preferences";
import { useIssueListMutationOwner } from "@/hooks/use-issue-list-mutation-owner";
import { captureIssueListMutationOwner, refreshCapturedIssueLists } from "@/queries/issue-list-mutations";

function intakeAccessReady(input: {
  accountId: string | undefined;
  membershipRole: EUserWorkspaceRoles | undefined;
  membershipRevision: string | undefined;
  projectRole: TUserPermissions | undefined;
  projectMembershipRole: TUserPermissions | undefined;
  isError: boolean;
  isPending: boolean;
}) {
  return (
    !input.isError &&
    !input.isPending &&
    Boolean(
      input.accountId &&
      input.membershipRole &&
      input.membershipRevision &&
      input.projectRole &&
      input.projectMembershipRole
    )
  );
}

function useIntakeAccess(workspaceSlug: string, projectId: string) {
  const client = useQueryClient();
  const account = useCurrentUser();
  const permissions = useProjectPermissions(workspaceSlug, projectId);
  const sessionGeneration = useIdentityGeneration(client);
  const accountId = account.isSuccess ? account.data?.id : undefined;
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const projectMembershipRole = permissions.roles?.[projectId];
  const membershipRole = permissions.workspaceRole;
  const membershipRevision = membership?.updated_at;
  const projectRole = permissions.projectRole;
  const scope = useMemo<IntakeScope>(
    () => ({
      accountId,
      sessionGeneration,
      workspaceSlug,
      projectId,
      membershipRole,
      membershipRevision,
      projectRole,
      projectMembershipRole,
    }),
    [
      accountId,
      sessionGeneration,
      workspaceSlug,
      projectId,
      membershipRole,
      membershipRevision,
      projectRole,
      projectMembershipRole,
    ]
  );
  const isError = account.isError || permissions.isError;
  const isPending = account.isPending || permissions.isPending;
  const isReady = intakeAccessReady({
    accountId,
    membershipRole,
    membershipRevision,
    projectRole,
    projectMembershipRole,
    isError,
    isPending,
  });
  const retryAccess = async () => {
    if (account.isError) await account.refetch();
    else if (permissions.membershipQuery.isError) await permissions.membershipQuery.refetch();
    else if (permissions.rolesQuery.isError) await permissions.rolesQuery.refetch();
  };
  const catalogueScope: TProjectCatalogueScope = {
    accountId,
    sessionGeneration,
    workspaceSlug,
    workspaceRole: membershipRole,
    membershipRevision,
    projectId,
    projectMembershipRole,
    projectRole,
    projectRoles: permissions.roles ? sortBy(Object.entries(permissions.roles), ([id]) => id) : undefined,
  };
  return {
    client,
    scope,
    catalogueScope,
    isError,
    isPending,
    isReady,
    isDenied: !isError && !isPending && !isReady,
    retryAccess,
  };
}

export function useIntakeList(enabled = true) {
  const preferences = useIntakePreferences();
  const { currentTime } = useCurrentTime();
  const access = useIntakeAccess(preferences.workspaceSlug, preferences.projectId);
  const query = useInfiniteQuery(intakeListOptions(access.client, access.scope, preferences, enabled));
  const readDenied = isAuthorizationError(query.error);
  const rows = useMemo(
    () =>
      enabled && access.isReady && !readDenied && query.data
        ? selectIntakeRows(
            query.data.pages.flatMap((page) => page.results),
            preferences,
            currentTime.getTime()
          )
        : [],
    [enabled, access.isReady, readDenied, query.data, preferences, currentTime]
  );
  return {
    ...query,
    rows,
    accessDenied: access.isDenied || readDenied,
    isPending: !access.isDenied && (access.isPending || query.isPending),
    isError: access.isError || query.isError,
    retry: access.isError ? access.retryAccess : query.refetch,
  };
}

export function useIntakeDetail(workspaceSlug: string, projectId: string, issueId: string) {
  const access = useIntakeAccess(workspaceSlug, projectId);
  const query = useQuery(intakeDetailOptions(access.client, access.scope, issueId));
  return {
    ...query,
    data: access.isReady && query.isSuccess ? query.data : undefined,
    accessDenied: access.isDenied,
    isPending: !access.isDenied && (access.isPending || query.isPending),
    isError: access.isError || query.isError,
    retry: access.isError ? access.retryAccess : query.refetch,
  };
}

export function useIntakeMutation(workspaceSlug: string, projectId: string) {
  const access = useIntakeAccess(workspaceSlug, projectId);
  const resolveListOwner = useIssueListMutationOwner(EIssuesStoreType.PROJECT);
  return useMutation(
    intakeMutationOptions(access.client, access.scope, async (input) => {
      const counter =
        input.kind === "issue"
          ? undefined
          : captureProjectCatalogueCount(access.client, access.catalogueScope, projectId);
      let capturedLists: ReturnType<typeof captureIssueListMutationOwner> | undefined;
      try {
        const owner = await resolveListOwner(workspaceSlug, projectId);
        if (owner) capturedLists = captureIssueListMutationOwner(access.client, owner);
      } catch (error) {
        console.error("Could not capture the work item list for intake reconciliation", error);
      }
      return {
        beforeWrite: async () => {
          await counter?.cancelReads();
        },
        accepted: async (receipt, isCurrent) => {
          if (
            isCurrent() &&
            capturedLists &&
            receipt.saved?.status === EInboxIssueStatus.ACCEPTED &&
            receipt.saved.issue.project_id === projectId
          )
            await refreshCapturedIssueLists(access.client, capturedLists);
          await counter?.refresh();
        },
      };
    })
  );
}
