/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { API_BASE_URL } from "@plane/constants";
import { isAxiosError } from "axios";
// local services
import type { TFileEntityInfo, TFileSignedURLResponse } from "@plane/types";
import { FileUploadService } from "./file-upload.service";
// helpers
import { FileService } from "./file.service";
import { generateFileUploadPayload, getAssetIdFromUrl, getFileMetaDataForUpload } from "./helper";

/**
 * Service class for managing file operations within plane sites application.
 * Extends FileService to manage file-related operations.
 * @extends {FileService}
 * @remarks This service is only available for plane sites
 */
export class SitesFileService extends FileService {
  private uploads = new Set<AbortController>();
  fileUploadService: FileUploadService;

  /**
   * Creates an instance of SitesFileService
   * @param {string} BASE_URL - The base URL for API requests
   */
  constructor(BASE_URL?: string) {
    super(BASE_URL || API_BASE_URL);
    this.cancelUpload = this.cancelUpload.bind(this);
    // services
    this.fileUploadService = new FileUploadService();
  }

  /**
   * Updates the upload status of an asset
   * @param {string} anchor - The anchor identifier
   * @param {string} assetId - The asset identifier
   * @returns {Promise<void>} Promise resolving to void
   * @throws {Error} If the request fails
   */
  private async updateAssetUploadStatus(anchor: string, assetId: string, signal: AbortSignal): Promise<void> {
    signal.throwIfAborted();
    return this.responseData(
      this.patch(`/api/public/assets/v2/anchor/${anchor}/${assetId}/`, {}, { signal }),
      "data-or-error"
    );
  }

  /**
   * Updates the upload status of multiple assets
   * @param {string} anchor - The anchor identifier
   * @param {string} entityId - The entity identifier
   * @param {Object} data - The data payload
   * @returns {Promise<void>} Promise resolving to void
   * @throws {Error} If the request fails
   */
  async updateBulkAssetsUploadStatus(
    anchor: string,
    entityId: string,
    data: {
      asset_ids: string[];
    }
  ): Promise<void> {
    return this.responseData(this.post(`/api/public/assets/v2/anchor/${anchor}/${entityId}/bulk/`, data));
  }

  /**
   * Uploads a file to the specified anchor
   * @param {string} anchor - The anchor identifier
   * @param {TFileEntityInfo} data - The data payload
   * @param {File} file - The file to upload
   * @returns {Promise<TFileSignedURLResponse>} Promise resolving to the signed URL response
   * @throws {Error} If the request fails
   */
  async uploadAsset(anchor: string, data: TFileEntityInfo, file: File): Promise<TFileSignedURLResponse> {
    const controller = new AbortController();
    const { signal } = controller;
    this.uploads.add(controller);
    try {
      const fileMetaData = await getFileMetaDataForUpload(file);
      signal.throwIfAborted();
      const response = await this.post(
        `/api/public/assets/v2/anchor/${anchor}/`,
        {
          ...data,
          ...fileMetaData,
        },
        { signal }
      );
      signal.throwIfAborted();
      const signedURLResponse: TFileSignedURLResponse = response.data;
      const fileUploadPayload = generateFileUploadPayload(signedURLResponse, file);
      await this.fileUploadService.uploadFile(signedURLResponse.upload_data.url, fileUploadPayload, signal);
      await this.updateAssetUploadStatus(anchor, signedURLResponse.asset_id, signal);
      return signedURLResponse;
    } catch (error) {
      throw isAxiosError(error) ? (error.response?.data ?? error) : error;
    } finally {
      this.uploads.delete(controller);
    }
  }

  /**
   * Restores a new asset
   * @param {string} workspaceSlug - The workspace slug
   * @param {string} src - The asset source
   * @returns {Promise<void>} Promise resolving to void
   * @throws {Error} If the request fails
   */
  async restoreNewAsset(anchor: string, src: string): Promise<void> {
    // remove the last slash and get the asset id
    const assetId = getAssetIdFromUrl(src);
    return this.responseData(this.post(`/api/public/assets/v2/anchor/${anchor}/restore/${assetId}/`));
  }

  /**
   * Cancels the upload
   */
  cancelUpload() {
    for (const controller of this.uploads) controller.abort();
  }
}
