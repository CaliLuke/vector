/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { EUserWorkspaceRoles } from "@plane/types";
import type { EUserPermissions as TProjectRole } from "@plane/types";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { workspaceMembershipOptions } from "@/queries/permissions";
import {
  addProjectMembersOptions,
  deleteWorkspaceInvitationOptions,
  inviteWorkspaceOptions,
  removeProjectMemberOptions,
  removeWorkspaceMemberOptions,
  updateProjectMemberRoleOptions,
  updateWorkspaceInvitationOptions,
  updateWorkspaceMemberRoleOptions,
  workspaceInvitationListOptions,
} from "@/queries/member-roster-mutations";

function useMemberMutationContext() {
  const client = useQueryClient();
  const accountId = useCurrentUserId();
  useIdentityGeneration(client);
  return { client, accountId };
}

export function useWorkspaceInvitations(workspaceSlug: string | undefined) {
  const client = useQueryClient();
  useIdentityGeneration(client);
  const accountId = useCurrentUserId();
  const membershipQuery = useQuery(workspaceMembershipOptions(accountId, workspaceSlug));
  const membership = membershipQuery.isSuccess && membershipQuery.data.is_active ? membershipQuery.data : undefined;
  const membershipRole = membership?.role ?? undefined;
  const canRead = membershipRole === EUserWorkspaceRoles.MEMBER || membershipRole === EUserWorkspaceRoles.ADMIN;
  const query = useQuery(
    workspaceInvitationListOptions({
      accountId,
      workspaceSlug,
      membershipRole: canRead ? membershipRole : undefined,
      membershipRevision: canRead ? membership?.updated_at : undefined,
    })
  );
  return {
    ...query,
    membershipQuery,
    data: canRead && query.isSuccess ? query.data : undefined,
  };
}

export function useInviteWorkspaceMembers(workspaceSlug: string) {
  const { client, accountId } = useMemberMutationContext();
  return useMutation(inviteWorkspaceOptions(client, accountId, workspaceSlug));
}

export function useUpdateWorkspaceMemberRole(workspaceSlug: string, memberId: string) {
  const { client, accountId } = useMemberMutationContext();
  return useMutation(updateWorkspaceMemberRoleOptions(client, accountId, workspaceSlug, memberId));
}

export function useRemoveWorkspaceMember(workspaceSlug: string, membershipId: string, memberUserId: string) {
  const { client, accountId } = useMemberMutationContext();
  return useMutation(removeWorkspaceMemberOptions(client, accountId, workspaceSlug, membershipId, memberUserId));
}

export function useUpdateWorkspaceInvitation(workspaceSlug: string, invitationId: string) {
  const { client, accountId } = useMemberMutationContext();
  return useMutation(updateWorkspaceInvitationOptions(client, accountId, workspaceSlug, invitationId));
}

export function useDeleteWorkspaceInvitation(workspaceSlug: string, invitationId: string) {
  const { client, accountId } = useMemberMutationContext();
  return useMutation(deleteWorkspaceInvitationOptions(client, accountId, workspaceSlug, invitationId));
}

export function useAddProjectMembers(workspaceSlug: string, projectId: string, projectRole: TProjectRole | undefined) {
  const { client, accountId } = useMemberMutationContext();
  return useMutation(addProjectMembersOptions(client, accountId, workspaceSlug, projectId, projectRole));
}

export function useUpdateProjectMemberRole(
  workspaceSlug: string,
  projectId: string,
  memberId: string,
  projectRole: TProjectRole | undefined
) {
  const { client, accountId } = useMemberMutationContext();
  return useMutation(
    updateProjectMemberRoleOptions(client, accountId, workspaceSlug, projectId, memberId, projectRole)
  );
}

export function useRemoveProjectMember(
  workspaceSlug: string,
  projectId: string,
  memberId: string,
  projectRole: TProjectRole | undefined
) {
  const { client, accountId } = useMemberMutationContext();
  return useMutation(removeProjectMemberOptions(client, accountId, workspaceSlug, projectId, memberId, projectRole));
}
