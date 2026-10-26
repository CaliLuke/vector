/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { API_BASE_URL } from "@plane/constants";
import type {
  IAnalyticsResponse,
  TAnalyticsTabsBase,
  TAnalyticsGraphsBase,
  TAnalyticsFilterParams,
} from "@plane/types";
// services
import { APIService } from "./api.service";

export class AnalyticsService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  private getAnalyticsData<T>(url: string, params: object, signal?: AbortSignal): Promise<T> {
    return this.get(url, { params, signal })
      .then((res) => res?.data)
      .catch((err) => {
        throw err?.response?.data;
      });
  }

  async getAdvanceAnalytics<T extends IAnalyticsResponse>(
    workspaceSlug: string,
    tab: TAnalyticsTabsBase,
    params?: TAnalyticsFilterParams,
    isPeekView?: boolean,
    signal?: AbortSignal
  ): Promise<T> {
    return this.getAnalyticsData<T>(
      this.processUrl("advance-analytics", workspaceSlug, tab, params, isPeekView),
      {
        tab,
        ...params,
      },
      signal
    );
  }

  async getAdvanceAnalyticsStats<T>(
    workspaceSlug: string,
    tab: Exclude<TAnalyticsTabsBase, "overview">,
    params?: TAnalyticsFilterParams,
    isPeekView?: boolean,
    signal?: AbortSignal
  ): Promise<T> {
    const processedUrl = this.processUrl("advance-analytics-stats", workspaceSlug, tab, params, isPeekView);
    return this.getAnalyticsData<T>(
      processedUrl,
      {
        type: tab,
        ...params,
      },
      signal
    );
  }

  async getAdvanceAnalyticsCharts<T>(
    workspaceSlug: string,
    tab: TAnalyticsGraphsBase,
    params?: TAnalyticsFilterParams,
    isPeekView?: boolean,
    signal?: AbortSignal
  ): Promise<T> {
    const processedUrl = this.processUrl("advance-analytics-charts", workspaceSlug, tab, params, isPeekView);
    return this.getAnalyticsData<T>(
      processedUrl,
      {
        type: tab,
        ...params,
      },
      signal
    );
  }

  processUrl(
    endpoint: string,
    workspaceSlug: string,
    tab: TAnalyticsGraphsBase | TAnalyticsTabsBase,
    params?: TAnalyticsFilterParams,
    isPeekView?: boolean
  ) {
    let processedUrl = `/api/workspaces/${workspaceSlug}`;
    if (isPeekView && (tab === "work-items" || tab === "custom-work-items")) {
      const projectIds = params?.project_ids;
      if (typeof projectIds !== "string" || !projectIds.trim()) {
        throw new Error("project_ids parameter is required for peek view of work items");
      }
      const projectId = projectIds.split(",")[0];
      if (!projectId) {
        throw new Error("Invalid project_ids format - no project ID found");
      }
      processedUrl += `/projects/${projectId}`;
    }
    return `${processedUrl}/${endpoint}`;
  }
}
