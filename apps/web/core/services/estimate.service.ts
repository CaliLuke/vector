/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { API_BASE_URL } from "@plane/constants";
import type { IEstimate, IEstimateFormData } from "@plane/types";
import { APIService } from "@/services/api.service";

export class EstimateService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }
  async fetchWorkspaceEstimates(workspaceSlug: string, signal?: AbortSignal): Promise<IEstimate[]> {
    const { data }: { data: IEstimate[] } = await this.get(`/api/workspaces/${workspaceSlug}/estimates/`, { signal });
    return data;
  }
  async fetchProjectEstimates(workspaceSlug: string, projectId: string, signal?: AbortSignal): Promise<IEstimate[]> {
    const { data }: { data: IEstimate[] } = await this.get(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/estimates/`,
      { signal }
    );
    return data;
  }
  async createEstimate(workspaceSlug: string, projectId: string, payload: IEstimateFormData): Promise<IEstimate> {
    const { data }: { data: IEstimate } = await this.post(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/estimates/`,
      payload
    );
    return data;
  }
  async deleteEstimate(workspaceSlug: string, projectId: string, estimateId: string): Promise<void> {
    await this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/estimates/${estimateId}/`);
  }
}
export default new EstimateService();
