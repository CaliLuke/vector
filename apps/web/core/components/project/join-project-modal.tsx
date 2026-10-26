/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
// types
import { Button } from "@plane/propel/button";
import type { IProject } from "@plane/types";
// ui
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
// hooks
import { useJoinProject } from "@/hooks/use-permissions";

// type
type TJoinProjectModalProps = {
  isOpen: boolean;
  workspaceSlug: string;
  project: IProject;
  handleClose: () => void;
};

export function JoinProjectModal(props: TJoinProjectModalProps) {
  const { handleClose, isOpen, project, workspaceSlug } = props;
  const joinProject = useJoinProject(workspaceSlug, project.id);
  const router = useRouter();
  const handleJoin = () => {
    joinProject.mutate(undefined, {
      onSuccess: () => {
        void router.navigate({ href: `/${workspaceSlug}/projects/${project.id}/issues` });
        handleClose();
      },
    });
  };

  return (
    <ModalCore isOpen={isOpen} handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.XL}>
      <div className="space-y-5 px-5 py-8 sm:p-6">
        <h3 className="text-16 leading-6 font-medium text-primary">Join Project?</h3>
        <p>
          Are you sure you want to join the project <span className="font-semibold break-words">{project?.name}</span>?
          Please click the &apos;Join Project&apos; button below to continue.
        </p>
        {joinProject.isError && (
          <p role="alert" className="text-danger-primary">
            Unable to join this project. Please try again.
          </p>
        )}
      </div>
      <div className="mt-5 flex justify-end gap-2 px-5 pb-8 sm:px-6 sm:pb-6">
        <Button variant="secondary" size="lg" onClick={handleClose}>
          Cancel
        </Button>
        <Button
          variant="primary"
          size="lg"
          type="submit"
          onClick={handleJoin}
          loading={joinProject.isPending}
          disabled={joinProject.isPending}
        >
          {joinProject.isPending ? "Joining..." : "Join Project"}
        </Button>
      </div>
    </ModalCore>
  );
}
