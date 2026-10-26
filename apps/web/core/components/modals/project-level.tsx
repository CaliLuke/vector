/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { CycleCreateUpdateModal } from "@/components/cycles/modal";
import { CreateUpdateModuleModal } from "@/components/modules";
import { CreatePageModal } from "@/components/pages/modals/create-page-modal";
import { CreateUpdateProjectViewModal } from "@/components/views/modal";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";

export type TProjectLevelModalsProps = {
  workspaceSlug: string;
  projectId: string;
};

export function ProjectLevelModals(props: TProjectLevelModalsProps) {
  const { workspaceSlug, projectId } = props;
  // store hooks
  const {
    isCreateCycleModalOpen,
    toggleCreateCycleModal,
    isCreateModuleModalOpen,
    toggleCreateModuleModal,
    isCreateViewModalOpen,
    toggleCreateViewModal,
    createPageModal,
    toggleCreatePageModal,
  } = useCommandPalette();

  return (
    <>
      <CycleCreateUpdateModal
        isOpen={isCreateCycleModalOpen}
        handleClose={() => toggleCreateCycleModal(false)}
        workspaceSlug={workspaceSlug.toString()}
        projectId={projectId.toString()}
      />
      <CreateUpdateModuleModal
        isOpen={isCreateModuleModalOpen}
        onClose={() => toggleCreateModuleModal(false)}
        workspaceSlug={workspaceSlug.toString()}
        projectId={projectId.toString()}
      />
      <CreateUpdateProjectViewModal
        isOpen={isCreateViewModalOpen}
        onClose={() => toggleCreateViewModal(false)}
        workspaceSlug={workspaceSlug.toString()}
        projectId={projectId.toString()}
      />
      <CreatePageModal
        workspaceSlug={workspaceSlug.toString()}
        projectId={projectId.toString()}
        isModalOpen={createPageModal.isOpen}
        pageAccess={createPageModal.pageAccess}
        handleModalClose={() => toggleCreatePageModal({ isOpen: false })}
        redirectionEnabled
      />
    </>
  );
}
