/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { AxiosRequestConfig } from "axios";
// plane types
import { API_BASE_URL } from "@plane/constants";
import { getFileMetaDataForUpload, generateFileUploadPayload } from "@plane/services";
import type { EFileAssetType, TFileEntityInfo, TFileSignedURLResponse } from "@plane/types";
import { getAssetIdFromUrl } from "@plane/utils/file";
// helpers
// services
import { authService } from "./auth.service";
import { APIService } from "@/services/api.service";
import { FileUploadService } from "@/services/file-upload.service";

export interface UnSplashImage {
  id: string;
  created_at: Date;
  updated_at: Date;
  promoted_at: Date;
  width: number;
  height: number;
  color: string;
  blur_hash: string;
  description: null;
  alt_description: string;
  urls: UnSplashImageUrls;
  [key: string]: any;
}

interface UnSplashImageUrls {
  raw: string;
  full: string;
  regular: string;
  small: string;
  thumb: string;
  small_s3: string;
}

export enum TFileAssetType {
  COMMENT_DESCRIPTION = "COMMENT_DESCRIPTION",
  ISSUE_ATTACHMENT = "ISSUE_ATTACHMENT",
  ISSUE_DESCRIPTION = "ISSUE_DESCRIPTION",
  PAGE_DESCRIPTION = "PAGE_DESCRIPTION",
  PROJECT_COVER = "PROJECT_COVER",
  USER_AVATAR = "USER_AVATAR",
  USER_COVER = "USER_COVER",
  WORKSPACE_LOGO = "WORKSPACE_LOGO",
}

export class FileService extends APIService {
  private fileUploadService: FileUploadService;

  constructor() {
    super(API_BASE_URL);
    // upload service
    this.fileUploadService = new FileUploadService();
  }

  private async uploadAsset(
    path: string,
    data: TFileEntityInfo,
    file: File,
    uploadProgressHandler?: AxiosRequestConfig["onUploadProgress"],
    signal?: AbortSignal
  ): Promise<TFileSignedURLResponse> {
    signal?.throwIfAborted();
    const fileMetaData = await getFileMetaDataForUpload(file);
    signal?.throwIfAborted();
    const response = await this.post(path, { ...data, ...fileMetaData }, { signal });
    const asset: TFileSignedURLResponse = response.data;
    await this.fileUploadService.uploadFile(
      asset.upload_data.url,
      generateFileUploadPayload(asset, file),
      uploadProgressHandler,
      signal
    );
    signal?.throwIfAborted();
    await this.patch(`${path}${asset.asset_id}/`, undefined, { signal });
    return asset;
  }

  async uploadWorkspaceAsset(
    workspaceSlug: string,
    data: TFileEntityInfo,
    file: File,
    uploadProgressHandler?: AxiosRequestConfig["onUploadProgress"],
    signal?: AbortSignal
  ): Promise<TFileSignedURLResponse> {
    return this.uploadAsset(`/api/assets/v2/workspaces/${workspaceSlug}/`, data, file, uploadProgressHandler, signal);
  }

  async deleteWorkspaceAsset(workspaceSlug: string, assetId: string): Promise<void> {
    return this.delete(`/api/assets/v2/workspaces/${workspaceSlug}/${assetId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async updateBulkWorkspaceAssetsUploadStatus(
    workspaceSlug: string,
    entityId: string,
    data: {
      asset_ids: string[];
    }
  ): Promise<void> {
    const { csrf_token } = await authService.requestCSRFToken();
    if (!csrf_token) throw new Error("Could not verify this session. Reload and try again.");
    return this.post(`/api/assets/v2/workspaces/${workspaceSlug}/${entityId}/bulk/`, data, {
      headers: { "X-CSRFTOKEN": csrf_token },
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async updateBulkProjectAssetsUploadStatus(
    workspaceSlug: string,
    projectId: string,
    entityId: string,
    data: {
      asset_ids: string[];
    }
  ): Promise<void> {
    return this.post(`/api/assets/v2/workspaces/${workspaceSlug}/projects/${projectId}/${entityId}/bulk/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async uploadProjectAsset(
    workspaceSlug: string,
    projectId: string,
    data: TFileEntityInfo,
    file: File,
    uploadProgressHandler?: AxiosRequestConfig["onUploadProgress"],
    signal?: AbortSignal
  ): Promise<TFileSignedURLResponse> {
    return this.uploadAsset(
      `/api/assets/v2/workspaces/${workspaceSlug}/projects/${projectId}/`,
      data,
      file,
      uploadProgressHandler,
      signal
    );
  }

  async uploadUserAsset(data: TFileEntityInfo, file: File, signal?: AbortSignal): Promise<TFileSignedURLResponse> {
    return this.uploadAsset("/api/assets/v2/user-assets/", data, file, undefined, signal);
  }

  async deleteUserAsset(assetId: string): Promise<void> {
    return this.delete(`/api/assets/v2/user-assets/${assetId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async deleteNewAsset(assetPath: string): Promise<void> {
    return this.delete(assetPath)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async deleteOldWorkspaceAsset(workspaceId: string, src: string): Promise<any> {
    const assetKey = getAssetIdFromUrl(src);
    return this.delete(`/api/workspaces/file-assets/${workspaceId}/${assetKey}/`)
      .then((response) => response?.status)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async deleteOldUserAsset(src: string): Promise<any> {
    const assetKey = getAssetIdFromUrl(src);
    return this.delete(`/api/users/file-assets/${assetKey}/`)
      .then((response) => response?.status)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async restoreNewAsset(workspaceSlug: string, src: string): Promise<void> {
    // remove the last slash and get the asset id
    const assetId = getAssetIdFromUrl(src);
    return this.post(`/api/assets/v2/workspaces/${workspaceSlug}/restore/${assetId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async checkIfAssetExists(
    workspaceSlug: string,
    assetId: string
  ): Promise<{
    exists: boolean;
  }> {
    return this.get(`/api/assets/v2/workspaces/${workspaceSlug}/check/${assetId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async restoreOldEditorAsset(workspaceId: string, src: string): Promise<void> {
    const assetKey = getAssetIdFromUrl(src);
    return this.post(`/api/workspaces/file-assets/${workspaceId}/${assetKey}/restore/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async getUnsplashImages(query?: string, signal?: AbortSignal): Promise<UnSplashImage[]> {
    return this.get(`/api/unsplash/`, {
      signal,
      params: {
        query,
      },
    })
      .then((res) => res?.data?.results ?? res?.data)
      .catch((err) => {
        throw err?.response?.data ?? err;
      });
  }

  async duplicateAsset(
    workspaceSlug: string,
    assetId: string,
    data: {
      entity_id?: string;
      entity_type: EFileAssetType;
      project_id?: string;
    },
    signal?: AbortSignal
  ): Promise<{ asset_id: string }> {
    return this.post(`/api/assets/v2/workspaces/${workspaceSlug}/duplicate-assets/${assetId}/`, data, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }
}
