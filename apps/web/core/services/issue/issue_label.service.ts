/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type { IIssueLabel } from "@plane/types";
// services
import { APIService } from "@/services/api.service";
// types

export class IssueLabelService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async getWorkspaceIssueLabels(workspaceSlug: string, signal?: AbortSignal): Promise<IIssueLabel[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/labels/`, {}, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getProjectLabels(workspaceSlug: string, projectId: string, signal?: AbortSignal): Promise<IIssueLabel[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issue-labels/`, {}, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async createIssueLabel(workspaceSlug: string, projectId: string, data: Partial<IIssueLabel>): Promise<IIssueLabel> {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issue-labels/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async patchIssueLabel(
    workspaceSlug: string,
    projectId: string,
    labelId: string,
    data: Partial<IIssueLabel>
  ): Promise<IIssueLabel> {
    return this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issue-labels/${labelId}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async deleteIssueLabel(workspaceSlug: string, projectId: string, labelId: string): Promise<any> {
    return this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issue-labels/${labelId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
}
