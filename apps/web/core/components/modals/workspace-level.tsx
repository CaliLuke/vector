/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { CreateProjectModal } from "@/components/project/create-project-modal";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";

export type TWorkspaceLevelModalsProps = {
  workspaceSlug: string;
};

export function WorkspaceLevelModals(props: TWorkspaceLevelModalsProps) {
  const { workspaceSlug } = props;
  // store hooks
  const { isCreateProjectModalOpen, toggleCreateProjectModal } = useCommandPalette();

  return (
    <>
      <CreateProjectModal
        isOpen={isCreateProjectModalOpen}
        onClose={() => toggleCreateProjectModal(false)}
        workspaceSlug={workspaceSlug.toString()}
      />
    </>
  );
}
