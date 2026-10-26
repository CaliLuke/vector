/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type { TIssue, TWorkspaceDraftIssue, TWorkspaceDraftPaginationInfo } from "@plane/types";
// helpers
// services
import { APIService } from "@/services/api.service";

export class WorkspaceDraftService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async getIssues(
    workspaceSlug: string,
    query: object = {},
    signal?: AbortSignal
  ): Promise<TWorkspaceDraftPaginationInfo<TWorkspaceDraftIssue> | undefined> {
    return this.get(`/api/workspaces/${workspaceSlug}/draft-issues/`, { params: { ...query }, signal }).then(
      (response) => response?.data
    );
  }

  async getIssueById(
    workspaceSlug: string,
    issueId: string,
    signal?: AbortSignal
  ): Promise<TWorkspaceDraftIssue | undefined> {
    return this.get(`/api/workspaces/${workspaceSlug}/draft-issues/${issueId}/`, { signal }).then(
      (response) => response?.data
    );
  }

  async createIssue(
    workspaceSlug: string,
    payload: Partial<TWorkspaceDraftIssue | TIssue>
  ): Promise<TWorkspaceDraftIssue | undefined> {
    return this.post(`/api/workspaces/${workspaceSlug}/draft-issues/`, payload).then((response) => response?.data);
  }

  async updateIssue(
    workspaceSlug: string,
    issueId: string,
    payload: Partial<TWorkspaceDraftIssue | TIssue>
  ): Promise<TWorkspaceDraftIssue | undefined> {
    return this.patch(`/api/workspaces/${workspaceSlug}/draft-issues/${issueId}/`, payload).then(
      (response) => response?.data
    );
  }

  async deleteIssue(workspaceSlug: string, issueId: string): Promise<void> {
    return this.delete(`/api/workspaces/${workspaceSlug}/draft-issues/${issueId}/`).then((response) => response?.data);
  }
}
