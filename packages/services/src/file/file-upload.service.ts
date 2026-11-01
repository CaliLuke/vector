/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { isAxiosError } from "axios";
// api service
import { APIService } from "../api.service";

/**
 * Service class for handling file upload operations
 * Handles file uploads
 * @extends {APIService}
 */
export class FileUploadService extends APIService {
  private uploads = new Set<AbortController>();

  constructor() {
    super("");
  }

  /**
   * Uploads a file to the specified signed URL
   * @param {string} url - The URL to upload the file to
   * @param {FormData} data - The form data to upload
   * @returns {Promise<void>} Promise resolving to void
   * @throws {Error} If the request fails
   */
  async uploadFile(url: string, data: FormData, signal?: AbortSignal): Promise<void> {
    const controller = new AbortController();
    const uploadSignal = signal ? AbortSignal.any([signal, controller.signal]) : controller.signal;
    uploadSignal.throwIfAborted();
    this.uploads.add(controller);
    try {
      await this.post(url, data, {
        headers: { "Content-Type": "multipart/form-data" },
        signal: uploadSignal,
        withCredentials: false,
      });
      uploadSignal.throwIfAborted();
    } catch (error) {
      throw isAxiosError(error) ? (error.response?.data ?? error) : error;
    } finally {
      this.uploads.delete(controller);
    }
  }

  /**
   * Cancels the upload
   */
  cancelUpload() {
    for (const controller of this.uploads) controller.abort();
  }
}
