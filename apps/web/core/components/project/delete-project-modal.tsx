/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IProject } from "@plane/types";
import { ConfirmActionModal } from "@/components/common/confirm-action-modal";
import { useProjectMutations } from "@/hooks/use-projects";

type Props = { isOpen: boolean; project: IProject; onClose: () => void };
export function DeleteProjectModal({ isOpen, project, onClose }: Props) {
  const router = useRouter();
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const projectMutation = useProjectMutations(workspaceSlug, project.id);
  return (
    <ConfirmActionModal
      key={project.id}
      isOpen={isOpen}
      title="Delete project"
      name={project.name}
      confirmation="delete my project"
      actionLabel="Delete project"
      pendingLabel="Deleting"
      errorMessage="Unable to delete this project. Please try again."
      description={
        <>
          Are you sure you want to delete <strong>{project.name}</strong>? All data related to this project will be
          permanently removed. This action cannot be undone.
        </>
      }
      onClose={onClose}
      onConfirm={async () => {
        if (!workspaceSlug) throw new Error("Workspace not found");
        await projectMutation.mutateAsync({ type: "delete", projectId: project.id });
      }}
      onConfirmed={() => {
        setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Project deleted successfully." });
        if (projectId === project.id) void router.navigate({ href: `/${workspaceSlug}/projects` });
      }}
    />
  );
}
