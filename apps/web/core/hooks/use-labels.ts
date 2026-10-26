/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { EUserWorkspaceRoles } from "@plane/types";
import type { IIssueLabel, IUserProjectsRole } from "@plane/types";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useProjectPermissions, useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import {
  labelsById,
  currentLabelRows,
  projectLabelMutationOptions,
  projectLabelOptions,
  projectLabelTree,
  projectRoleSnapshot,
  workspaceLabelOptions,
} from "@/queries/labels";
import type { ProjectCatalogueScope, WorkspaceCatalogueScope } from "@/queries/catalogue-scope";

export function useProjectLabels(workspaceSlug: string | undefined, projectId: string | null | undefined) {
  const client = useQueryClient();
  const account = useCurrentUser();
  const sessionGeneration = useIdentityGeneration(client);
  const scopedProjectId = projectId ?? undefined;
  const permissions = useProjectPermissions(workspaceSlug, scopedProjectId);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const projectRoles = permissions.rolesQuery.isSuccess ? permissions.rolesQuery.data : undefined;
  const projectMembershipRole = projectRoles?.[scopedProjectId ?? ""];
  const scope = useMemo<ProjectCatalogueScope>(
    () => ({
      accountId: account.data?.id,
      workspaceSlug,
      projectId: scopedProjectId,
      workspaceRole: permissions.workspaceRole,
      membershipRevision: membership?.updated_at,
      projectRole: permissions.projectRole,
      projectMembershipRole,
      sessionGeneration,
    }),
    [
      account.data?.id,
      membership?.updated_at,
      permissions.projectRole,
      permissions.workspaceRole,
      scopedProjectId,
      projectMembershipRole,
      sessionGeneration,
      workspaceSlug,
    ]
  );
  const query = useQuery(projectLabelOptions(scope));
  const { data: queryData, isSuccess } = query;
  const labels = useMemo(() => currentLabelRows({ data: queryData, isSuccess }), [queryData, isSuccess]);
  const byId = useMemo(() => labelsById(labels), [labels]);
  const tree = useMemo(() => projectLabelTree(labels), [labels]);
  const canCreate =
    (permissions.workspaceRole === EUserWorkspaceRoles.ADMIN ||
      permissions.workspaceRole === EUserWorkspaceRoles.MEMBER) &&
    permissions.projectRole !== undefined;
  return {
    ...query,
    data: labels,
    scope,
    labels,
    labelsById: byId,
    labelTree: tree,
    labelIds: labels?.map((label) => label.id),
    getLabelById: (labelId: string) => byId?.[labelId] ?? null,
    canCreate,
  };
}

export function useWorkspaceLabels(workspaceSlug: string | undefined) {
  const client = useQueryClient();
  const account = useCurrentUser();
  const sessionGeneration = useIdentityGeneration(client);
  const permissions = useWorkspaceProjectRoles(workspaceSlug);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const roleData: IUserProjectsRole | undefined = permissions.rolesQuery.isSuccess
    ? permissions.rolesQuery.data
    : undefined;
  const projectRoles = useMemo<WorkspaceCatalogueScope["projectRoles"]>(() => {
    if (!roleData) return undefined;
    return projectRoleSnapshot(roleData);
  }, [roleData]);
  const scope = useMemo<WorkspaceCatalogueScope>(
    () => ({
      accountId: account.data?.id,
      workspaceSlug,
      workspaceRole: permissions.workspaceRole,
      membershipRevision: membership?.updated_at,
      projectRoles,
      sessionGeneration,
    }),
    [
      account.data?.id,
      membership?.updated_at,
      permissions.workspaceRole,
      projectRoles,
      sessionGeneration,
      workspaceSlug,
    ]
  );
  const query = useQuery(workspaceLabelOptions(scope));
  const { data: queryData, isSuccess } = query;
  const labels = useMemo(() => currentLabelRows({ data: queryData, isSuccess }), [queryData, isSuccess]);
  const byId = useMemo(() => labelsById(labels), [labels]);
  return {
    ...query,
    data: labels,
    scope,
    labels,
    labelsById: byId,
    labelIds: labels?.map((label) => label.id),
    getLabelById: (labelId: string) => byId?.[labelId] ?? null,
    getProjectLabels: (projectId: string | undefined | null) =>
      projectId ? labels?.filter((label) => label.project_id === projectId) : undefined,
    getProjectLabelIds: (projectId: string | undefined | null) =>
      projectId ? labels?.filter((label) => label.project_id === projectId).map((label) => label.id) : undefined,
    getWorkspaceLabels: (workspaceId: string | undefined) =>
      workspaceId ? labels?.filter((label) => label.workspace_id === workspaceId) : undefined,
    getWorkspaceLabelIds: (workspaceId: string | undefined) =>
      workspaceId ? labels?.filter((label) => label.workspace_id === workspaceId).map((label) => label.id) : undefined,
  };
}

export function useProjectLabelMutations(workspaceSlug: string | undefined, projectId: string | undefined) {
  const client = useQueryClient();
  const projectLabels = useProjectLabels(workspaceSlug, projectId);
  const mutation = useMutation(projectLabelMutationOptions(client, projectLabels.scope));
  const { mutateAsync } = mutation;

  const createLabel = useCallback(
    (data: Partial<IIssueLabel>) => mutateAsync({ type: "create", data }).then((result) => result.label!),
    [mutateAsync]
  );
  const updateLabel = useCallback(
    (labelId: string, data: Partial<IIssueLabel>) =>
      mutateAsync({ type: "update", labelId, data }).then((result) => result.label!),
    [mutateAsync]
  );
  const updateLabelPosition = useCallback(
    (position: {
      draggingLabelId: string;
      droppedParentId: string | null;
      droppedLabelId: string | undefined;
      dropAtEndOfList: boolean;
    }) => mutateAsync({ type: "reorder", position }).then(() => undefined),
    [mutateAsync]
  );
  const deleteLabel = useCallback(
    (labelId: string) => mutateAsync({ type: "delete", labelId }).then(() => undefined),
    [mutateAsync]
  );

  return {
    ...projectLabels,
    createLabel,
    updateLabel,
    updateLabelPosition,
    deleteLabel,
    mutation,
  };
}
