/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { AxiosRequestConfig } from "axios";
// services
import { APIService } from "@/services/api.service";

export class FileUploadService extends APIService {
  constructor() {
    super("");
  }

  async uploadFile(
    url: string,
    data: FormData,
    uploadProgressHandler?: AxiosRequestConfig["onUploadProgress"],
    signal?: AbortSignal
  ): Promise<void> {
    signal?.throwIfAborted();
    await this.post(url, data, {
      headers: { "Content-Type": "multipart/form-data" },
      signal,
      withCredentials: false,
      onUploadProgress: uploadProgressHandler,
    });
  }
}
