/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useProjectPermissions, useWorkspacePermissions } from "@/hooks/use-permissions";
import { projectMemberRosterOptions, workspaceMemberRosterOptions } from "@/queries/member-rosters";

export function useWorkspaceMemberRoster(workspaceSlug: string | undefined) {
  const { data: user } = useCurrentUser();
  const permissions = useWorkspacePermissions(workspaceSlug);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const query = useRecordCollection(
    workspaceMemberRosterOptions({
      accountId: user?.id,
      workspaceSlug,
      membershipRole: permissions.workspaceRole,
      membershipRevision: membership?.updated_at,
    }),
    (row) => row.member.id
  );
  const members = query.isSuccess ? query.data : undefined;
  const membersById = query.byId;
  const memberById = query.get;
  const retry = async () => {
    if (permissions.membershipQuery.isError) {
      await permissions.membershipQuery.refetch();
      return;
    }
    if (query.isError) await query.refetch();
  };
  return {
    query,
    members,
    membersById,
    memberById,
    isLoading: permissions.isPending || query.isLoading,
    isError: permissions.isError || query.isError,
    retry,
  };
}

export function useProjectMemberRoster(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  { canLoad = true }: { canLoad?: boolean } = {}
) {
  const { data: user } = useCurrentUser();
  const permissions = useProjectPermissions(workspaceSlug, projectId);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const query = useRecordCollection(
    projectMemberRosterOptions(
      {
        accountId: user?.id,
        workspaceSlug,
        projectId,
        membershipRole: permissions.workspaceRole,
        membershipRevision: membership?.updated_at,
        projectRole: permissions.projectRole,
      },
      canLoad
    ),
    (row) => row.member
  );
  const memberships = canLoad && query.isSuccess ? query.data : undefined;
  const membershipsByMemberId = canLoad ? query.byId : undefined;
  const memberIds = memberships ? new Set(memberships.map((row) => row.member)) : undefined;
  const retry = async () => {
    if (permissions.membershipQuery.isError) {
      await permissions.membershipQuery.refetch();
      return;
    }
    if (permissions.rolesQuery.isError) {
      await permissions.rolesQuery.refetch();
      return;
    }
    if (canLoad && query.isError) await query.refetch();
  };
  return {
    query,
    memberships,
    membershipsByMemberId,
    memberIds,
    isLoading: permissions.isPending || (canLoad && query.isLoading),
    isError: permissions.isError || (canLoad && query.isError),
    retry,
  };
}
