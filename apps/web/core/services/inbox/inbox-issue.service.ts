/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { API_BASE_URL } from "@plane/constants";
import type { TInboxIssue, TIssue, TInboxIssueWithPagination } from "@plane/types";
import { EInboxIssueSource } from "@plane/types";
// helpers
// services
import { APIService } from "@/services/api.service";

export class InboxIssueService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async list(
    workspaceSlug: string,
    projectId: string,
    params = {},
    options?: { signal?: AbortSignal }
  ): Promise<TInboxIssueWithPagination> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/inbox-issues/`, {
      params,
      signal: options?.signal,
    }).then((response) => response?.data);
  }

  async retrieve(
    workspaceSlug: string,
    projectId: string,
    inboxIssueId: string,
    options?: { signal?: AbortSignal }
  ): Promise<TInboxIssue> {
    return this.get(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/inbox-issues/${inboxIssueId}/?expand=issue_inbox`,
      { signal: options?.signal }
    ).then((response) => response?.data);
  }

  async create(workspaceSlug: string, projectId: string, data: Partial<TIssue>): Promise<TInboxIssue> {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/inbox-issues/`, {
      source: EInboxIssueSource.IN_APP,
      issue: data,
    }).then((response) => response?.data);
  }

  async update(
    workspaceSlug: string,
    projectId: string,
    inboxIssueId: string,
    data: Partial<TInboxIssue>
  ): Promise<TInboxIssue> {
    return this.patch(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/inbox-issues/${inboxIssueId}/`,
      data
    ).then((response) => response?.data);
  }

  async updateIssue(
    workspaceSlug: string,
    projectId: string,
    inboxIssueId: string,
    data: Partial<TIssue>
  ): Promise<TInboxIssue> {
    return this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/inbox-issues/${inboxIssueId}/`, {
      issue: data,
    }).then((response) => response?.data);
  }

  async destroy(workspaceSlug: string, projectId: string, inboxIssueId: string): Promise<void> {
    return this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/inbox-issues/${inboxIssueId}/`).then(
      (response) => response?.data
    );
  }
}
