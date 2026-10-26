/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane types
import { API_BASE_URL } from "@plane/constants";
import type { TIssueComment, TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// services
import { APIService } from "@/services/api.service";

export class IssueCommentService extends APIService {
  private serviceType: TIssueServiceType;

  constructor(serviceType: TIssueServiceType = EIssueServiceType.ISSUES) {
    super(API_BASE_URL);
    this.serviceType = serviceType;
  }

  async getIssueComments(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    signal?: AbortSignal
  ): Promise<TIssueComment[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/history/`, {
      signal,
      params: {
        activity_type: this.serviceType === EIssueServiceType.EPICS ? "epic-comment" : "issue-comment",
      },
    }).then((response) => response?.data);
  }

  async createIssueComment(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    data: Partial<TIssueComment>
  ): Promise<TIssueComment> {
    return this.post(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/comments/`,
      data
    ).then((response) => response?.data);
  }

  async patchIssueComment(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    commentId: string,
    data: Partial<TIssueComment>
  ): Promise<TIssueComment> {
    return this.patch(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/comments/${commentId}/`,
      data
    ).then((response) => response?.data);
  }

  async deleteIssueComment(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    commentId: string
  ): Promise<void> {
    return this.delete(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/comments/${commentId}/`
    ).then((response) => response?.data);
  }
}
