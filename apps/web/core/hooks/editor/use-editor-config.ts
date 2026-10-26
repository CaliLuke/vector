/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { TFileHandler } from "@plane/editor";
import { EFileAssetType } from "@plane/types";
import type { TFileEntityInfo } from "@plane/types";
import { getEditorAssetDownloadSrc, getEditorAssetSrc } from "@plane/utils/editor/common";
// hooks
import { useEditorUploads } from "./use-editor-uploads";
// plane web hooks
import { useExtendedEditorConfig } from "@/hooks/editor/use-extended-editor-config";
import { useFileSize } from "@/hooks/use-file-size";
// services
import { FileService } from "@/services/file.service";
const fileService = new FileService();

type TArgs = {
  projectId?: string;
  fileAsset?: TFileEntityInfo;
  onAssetUpload?: (assetId: string) => void;
  workspaceId: string;
  workspaceSlug: string;
};

export const useEditorConfig = (args: TArgs): TFileHandler => {
  const { projectId, fileAsset, onAssetUpload, workspaceId, workspaceSlug } = args;
  const assetProjectId = fileAsset?.entity_type === EFileAssetType.DRAFT_ISSUE_DESCRIPTION ? undefined : projectId;
  const scopeKey = JSON.stringify([
    workspaceSlug,
    assetProjectId,
    fileAsset?.entity_type,
    fileAsset?.entity_identifier,
  ]);
  const { progress, uploadAsset, duplicateAsset, cancelUploads } = useEditorUploads(scopeKey);
  const { maxFileSize } = useFileSize();
  const { getExtendedEditorFileHandlers } = useExtendedEditorConfig();

  return {
    assetsUploadStatus: progress,
    cancel: cancelUploads,
    checkIfAssetExists: async (assetId: string) => {
      const res = await fileService.checkIfAssetExists(workspaceSlug, assetId);
      return res?.exists ?? false;
    },
    delete: async (src: string) => {
      if (src?.startsWith("http")) {
        await fileService.deleteOldWorkspaceAsset(workspaceId, src);
      } else {
        await fileService.deleteNewAsset(
          getEditorAssetSrc({
            assetId: src,
            projectId: assetProjectId,
            workspaceSlug,
          }) ?? ""
        );
      }
    },
    getAssetDownloadSrc: async (path) => {
      if (!path) return "";
      if (path?.startsWith("http")) {
        return path;
      } else {
        return (
          getEditorAssetDownloadSrc({
            assetId: path,
            projectId: assetProjectId,
            workspaceSlug,
          }) ?? ""
        );
      }
    },
    getAssetSrc: async (path) => {
      if (!path) return "";
      if (path?.startsWith("http")) {
        return path;
      } else {
        return (
          getEditorAssetSrc({
            assetId: path,
            projectId: assetProjectId,
            workspaceSlug,
          }) ?? ""
        );
      }
    },
    restore: async (src: string) => {
      if (src?.startsWith("http")) {
        await fileService.restoreOldEditorAsset(workspaceId, src);
      } else {
        await fileService.restoreNewAsset(workspaceSlug, src);
      }
    },
    upload: async (blockId, file) => {
      if (!fileAsset) throw new Error("Uploads are unavailable in this editor");
      const assetId = await uploadAsset({ blockId, file, data: fileAsset, projectId: assetProjectId, workspaceSlug });
      onAssetUpload?.(assetId);
      return assetId;
    },
    duplicate: async (assetId) => {
      if (!fileAsset) throw new Error("Asset duplication is unavailable in this editor");
      const newAssetId = await duplicateAsset({ assetId, data: fileAsset, projectId: assetProjectId, workspaceSlug });
      onAssetUpload?.(newAssetId);
      return newAssetId;
    },
    validation: {
      maxFileSize,
    },
    ...getExtendedEditorFileHandlers({ projectId, workspaceSlug }),
  };
};
