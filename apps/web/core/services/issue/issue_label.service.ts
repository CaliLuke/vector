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
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/labels/`, {}, { signal }));
  }

  async getProjectLabels(workspaceSlug: string, projectId: string, signal?: AbortSignal): Promise<IIssueLabel[]> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issue-labels/`, {}, { signal })
    );
  }

  async createIssueLabel(workspaceSlug: string, projectId: string, data: Partial<IIssueLabel>): Promise<IIssueLabel> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issue-labels/`, data));
  }

  async patchIssueLabel(
    workspaceSlug: string,
    projectId: string,
    labelId: string,
    data: Partial<IIssueLabel>
  ): Promise<IIssueLabel> {
    return this.responseData(
      this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issue-labels/${labelId}/`, data)
    );
  }

  async deleteIssueLabel(workspaceSlug: string, projectId: string, labelId: string): Promise<unknown> {
    return this.responseData(
      this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issue-labels/${labelId}/`)
    );
  }
}
