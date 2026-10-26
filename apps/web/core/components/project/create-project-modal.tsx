/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
import { getAssetIdFromUrl } from "@plane/utils/file";
import { checkURLValidity } from "@plane/utils/string";
// plane ui
// helpers
// hooks

// plane web components
import { CreateProjectForm } from "@/components/projects/create/root";
// plane web types
import type { TProject } from "@plane/types";
// services
import { FileService } from "@/services/file.service";
const fileService = new FileService();
import { ProjectFeatureUpdate } from "./project-feature-update";

type Props = {
  isOpen: boolean;
  onClose: () => void;
  setToFavorite?: boolean;
  workspaceSlug: string;
  data?: Partial<TProject>;
  templateId?: string;
};

export function CreateProjectModal(props: Props) {
  return props.isOpen ? <CreateProjectFlow key={props.workspaceSlug} {...props} /> : null;
}
function CreateProjectFlow({ onClose, setToFavorite = false, workspaceSlug, data }: Props) {
  const [createdProjectId, setCreatedProjectId] = useState<string | null>(null);
  const handleCoverImageStatusUpdate = async (projectId: string, coverImage: string) => {
    if (!checkURLValidity(coverImage)) {
      await fileService.updateBulkProjectAssetsUploadStatus(workspaceSlug, projectId, projectId, {
        asset_ids: [getAssetIdFromUrl(coverImage)],
      });
    }
  };

  return createdProjectId ? (
    <ModalCore isOpen handleClose={onClose} position={EModalPosition.TOP} width={EModalWidth.XXXXL}>
      <ProjectFeatureUpdate projectId={createdProjectId} workspaceSlug={workspaceSlug} onClose={onClose} />
    </ModalCore>
  ) : (
    <CreateProjectForm
      setToFavorite={setToFavorite}
      workspaceSlug={workspaceSlug}
      onClose={onClose}
      updateCoverImageStatus={handleCoverImageStatusUpdate}
      handleNextStep={setCreatedProjectId}
      data={data}
    />
  );
}
