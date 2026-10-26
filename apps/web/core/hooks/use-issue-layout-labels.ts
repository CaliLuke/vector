/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import sortBy from "lodash-es/sortBy.js";
import { EUserPermissions } from "@plane/constants";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useProjectLabels, useWorkspaceLabels } from "@/hooks/use-labels";
import { useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import { isWorkspaceRole } from "@/helpers/permission-policy";
import { labelKeys } from "@/queries/labels";

type TAccessState = "pending" | "error" | "denied" | "ready";

function canReadProjectLabels(role: number | undefined, projectId: string | undefined) {
  return Boolean(
    projectId &&
    (role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER || role === EUserPermissions.GUEST)
  );
}

function getAccessState({
  account,
  permissions,
  workspaceSlug,
  isWorkspaceLevel,
  projectId,
  projectRole,
}: {
  account: ReturnType<typeof useCurrentUser>;
  permissions: ReturnType<typeof useWorkspaceProjectRoles>;
  workspaceSlug: string | undefined;
  isWorkspaceLevel: boolean;
  projectId: string | undefined;
  projectRole: number | undefined;
}): TAccessState {
  if (account.isError || permissions.isError) return "error";
  if (account.isPending || (workspaceSlug && permissions.isPending)) return "pending";
  if (!account.data?.id || !workspaceSlug || !isWorkspaceRole(permissions.workspaceRole)) return "denied";
  if (!permissions.rolesQuery.isSuccess) return "denied";
  if (!isWorkspaceLevel && !canReadProjectLabels(projectRole, projectId)) return "denied";
  return "ready";
}

function getLabelPrefix(isWorkspaceLevel: boolean, workspaceSlug: string | undefined, projectId: string | undefined) {
  return isWorkspaceLevel
    ? labelKeys.workspacePrefix(workspaceSlug)
    : labelKeys.projectPrefix(workspaceSlug, projectId);
}

async function retryLabelAccess({
  accountError,
  refetchAccount,
  membershipError,
  refetchMembership,
  rolesError,
  refetchRoles,
  refetchLabels,
  client,
  workspaceSlug,
  labelPrefix,
}: {
  accountError: boolean;
  refetchAccount: () => Promise<unknown>;
  membershipError: boolean;
  refetchMembership: () => Promise<unknown>;
  rolesError: boolean;
  refetchRoles: () => Promise<unknown>;
  refetchLabels: () => Promise<unknown>;
  client: ReturnType<typeof useQueryClient>;
  workspaceSlug: string | undefined;
  labelPrefix: readonly unknown[];
}) {
  if (accountError) await refetchAccount();
  if (membershipError) await refetchMembership();
  if (rolesError && !accountError && !membershipError) await refetchRoles();

  if (workspaceSlug && (accountError || membershipError)) {
    await client.invalidateQueries(
      {
        predicate: (query) =>
          query.queryKey[0] === "permissions" &&
          query.queryKey[2] === workspaceSlug &&
          query.queryKey[3] === "project-roles",
      },
      { throwOnError: false }
    );
  }
  if (accountError || membershipError || rolesError)
    await client.invalidateQueries({ queryKey: labelPrefix }, { throwOnError: false });
  else await refetchLabels();
}

function getIdsRevision(labels: readonly { id: string }[] | undefined) {
  if (!labels) return undefined;
  return JSON.stringify(sortBy([...new Set(labels.map((label) => label.id))], (id) => id));
}

export function useIssueLayoutLabels(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  isWorkspaceLevel: boolean
) {
  const client = useQueryClient();
  const account = useCurrentUser();
  const permissions = useWorkspaceProjectRoles(workspaceSlug);
  const projectLabels = useProjectLabels(
    isWorkspaceLevel ? undefined : workspaceSlug,
    isWorkspaceLevel ? undefined : projectId
  );
  const workspaceLabels = useWorkspaceLabels(isWorkspaceLevel ? workspaceSlug : undefined);
  const selectedLabels = isWorkspaceLevel ? workspaceLabels : projectLabels;
  const projectRole = isWorkspaceLevel ? undefined : projectLabels.scope.projectRole;
  const accessState = getAccessState({ account, permissions, workspaceSlug, isWorkspaceLevel, projectId, projectRole });
  const scopeIsReady = accessState === "ready";
  const scopeIsError = accessState === "error" || accessState === "denied";
  const isAccessDenied = accessState === "denied";
  const isError = accessState === "error" || selectedLabels.isError;
  const isPending = accessState === "pending" || (scopeIsReady && selectedLabels.isPending && !selectedLabels.isError);
  const accountError = account.isError;
  const refetchAccount = account.refetch;
  const membershipError = permissions.membershipQuery.isError;
  const refetchMembership = permissions.membershipQuery.refetch;
  const rolesError = permissions.rolesQuery.isError;
  const refetchRoles = permissions.rolesQuery.refetch;
  const refetchLabels = selectedLabels.refetch;
  const labelPrefix = getLabelPrefix(isWorkspaceLevel, workspaceSlug, projectId);
  const retry = useCallback(
    () =>
      retryLabelAccess({
        accountError,
        refetchAccount,
        membershipError,
        refetchMembership,
        rolesError,
        refetchRoles,
        refetchLabels,
        client,
        workspaceSlug,
        labelPrefix,
      }),
    [
      accountError,
      client,
      labelPrefix,
      membershipError,
      refetchAccount,
      refetchLabels,
      refetchMembership,
      refetchRoles,
      rolesError,
      workspaceSlug,
    ]
  );
  const labels = selectedLabels.labels;
  const labelsById = selectedLabels.labelsById;
  const idsRevision = useMemo(() => getIdsRevision(labels), [labels]);
  const scopeKey = JSON.stringify(
    isWorkspaceLevel ? labelKeys.workspace(workspaceLabels.scope) : labelKeys.project(projectLabels.scope)
  );

  return {
    labels,
    labelsById,
    scopeKey,
    idsRevision,
    isPending,
    isError,
    isAccessDenied,
    scopeIsReady,
    scopeIsError,
    retry,
  };
}
