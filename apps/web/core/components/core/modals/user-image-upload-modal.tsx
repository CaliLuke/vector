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
const upload = async (file: File, signal: AbortSignal) => {
  const asset = await fileService.uploadUserAsset(
    { entity_identifier: "", entity_type: EFileAssetType.USER_AVATAR },
    file,
    signal
  );
  return asset.asset_url;
};
const remove = async (url: string) => {
  if (checkURLValidity(url)) await fileService.deleteOldUserAsset(url);
  else await fileService.deleteUserAsset(getAssetIdFromUrl(url));
};

export function UserImageUploadModal({ isOpen, ...props }: ImageUploadModalProps & { isOpen: boolean }) {
  return isOpen ? <ImageUploadModal {...props} upload={upload} remove={remove} /> : null;
}
