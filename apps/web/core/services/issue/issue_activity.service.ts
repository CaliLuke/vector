/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type { TIssueActivity, TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
import { APIService } from "@/services/api.service";
// types
// helper

export class IssueActivityService extends APIService {
  private serviceType: TIssueServiceType;

  constructor(serviceType: TIssueServiceType = EIssueServiceType.ISSUES) {
    super(API_BASE_URL);
    this.serviceType = serviceType;
  }

  async getIssueActivities(
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    signal?: AbortSignal
  ): Promise<TIssueActivity[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/${this.serviceType}/${issueId}/history/`, {
      signal,
      params: {
        activity_type: this.serviceType === EIssueServiceType.EPICS ? "epic-property" : "issue-property",
      },
    }).then((response) => response?.data);
  }
}
