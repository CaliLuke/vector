/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// helpers
import { STICKIES_PER_PAGE, API_BASE_URL } from "@plane/constants";
import type { TSticky, TPaginationInfo } from "@plane/types";
// services
import { APIService } from "@/services/api.service";

export class StickyService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async createSticky(workspaceSlug: string, payload: Partial<TSticky>): Promise<TSticky> {
    return this.post(`/api/workspaces/${workspaceSlug}/stickies/`, payload)
      .then((res) => res?.data)
      .catch((err) => {
        throw err?.response?.data ?? err;
      });
  }

  async getStickies(
    workspaceSlug: string,
    cursor: string,
    query?: string,
    per_page?: number,
    signal?: AbortSignal
  ): Promise<TPaginationInfo & { results: TSticky[] }> {
    return this.get(`/api/workspaces/${workspaceSlug}/stickies/`, {
      signal,
      params: {
        cursor,
        per_page: per_page || STICKIES_PER_PAGE,
        query,
      },
    })
      .then((res) => res?.data)
      .catch((err) => {
        throw err?.response?.data ?? err;
      });
  }

  async getSticky(workspaceSlug: string, id: string) {
    return this.get(`/api/workspaces/${workspaceSlug}/stickies/${id}`)
      .then((res) => res?.data)
      .catch((err) => {
        throw err?.response?.data ?? err;
      });
  }

  async updateSticky(workspaceSlug: string, id: string, data: Partial<TSticky>): Promise<TSticky> {
    return await this.patch(`/api/workspaces/${workspaceSlug}/stickies/${id}/`, data)
      .then((res) => res?.data)
      .catch((err) => {
        throw err?.response?.data ?? err;
      });
  }

  async deleteSticky(workspaceSlug: string, id: string) {
    return await this.delete(`/api/workspaces/${workspaceSlug}/stickies/${id}`)
      .then((res) => res?.data)
      .catch((err) => {
        throw err?.response?.data ?? err;
      });
  }
}
