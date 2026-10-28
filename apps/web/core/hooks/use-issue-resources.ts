/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useCallback, useMemo, useState } from "react";
import type { IWorkspaceMemberMe, TIssueLink, TIssueRelation, TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { isWorkspaceRole } from "@/helpers/permission-policy";
import { currentUserOptions, getIdentitySessionGeneration } from "@/queries/identity";
import { workspaceMembershipOptions, workspaceProjectRolesOptions } from "@/queries/permissions";
import { useItemAccess } from "@/hooks/use-item-access";
import type { TIssueResourceScope } from "@/queries/issue-resources";
import {
  issueAttachmentsOptions,
  issueAttachmentMutationOptions,
  issueLinksOptions,
  issueLinkMutationOptions,
  issueRelationsOptions,
  issueRelationMutationOptions,
} from "@/queries/issue-resources";

function getAccessState({
  accountId,
  identityPending,
  identityError,
  membership,
  membershipPending,
  membershipError,
  projectMembershipRole,
  projectRolesPending,
  projectRolesError,
  workspaceSlug,
  projectId,
}: {
  accountId: string | undefined;
  identityPending: boolean;
  identityError: boolean;
  membership: IWorkspaceMemberMe | undefined;
  membershipPending: boolean;
  membershipError: boolean;
  projectMembershipRole: number | undefined;
  projectRolesPending: boolean;
  projectRolesError: boolean;
  workspaceSlug: string | undefined;
  projectId: string | undefined;
}) {
  if (identityError || membershipError || projectRolesError) return "error";
  if (!workspaceSlug || !projectId) return "denied";
  if (identityPending || membershipPending) return "pending";
  if (!accountId || !membership?.is_active || !isWorkspaceRole(membership.role)) return "denied";
  if (projectRolesPending) return "pending";
  if (membership.updated_at) return projectMembershipRole ? "ready" : "denied";
  return "pending";
}

function useIssueResourceScope(
  workspaceSlug: string,
  projectId: string,
  issueId: string,
  serviceType: TIssueServiceType
) {
  const client = useQueryClient();
  const router = useRouter();
  const currentUser = useCurrentUser();
  const sessionGeneration = useIdentityGeneration(client);
  const permissions = useProjectPermissions(workspaceSlug, projectId);
  const accountId = currentUser.isSuccess ? currentUser.data?.id : undefined;
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const projectMembershipRole =
    permissions.rolesQuery.isSuccess && projectId ? permissions.rolesQuery.data?.[projectId] : undefined;
  const itemAccess = useItemAccess({
    accountId,
    generation: sessionGeneration,
    workspaceSlug,
    projectId,
    issueId,
    serviceType,
  });
  const retryItemAccess = itemAccess.retry;
  const permissionState = getAccessState({
    accountId,
    identityPending: currentUser.isPending,
    identityError: currentUser.isError,
    membership,
    membershipPending: permissions.membershipQuery.isPending,
    membershipError: permissions.membershipQuery.isError,
    projectMembershipRole,
    projectRolesPending: permissions.rolesQuery.isPending,
    projectRolesError: permissions.rolesQuery.isError,
    workspaceSlug,
    projectId,
  });
  const accessState = itemAccess.isDenied ? "denied" : permissionState;
  const scope = useMemo<TIssueResourceScope>(
    () => ({
      accountId,
      sessionGeneration,
      workspaceSlug,
      membershipRole: membership?.role ?? undefined,
      membershipRevision: membership?.updated_at,
      projectId,
      projectMembershipRole,
      projectRole: permissions.projectRole,
      serviceType,
      issueId,
    }),
    [
      accountId,
      issueId,
      membership?.role,
      membership?.updated_at,
      permissions.projectRole,
      projectId,
      projectMembershipRole,
      serviceType,
      sessionGeneration,
      workspaceSlug,
    ]
  );
  const initialRoute = router.state.location.pathname;
  const retry = useCallback(async () => {
    const isRetryOwnerCurrent = () =>
      router.state.location.pathname === initialRoute && getIdentitySessionGeneration(client) === sessionGeneration;
    if (!isRetryOwnerCurrent()) return;
    retryItemAccess();
    const currentUserQuery = currentUserOptions();
    const user = await client.fetchQuery({ ...currentUserQuery, staleTime: 0 });
    if (!isRetryOwnerCurrent() || !user?.id) return;
    const membershipQuery = workspaceMembershipOptions(user.id, workspaceSlug);
    const refreshedMembership = await client.fetchQuery({ ...membershipQuery, staleTime: 0 });
    if (!isRetryOwnerCurrent() || !refreshedMembership.is_active || !isWorkspaceRole(refreshedMembership.role)) return;
    const rolesQuery = workspaceProjectRolesOptions(
      user.id,
      workspaceSlug,
      refreshedMembership.role,
      refreshedMembership.updated_at
    );
    await client.fetchQuery({ ...rolesQuery, staleTime: 0 });
    if (!isRetryOwnerCurrent()) return;
  }, [client, initialRoute, retryItemAccess, router, sessionGeneration, workspaceSlug]);
  return {
    client,
    scope,
    isReady: accessState === "ready",
    isPending: accessState === "pending",
    isError: accessState === "error",
    isDenied: accessState === "denied",
    retry,
  };
}

type IssueResourceArgs = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  serviceType?: TIssueServiceType;
};

