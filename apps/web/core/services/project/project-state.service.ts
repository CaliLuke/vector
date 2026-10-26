/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// services
import { API_BASE_URL } from "@plane/constants";
import type { IIntakeState, IState } from "@plane/types";
import { APIService } from "@/services/api.service";
// helpers
// types

export class ProjectStateService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async createState(workspaceSlug: string, projectId: string, data: Partial<IState>): Promise<IState> {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/states/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
      });
  }

  async markDefault(workspaceSlug: string, projectId: string, stateId: string): Promise<void> {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/states/${stateId}/mark-default/`, {})
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response;
      });
  }

  async getStates(workspaceSlug: string, projectId: string, signal?: AbortSignal): Promise<IState[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/states/`, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getIntakeState(workspaceSlug: string, projectId: string, signal?: AbortSignal): Promise<IIntakeState> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/intake-state/`, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async patchState(workspaceSlug: string, projectId: string, stateId: string, data: Partial<IState>): Promise<IState> {
    return this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/states/${stateId}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async deleteState(workspaceSlug: string, projectId: string, stateId: string): Promise<void> {
    return this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/states/${stateId}/`)
      .then(() => undefined)
      .catch((error) => {
        throw error?.response;
      });
  }

  async getWorkspaceStates(workspaceSlug: string, signal?: AbortSignal): Promise<IState[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/states/`, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
}
