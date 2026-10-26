/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// services
import type {
  ICycle,
  CycleDateCheckData,
  TIssuesResponse,
  TProgressSnapshot,
  TCycleDistribution,
  TCycleEstimateDistribution,
} from "@plane/types";
import { API_BASE_URL } from "@plane/constants";
import type { AxiosRequestConfig } from "axios";
import { CycleService as SharedCycleService } from "@plane/services";
import { installUnauthorizedRedirectInterceptor } from "@/services/api.service";

export class CycleService extends SharedCycleService {
  constructor() {
    super(API_BASE_URL, installUnauthorizedRedirectInterceptor);
  }
  async getWorkspaceCycles(workspaceSlug: string, signal?: AbortSignal): Promise<ICycle[]> {
    return (await this.get(`/api/workspaces/${workspaceSlug}/cycles/`, { signal })).data;
  }
  async getWithParams(
    workspaceSlug: string,
    projectId: string,
    cycleType?: "current",
    signal?: AbortSignal
  ): Promise<ICycle[]> {
    return (
      await this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/`, {
        params: { cycle_view: cycleType },
        signal,
      })
    ).data;
  }
  async retrieve(workspaceSlug: string, projectId: string, cycleId: string, signal?: AbortSignal): Promise<ICycle> {
    return (await this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/`, { signal }))
      .data;
  }
  async create(workspaceSlug: string, projectId: string, data: Partial<ICycle>): Promise<ICycle> {
    return (await this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/`, data)).data;
  }
  async update(workspaceSlug: string, projectId: string, cycleId: string, data: Partial<ICycle>): Promise<ICycle> {
    return (await this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/`, data)).data;
  }
  async destroy(workspaceSlug: string, projectId: string, cycleId: string): Promise<void> {
    await this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/`);
  }
  async workspaceActiveCyclesProgress(
    workspaceSlug: string,
    projectId: string,
    cycleId: string,
    signal?: AbortSignal
  ): Promise<TProgressSnapshot> {
    return (
      await this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/progress/`, { signal })
    ).data;
  }
  async issueAnalytics(
    workspaceSlug: string,
    projectId: string,
    cycleId: string,
    signal?: AbortSignal
  ): Promise<TCycleDistribution> {
    return (
      await this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/analytics/`, {
        params: { type: "issues" },
        signal,
      })
    ).data;
  }
  async estimateAnalytics(
    workspaceSlug: string,
    projectId: string,
    cycleId: string,
    signal?: AbortSignal
  ): Promise<TCycleEstimateDistribution> {
    return (
      await this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/analytics/`, {
        params: { type: "points" },
        signal,
      })
    ).data;
  }
  async transferIssues(
    workspaceSlug: string,
    projectId: string,
    cycleId: string,
    data: { new_cycle_id: string }
  ): Promise<void> {
    await this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/transfer-issues/`, data);
  }
  async getCycleIssues(
    workspaceSlug: string,
    projectId: string,
    cycleId: string,
    params?: Record<string, string | number | boolean>,
    config: AxiosRequestConfig = {}
  ): Promise<TIssuesResponse> {
    return (
      await this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/cycle-issues/`, {
        params,
        ...config,
      })
    ).data;
  }

  async validateDates(
    workspaceSlug: string,
    projectId: string,
    data: CycleDateCheckData
  ): Promise<{ status: boolean }> {
    return (await this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/date-check/`, data)).data;
  }
}
