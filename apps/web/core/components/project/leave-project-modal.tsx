/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import type { IProject } from "@plane/types";
import { ConfirmActionModal } from "@/components/common/confirm-action-modal";
import { useLeaveProject } from "@/hooks/use-permissions";
import { useQueryClient } from "@tanstack/react-query";
import { useCurrentUser } from "@/hooks/use-current-user";
import { captureIdentityScope, isIdentityScopeCurrent } from "@/queries/identity";

type Props = { isOpen: boolean; project: IProject; onClose: () => void };
export function LeaveProjectModal({ isOpen, project, onClose }: Props) {
  const router = useRouter();
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const leaveProject = useLeaveProject(workspaceSlug, project.id);
  const client = useQueryClient();
  const { data: user } = useCurrentUser();
  return (
    <ConfirmActionModal
      key={project.id}
      isOpen={isOpen}
      title="Leave Project"
      name={project.name}
      confirmation="Leave Project"
      actionLabel="Leave Project"
      pendingLabel="Leaving..."
      errorMessage="Unable to leave this project. Please try again."
      description={
        <>
          Are you sure you want to leave <strong>{project.name}</strong>? All work items associated with you in this
          project will become inaccessible.
        </>
      }
      onClose={onClose}
      onConfirm={async () => {
        const identity = captureIdentityScope(client, user?.id);
        const pathname = router.state.location.pathname;
        await leaveProject.mutateAsync();
        if (isIdentityScopeCurrent(client, identity) && router.state.location.pathname === pathname)
          void router.navigate({ href: `/${workspaceSlug}/projects` });
      }}
    />
  );
}
