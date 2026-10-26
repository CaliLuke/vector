/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import { EIssueServiceType } from "@plane/types";
import type { TIssueCommentReaction, TIssueReaction, TIssueServiceType } from "@plane/types";
// services
import { APIService } from "@/services/api.service";
// types

export class IssueReactionService extends APIService {
  private serviceType: TIssueServiceType;

  constructor(serviceType: TIssueServiceType = EIssueServiceType.ISSUES) {
    super(API_BASE_URL);
    this.serviceType = serviceType;
  }

  async createIssueReaction(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    data: Partial<TIssueReaction>
  ): Promise<TIssueReaction> {
    return this.post(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/reactions/`,
      data
    ).then((response) => response?.data);
  }

  async listIssueReactions(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    signal?: AbortSignal
  ): Promise<TIssueReaction[]> {
    // The parent detail read enforces item visibility, including restricted guests.
    const response = await this.get(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/`,
      {
        signal,
        params: { fields: "id,issue_reactions", expand: "issue_reactions" },
      }
    );
    const item: { id?: string; issue_reactions?: TIssueReaction[] } = response.data;
    if (
      item?.id !== issueId ||
      !Array.isArray(item.issue_reactions) ||
      item.issue_reactions.some((row) => row.issue !== issueId)
    )
      throw new Error("The reaction projection did not match this work item.");
    return item.issue_reactions;
  }

  async deleteIssueReaction(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    reaction: string
  ): Promise<void> {
    return this.delete(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/reactions/${reaction}/`
    ).then((response) => response?.data);
  }

  async createIssueCommentReaction(
    workspaceSlug: string,
    projectId: string,
    commentId: string,
    data: Partial<TIssueCommentReaction>
  ): Promise<TIssueCommentReaction> {
    return this.post(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/comments/${commentId}/reactions/`,
      data
    ).then((response) => response?.data);
  }

  async deleteIssueCommentReaction(
    workspaceSlug: string,
    projectId: string,
    commentId: string,
    reaction: string
  ): Promise<void> {
    return this.delete(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/comments/${commentId}/reactions/${reaction}/`
    ).then((response) => response?.data);
  }
}
