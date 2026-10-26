/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
// plane imports
import { Table } from "@plane/ui";
// hooks
import { useQueryClient } from "@tanstack/react-query";
import { captureIdentityScope, isIdentityScopeCurrent } from "@/queries/identity";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useLeaveProject, useProjectPermissions } from "@/hooks/use-permissions";
import { useRemoveProjectMember } from "@/hooks/use-member-roster-mutations";

// plane web imports
import { useProjectColumns } from "@/components/projects/settings/useProjectColumns";
// store
import type { RowData } from "@/components/project/settings/member-columns";
import type { IMemberFilters } from "@/helpers/member-ordering";
// local imports
import { ConfirmProjectMemberRemove } from "./confirm-project-member-remove";

type Props = {
  memberDetails: RowData[];
  projectId: string;
  workspaceSlug: string;
  displayFilters: IMemberFilters;
  handleDisplayFilterUpdate: (filters: Partial<IMemberFilters>) => void;
};

export function ProjectMemberListItem(props: Props) {
  const { memberDetails, projectId, workspaceSlug, displayFilters, handleDisplayFilterUpdate } = props;
  // router
  const router = useRouter();
  // store hooks
  const leaveProject = useLeaveProject(workspaceSlug, projectId);
  const { projectRole } = useProjectPermissions(workspaceSlug, projectId);
  const client = useQueryClient();
  const { data: currentUser } = useCurrentUser();
  // helper hooks
  const { columns, removeMemberModal, setRemoveMemberModal } = useProjectColumns({
    projectId,
    workspaceSlug,
    displayFilters,
    handleDisplayFilterUpdate,
  });
  const removeMember = useRemoveProjectMember(
    workspaceSlug,
    projectId,
    removeMemberModal?.member.id ?? "",
    projectRole
  );

  const handleRemove = async (memberId: string) => {
    if (!workspaceSlug || !projectId || !memberId) throw new Error("A workspace, project and member are required.");

    if (memberId === currentUser?.id) {
      const identity = captureIdentityScope(client, currentUser.id);
      const pathname = router.state.location.pathname;
      await leaveProject.mutateAsync();
      if (isIdentityScopeCurrent(client, identity) && router.state.location.pathname === pathname)
        void router.navigate({ href: `/${workspaceSlug}/projects` });
    } else await removeMember.mutateAsync();
  };

  if (!memberDetails) return null;
  return (
    <>
      {removeMemberModal && (
        <ConfirmProjectMemberRemove
          isOpen={removeMemberModal !== null}
          onClose={() => setRemoveMemberModal(null)}
          data={{ id: removeMemberModal.member.id, display_name: removeMemberModal.member.display_name || "" }}
          onSubmit={() => handleRemove(removeMemberModal.member.id)}
        />
      )}
      <Table
        columns={columns}
        data={memberDetails}
        keyExtractor={(rowData) => rowData?.member.id ?? ""}
        tHeadClassName="border-b border-subtle"
        thClassName="text-left font-medium divide-x-0 text-placeholder"
        tBodyClassName="divide-y-0"
        tBodyTrClassName="divide-x-0 p-4 h-[40px] text-secondary"
        tHeadTrClassName="divide-x-0"
      />
    </>
  );
}
