/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// services
import { API_BASE_URL } from "@plane/constants";
import type {
  CycleDateCheckData,
  ICycle,
  TCycleDistribution,
  TProgressSnapshot,
  TCycleEstimateDistribution,
} from "@plane/types";
import { CycleService as SharedCycleService } from "@plane/services";
import { installUnauthorizedRedirectInterceptor } from "@/services/api.service";

export class CycleService extends SharedCycleService {
  constructor() {
    super(API_BASE_URL, installUnauthorizedRedirectInterceptor);
  }

  async workspaceActiveCyclesAnalytics(
    workspaceSlug: string,
    projectId: string,
    cycleId: string,
    analytic_type: string = "points"
  ): Promise<TCycleDistribution | TCycleEstimateDistribution> {
    return this.get(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/analytics?type=${analytic_type}`
    )
      .then((res) => res?.data)
      .catch((err) => {
        throw err?.response?.data;
      });
  }

  async workspaceActiveCyclesProgress(
    workspaceSlug: string,
    projectId: string,
    cycleId: string
  ): Promise<TProgressSnapshot> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/progress/`)
      .then((res) => res?.data)
      .catch((err) => {
        throw err?.response?.data;
      });
  }

  async workspaceActiveCyclesProgressPro(
    workspaceSlug: string,
    projectId: string,
    cycleId: string
  ): Promise<TProgressSnapshot> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/cycle-progress/`)
      .then((res) => res?.data)
      .catch((err) => {
        throw err?.response?.data;
      });
  }

  async createCycle(workspaceSlug: string, projectId: string, data: any): Promise<ICycle> {
    return this.create(workspaceSlug, projectId, data);
  }

  async getCyclesWithParams(workspaceSlug: string, projectId: string, cycleType?: "current"): Promise<ICycle[]> {
    return this.getWithParams(workspaceSlug, projectId, cycleType);
  }

  async getCycleDetails(workspaceSlug: string, projectId: string, cycleId: string): Promise<ICycle> {
    return this.retrieve(workspaceSlug, projectId, cycleId);
  }

  async patchCycle(workspaceSlug: string, projectId: string, cycleId: string, data: Partial<ICycle>): Promise<any> {
    return this.update(workspaceSlug, projectId, cycleId, data);
  }

  async deleteCycle(workspaceSlug: string, projectId: string, cycleId: string): Promise<any> {
    return this.destroy(workspaceSlug, projectId, cycleId);
  }

  async cycleDateCheck(workspaceSlug: string, projectId: string, data: CycleDateCheckData): Promise<any> {
    return this.validateDates(workspaceSlug, projectId, data);
  }

  async addCycleToFavorites(
    workspaceSlug: string,
    projectId: string,
    data: {
      cycle: string;
    }
  ): Promise<any> {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/user-favorite-cycles/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async transferIssues(
    workspaceSlug: string,
    projectId: string,
    cycleId: string,
    data: {
      new_cycle_id: string;
    }
  ): Promise<any> {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/transfer-issues/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async removeCycleFromFavorites(workspaceSlug: string, projectId: string, cycleId: string): Promise<any> {
    return this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/user-favorite-cycles/${cycleId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
}