export type TAttachmentUploadStatus = { id: string; name: string; progress: number; size: number; type: string };

export function issueAttachmentUploadScopeKey(scope: TIssueResourceScope) {
  return JSON.stringify([
    scope.accountId,
    scope.sessionGeneration,
    scope.workspaceSlug,
    scope.projectId,
    scope.serviceType,
    scope.issueId,
  ]);
}

export function useIssueAttachments({
  workspaceSlug,
  projectId,
  issueId,
  serviceType = EIssueServiceType.ISSUES,
}: IssueResourceArgs) {
  const access = useIssueResourceScope(workspaceSlug, projectId, issueId, serviceType);
  const options = issueAttachmentsOptions(access.client, access.scope);
  const query = useRecordCollection(options);
  const attachments = query.isSuccess && access.isReady ? query.data : undefined;
  const [uploadStatusByScope, setUploadStatusByScope] = useState<Record<string, readonly TAttachmentUploadStatus[]>>(
    {}
  );
  const uploadScopeKey = issueAttachmentUploadScopeKey(access.scope);
  const uploadStatus = uploadStatusByScope[uploadScopeKey] ?? [];
  const updateUploadStatus = (
    scopeKey: string,
    update: (current: readonly TAttachmentUploadStatus[]) => readonly TAttachmentUploadStatus[]
  ) => {
    setUploadStatusByScope((current) => {
      const next = update(current[scopeKey] ?? []);
      if (next.length > 0) return { ...current, [scopeKey]: next };
      const { [scopeKey]: _removed, ...remaining } = current;
      return remaining;
    });
  };
  const { mutateAsync: mutateAttachment, isPending } = useMutation(
    issueAttachmentMutationOptions(access.client, access.scope)
  );
  const create = async (file: File) => {
    if (!access.isReady) throw new Error("The work item access scope is unavailable.");
    const id = crypto.randomUUID();
    updateUploadStatus(uploadScopeKey, (previous) => [
      ...previous,
      { id, name: file.name, progress: 0, size: file.size, type: file.type },
    ]);
    try {
      return await mutateAttachment({
        kind: "upload",
        file,
        onProgress: (progress) =>
          updateUploadStatus(uploadScopeKey, (previous) =>
            previous.map((entry) => (entry.id === id ? { ...entry, progress } : entry))
          ),
      });
    } finally {
      updateUploadStatus(uploadScopeKey, (previous) => previous.filter((entry) => entry.id !== id));
    }
  };
  const remove = (attachmentId: string) => mutateAttachment({ kind: "delete", attachmentId });
  return {
    ...access,
    query,
    attachments,
    uploadStatus,
    uploadScopeKey,
    create,
    remove,
    isUploading: isPending,
  };
}

export function useIssueLinks({
  workspaceSlug,
  projectId,
  issueId,
  serviceType = EIssueServiceType.ISSUES,
}: IssueResourceArgs) {
  const access = useIssueResourceScope(workspaceSlug, projectId, issueId, serviceType);
  const options = issueLinksOptions(access.client, access.scope);
  const query = useRecordCollection(options);
  const links = query.isSuccess && access.isReady ? query.data : undefined;
  const { mutateAsync: mutateLink, isPending } = useMutation(issueLinkMutationOptions(access.client, access.scope));
  const create = useCallback(
    async (data: Partial<TIssueLink>) => {
      return mutateLink({ kind: "create", data });
    },
    [mutateLink]
  );
  const update = useCallback(
    async (linkId: string, data: Partial<TIssueLink>) => {
      return mutateLink({ kind: "update", linkId, data });
    },
    [mutateLink]
  );
  const remove = useCallback(
    async (linkId: string) => {
      return mutateLink({ kind: "delete", linkId });
    },
    [mutateLink]
  );
  return { ...access, query, links, create, update, remove, isMutating: isPending };
}

export function useIssueRelations({
  workspaceSlug,
  projectId,
  issueId,
  serviceType = EIssueServiceType.ISSUES,
}: IssueResourceArgs) {
  const access = useIssueResourceScope(workspaceSlug, projectId, issueId, serviceType);
  const options = issueRelationsOptions(access.client, access.scope);
  const query = useQuery(options);
  const relations: TIssueRelation | undefined = query.isSuccess && access.isReady ? query.data : undefined;
  const { mutateAsync: mutateRelation, isPending } = useMutation(
    issueRelationMutationOptions(access.client, access.scope)
  );
  const create = useCallback(
    async (relationType: import("@plane/types").TIssueRelationTypes, issueIds: string[]) => {
      return mutateRelation({ kind: "create", relationType, issueIds });
    },
    [mutateRelation]
  );
  const remove = useCallback(
    async (relationType: import("@plane/types").TIssueRelationTypes, relatedIssueId: string) => {
      return mutateRelation({ kind: "delete", relationType, relatedIssueId });
    },
    [mutateRelation]
  );
  return { ...access, query, relations, create, remove, isMutating: isPending };
}
