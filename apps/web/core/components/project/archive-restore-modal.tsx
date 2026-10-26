/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
import { useState } from "react";
// ui
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
// hooks
import { useProjectDetail, useProjectMutations } from "@/hooks/use-projects";

type Props = {
  workspaceSlug: string;

  projectId: string;
  isOpen: boolean;
  onClose: () => void;
  archive: boolean;
};

export function ArchiveRestoreProjectModal(props: Props) {
  const { workspaceSlug, projectId, isOpen, onClose, archive } = props;
  // router
  const router = useRouter();
  // states
  const [isLoading, setIsLoading] = useState(false);
  // store hooks
  const projectQuery = useProjectDetail(workspaceSlug, projectId);
  const projectDetails = projectQuery.project;
  const projectMutation = useProjectMutations(workspaceSlug, projectId);
  if (!projectDetails) return null;

  const handleClose = () => {
    setIsLoading(false);
    onClose();
  };

  const handleArchiveProject = async () => {
    setIsLoading(true);
    await projectMutation
      .mutateAsync({ type: "archive", projectId })
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Archive success",
          message: `${projectDetails.name} has been archived successfully`,
        });
        onClose();
        void router.navigate({ href: `/${workspaceSlug}/projects/` });
        return;
      })
      .catch(() =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "Project could not be archived. Please try again.",
        })
      )
      .finally(() => setIsLoading(false));
  };

  const handleRestoreProject = async () => {
    setIsLoading(true);
    await projectMutation
      .mutateAsync({ type: "restore", projectId })
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Restore success",
          message: `You can find ${projectDetails.name} in your projects.`,
        });
        onClose();
        void router.navigate({ href: `/${workspaceSlug}/projects/` });
        return;
      })
      .catch(() =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "Project could not be restored. Please try again.",
        })
      )
      .finally(() => setIsLoading(false));
  };

  return (
    <ModalCore isOpen={isOpen} handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.LG}>
      <div className="px-5 py-4">
        <h3 className="text-18 font-medium 2xl:text-20">
          {archive ? "Archive" : "Restore"} {projectDetails.name}
        </h3>
        <p className="mt-3 text-13 text-secondary">
          {archive
            ? "This project and its work items, cycles, modules, and pages will be archived. Its work items won't appear in search. Only project admins can restore the project."
            : "Restoring a project will activate it and make it visible to all members of the project. Are you sure you want to continue?"}
        </p>
        <div className="mt-3 flex justify-end gap-2">
          <Button variant="secondary" size="lg" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="primary"
            size="lg"
            onClick={archive ? handleArchiveProject : handleRestoreProject}
            loading={isLoading}
          >
            {archive ? (isLoading ? "Archiving" : "Archive") : isLoading ? "Restoring" : "Restore"}
          </Button>
        </div>
      </div>
    </ModalCore>
  );
}
