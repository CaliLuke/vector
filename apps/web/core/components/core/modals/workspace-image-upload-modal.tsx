/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { EFileAssetType } from "@plane/types";
import { checkURLValidity } from "@plane/utils/string";
import { getAssetIdFromUrl } from "@plane/utils/file";
import { FileService } from "@/services/file.service";
import { ImageUploadModal } from "./image-upload-modal";
import type { ImageUploadModalProps } from "./image-upload-modal";

const fileService = new FileService();

type Props = ImageUploadModalProps & {
  isOpen: boolean;
  workspaceId: string;
  workspaceSlug: string;
};
export function WorkspaceImageUploadModal({ isOpen, workspaceId, workspaceSlug, ...props }: Props) {
  if (!isOpen) return null;
  return (
    <ImageUploadModal
      key={workspaceId}
      {...props}
      upload={async (file, signal) => {
        const asset = await fileService.uploadWorkspaceAsset(
          workspaceSlug,
          { entity_identifier: workspaceId, entity_type: EFileAssetType.WORKSPACE_LOGO },
          file,
          undefined,
          signal
        );
        return asset.asset_url;
      }}
      remove={async (url) => {
        if (checkURLValidity(url)) await fileService.deleteOldWorkspaceAsset(workspaceId, url);
        else await fileService.deleteWorkspaceAsset(workspaceSlug, getAssetIdFromUrl(url));
      }}
    />
  );
}
