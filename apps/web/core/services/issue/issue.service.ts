/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { API_BASE_URL } from "@plane/constants";
import { EIssueServiceType } from "@plane/types";
import type {
  TIssueParams,
  IIssueDisplayProperties,
  TBulkOperationsPayload,
  TIssue,
  TIssueActivity,
  TIssueLink,
  TIssueServiceType,
  TIssuesResponse,
  TIssueSubIssues,
} from "@plane/types";
// services
import { APIService } from "@/services/api.service";

export class IssueService extends APIService {
  private serviceType: TIssueServiceType;

  constructor(serviceType: TIssueServiceType = EIssueServiceType.ISSUES) {
    super(API_BASE_URL);
    this.serviceType = serviceType;
  }

  async createIssue(workspaceSlug: string, projectId: string, data: Partial<TIssue>): Promise<TIssue> {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/`, data)
    );
  }

  async getIssuesFromServer(
    workspaceSlug: string,
    projectId: string,
    queries?: Partial<Record<TIssueParams, string | boolean>>,
    config = {}
  ): Promise<TIssuesResponse> {
    const path =
      typeof queries?.expand === "string" && queries.expand.includes("issue_relation") && !queries.group_by
        ? `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}-detail/`
        : `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/`;
    return this.responseData(
      this.get(
        path,
        {
          params: queries,
        },
        config
      )
    );
  }

  async getIssuesForSync(
    workspaceSlug: string,
    projectId: string,
    queries?: Partial<Record<TIssueParams, string | boolean>>,
    config = {}
  ): Promise<TIssuesResponse> {
    return this.responseData(
      this.get(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/v2/${this.serviceType}/`,
        { params: queries },
        config
      )
    );
  }

  async getIssues(
    workspaceSlug: string,
    projectId: string,
    queries?: Partial<Record<TIssueParams, string | boolean>>,
    config = {}
  ): Promise<TIssuesResponse> {
    return this.getIssuesFromServer(workspaceSlug, projectId, queries, config);
  }

  async getDeletedIssues(
    workspaceSlug: string,
    projectId: string,
    queries?: Partial<Record<TIssueParams, string | boolean>>
  ): Promise<TIssuesResponse> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/deleted-issues/`, {
        params: queries,
      })
    );
  }

  async getIssuesWithParams(
    workspaceSlug: string,
    projectId: string,
    queries?: Partial<Record<TIssueParams, string | boolean>>
  ): Promise<TIssue[] | { [key: string]: TIssue[] }> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/`, {
        params: queries,
      })
    );
  }

  async retrieve(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    queries?: Record<string, unknown>,
    options?: { signal?: AbortSignal }
  ): Promise<TIssue> {
    return this.retrieveItem(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/`,
      queries,
      options
    );
  }

  async retrieveIssues(
    workspaceSlug: string,
    projectId: string,
    issueIds: string[],
    options?: { includeProperties?: boolean; signal?: AbortSignal }
  ): Promise<TIssue[]> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/list/`, {
        params: { issues: issueIds.join(","), ...(options?.includeProperties ? { include_properties: true } : {}) },
        signal: options?.signal,
      })
    );
  }

  async getIssueActivities(workspaceSlug: string, projectId: string, issueId: string): Promise<TIssueActivity[]> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/history/`)
    );
  }

  async addIssueToCycle(
    workspaceSlug: string,
    projectId: string,
    cycleId: string,
    data: {
      issues: string[];
    }
  ) {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/cycle-issues/`, data)
    );
  }

  async removeIssueFromCycle(workspaceSlug: string, projectId: string, cycleId: string, bridgeId: string) {
    return this.responseData(
      this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/cycle-issues/${bridgeId}/`)
    );
  }

  async createIssueRelation(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    data: {
      related_list: Array<{
        relation_type: "duplicate" | "relates_to" | "blocked_by";
        related_issue: string;
      }>;
      relation?: "blocking" | null;
    }
  ) {
    return this.responseData(
      this.post(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/issue-relation/`,
        data
      ),
      "response"
    );
  }

  async deleteIssueRelation(workspaceSlug: string, projectId: string, issueId: string, relationId: string) {
    return this.responseData(
      this.delete(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/issue-relation/${relationId}/`
      ),
      "response"
    );
  }

  async getIssueDisplayProperties(workspaceSlug: string, projectId: string): Promise<unknown> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issue-display-properties/`)
    );
  }

  async updateIssueDisplayProperties(
    workspaceSlug: string,
    projectId: string,
    data: IIssueDisplayProperties
  ): Promise<unknown> {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issue-display-properties/`, {
        properties: data,
      })
    );
  }

  async patchIssue(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    data: Partial<TIssue>
  ): Promise<TIssue | undefined> {
    const response = await this.patch(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/`,
      data
    ).catch((error) => {
      throw error?.response?.data ?? error;
    });
    if (response.status === 204) return undefined;
    if (!response.data) throw new Error("The work item response was empty.");
    return response.data;
  }

  async deleteIssue(workspaceSlug: string, projectId: string, issuesId: string): Promise<unknown> {
    return this.responseData(
      this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issuesId}/`)
    );
  }

  async updateIssueDates(
    workspaceSlug: string,
    projectId: string,
    updates: { id: string; start_date?: string; target_date?: string }[]
  ): Promise<void> {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issue-dates/`, { updates })
    );
  }

  async subIssues(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    queries?: Partial<Record<TIssueParams, string | boolean>>,
    options?: { signal?: AbortSignal }
  ): Promise<TIssueSubIssues> {
    return this.responseData(
      this.get(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/${this.serviceType === EIssueServiceType.EPICS ? "issues" : "sub-issues"}/`,
        { params: queries, signal: options?.signal }
      )
    );
  }

  async addSubIssues(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    data: { sub_issue_ids: string[] }
  ): Promise<TIssueSubIssues> {
    return this.responseData(
      this.post(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/${this.serviceType === EIssueServiceType.EPICS ? "issues" : "sub-issues"}/`,
        data
      )
    );
  }

  async fetchIssueLinks(workspaceSlug: string, projectId: string, issueId: string): Promise<TIssueLink[]> {
    return this.responseData(
      this.get(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/${this.serviceType === EIssueServiceType.EPICS ? "links" : "issue-links"}/`
      ),
      "response"
    );
  }

  async createIssueLink(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    data: Partial<TIssueLink>
  ): Promise<TIssueLink> {
    return this.responseData(
      this.post(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/${this.serviceType === EIssueServiceType.EPICS ? "links" : "issue-links"}/`,
        data
      ),
      "response"
    );
  }

  async updateIssueLink(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    linkId: string,
    data: Partial<TIssueLink>
  ): Promise<TIssueLink> {
    return this.responseData(
      this.patch(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/${this.serviceType === EIssueServiceType.EPICS ? "links" : "issue-links"}/${linkId}/`,
        data
      ),
      "response"
    );
  }

  async deleteIssueLink(workspaceSlug: string, projectId: string, issueId: string, linkId: string): Promise<unknown> {
    return this.responseData(
      this.delete(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/${this.serviceType === EIssueServiceType.EPICS ? "links" : "issue-links"}/${linkId}/`
      )
    );
  }

  async bulkOperations(workspaceSlug: string, projectId: string, data: TBulkOperationsPayload): Promise<unknown> {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/bulk-operation-issues/`, data)
    );
  }

  async bulkDeleteIssues(
    workspaceSlug: string,
    projectId: string,
    data: {
      issue_ids: string[];
    }
  ): Promise<unknown> {
    return this.responseData(
      this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/bulk-delete-issues/`, data)
    );
  }

  async bulkArchiveIssues(
    workspaceSlug: string,
    projectId: string,
    data: {
      issue_ids: string[];
    }
  ): Promise<{
    archived_at: string;
  }> {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/bulk-archive-issues/`, data)
    );
  }

  // issue subscriptions
  async getIssueNotificationSubscriptionStatus(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    signal?: AbortSignal
  ): Promise<{
    subscribed: boolean;
  }> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/subscribe/`, {
        signal,
      })
    );
  }

  async unsubscribeFromIssueNotifications(workspaceSlug: string, projectId: string, issueId: string): Promise<unknown> {
    return this.responseData(
      this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/subscribe/`)
    );
  }

  async subscribeToIssueNotifications(workspaceSlug: string, projectId: string, issueId: string): Promise<unknown> {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/subscribe/`)
    );
  }

  async bulkSubscribeIssues(
    workspaceSlug: string,
    projectId: string,
    data: {
      issue_ids: string[];
    }
  ): Promise<unknown> {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/bulk-subscribe-issues/`, data)
    );
  }

  async getIssueMetaFromURL(
    workspaceSlug: string,
    projectId: string,
    issueId: string
  ): Promise<{
    project_identifier: string;
    sequence_id: string;
  }> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/issues/${issueId}/meta/`)
    );
  }

  async retrieveWithIdentifier(
    workspaceSlug: string,
    project_identifier: string,
    issue_sequence: string,
    queries?: Record<string, unknown>,
    options?: { signal?: AbortSignal }
  ): Promise<TIssue> {
    return this.retrieveItem(
      `/api/workspaces/${workspaceSlug}/work-items/${project_identifier}-${issue_sequence}/`,
      queries,
      options
    );
  }

  private async retrieveItem(
    url: string,
    queries?: Record<string, unknown>,
    options?: { signal?: AbortSignal }
  ): Promise<TIssue> {
    return this.get(url, {
      params: queries,
      signal: options?.signal,
    })
      .then((response) => {
        // add is_epic flag when the service type is epic
        if (response.data && this.serviceType === EIssueServiceType.EPICS) {
          response.data.is_epic = true;
        }
        return response?.data;
      })
      .catch((error) => {
        throw error?.response?.data;
      });
  }
}
