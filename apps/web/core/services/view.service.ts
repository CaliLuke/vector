/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type { IProjectView } from "@plane/types";
import { APIService } from "@/services/api.service";

export class ViewService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async createView(workspaceSlug: string, projectId: string, data: Partial<IProjectView>): Promise<IProjectView> {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/views/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async patchView(
    workspaceSlug: string,
    projectId: string,
    viewId: string,
    data: Partial<IProjectView>
  ): Promise<IProjectView> {
    return this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/views/${viewId}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async deleteView(workspaceSlug: string, projectId: string, viewId: string): Promise<void> {
    return this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/views/${viewId}/`)
      .then(() => undefined)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getViews(workspaceSlug: string, projectId: string, signal?: AbortSignal): Promise<IProjectView[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/views/`, {}, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getViewDetails(
    workspaceSlug: string,
    projectId: string,
    viewId: string,
    signal?: AbortSignal
  ): Promise<IProjectView> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/views/${viewId}/`, {}, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
}
