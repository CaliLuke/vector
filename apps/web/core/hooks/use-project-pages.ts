/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useProjectPermissions } from "@/hooks/use-permissions";
import {
  canReadProjectPages,
  createProjectPageOptions,
  deleteProjectPageOptions,
  duplicateProjectPageOptions,
  projectPageActionOptions,
  projectPageOptions,
  projectPagesOptions,
  updateProjectPageDescriptionOptions,
  updateProjectPageOptions,
} from "@/queries/pages";
import { projectRolesRevision } from "@/queries/project-access-scope";
import type { TProjectPageScope } from "@/queries/pages";
import { useIdentityGeneration } from "./use-identity-generation";

function useProjectPageScope(workspaceSlug: string | undefined, projectId: string | undefined) {
  const client = useQueryClient();
  const currentUser = useCurrentUser();
  const permissions = useProjectPermissions(workspaceSlug, projectId);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const roles = permissions.roles;
  const accountId = currentUser.isSuccess ? currentUser.data?.id : undefined;
  const projectMembershipRole = projectId ? roles?.[projectId] : undefined;
  const scope: TProjectPageScope = {
    accountId,
    workspaceSlug,
    membershipRole: permissions.workspaceRole,
    membershipRevision: membership?.updated_at,
    projectId,
    projectRole: permissions.projectRole,
    projectMembershipRole,
    rolesRevision: projectRolesRevision(roles),
    sessionGeneration: useIdentityGeneration(client),
  };
  const isError = currentUser.isError || permissions.isError;
  const isReady = currentUser.isSuccess && !isError && canReadProjectPages(scope);
  const status = isError
    ? "error"
    : isReady
      ? "ready"
      : currentUser.isPending || permissions.isPending
        ? "pending"
        : "denied";
  return {
    scope,
    queryScope: isReady ? scope : undefined,
    isReady,
    isDenied: status === "denied",
    isError,
    isPending: status === "pending",
    client,
  };
}

export function useProjectPages(workspaceSlug: string | undefined, projectId: string | undefined) {
  const { client, scope, ...access } = useProjectPageScope(workspaceSlug, projectId);
  const query = useRecordCollection(projectPagesOptions(client, scope));
  return { ...query, access: { ...access, scope } };
}

export function useProjectPage(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  pageId: string | undefined,
  trackVisit = true
) {
  const { client, scope, ...access } = useProjectPageScope(workspaceSlug, projectId);
  const query = useQuery(projectPageOptions(client, scope, pageId, trackVisit));
  return { ...query, access: { ...access, scope } };
}

export function useCreateProjectPage(workspaceSlug: string | undefined, projectId: string | undefined) {
  const { client, scope } = useProjectPageScope(workspaceSlug, projectId);
  return useMutation(createProjectPageOptions(client, scope));
}

export function useUpdateProjectPage(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  pageId: string | undefined
) {
  const { client, scope } = useProjectPageScope(workspaceSlug, projectId);
  return useMutation(updateProjectPageOptions(client, scope, pageId ?? ""));
}

export function useUpdateProjectPageDescription(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  pageId: string | undefined
) {
  const { client, scope } = useProjectPageScope(workspaceSlug, projectId);
  return useMutation(updateProjectPageDescriptionOptions(client, scope, pageId ?? ""));
}

export function useDeleteProjectPage(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  pageId: string | undefined
) {
  const { client, scope } = useProjectPageScope(workspaceSlug, projectId);
  return useMutation(deleteProjectPageOptions(client, scope, pageId ?? ""));
}

export function useDuplicateProjectPage(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  pageId: string | undefined
) {
  const { client, scope } = useProjectPageScope(workspaceSlug, projectId);
  return useMutation(duplicateProjectPageOptions(client, scope, pageId ?? ""));
}

export function useProjectPageAction(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  pageId: string | undefined
) {
  const { client, scope } = useProjectPageScope(workspaceSlug, projectId);
  return useMutation(projectPageActionOptions(client, scope, pageId ?? ""));
}
