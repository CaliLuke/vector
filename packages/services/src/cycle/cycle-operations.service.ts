/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { AxiosInstance } from "axios";
import { API_BASE_URL } from "@plane/constants";
import { CycleAnalyticsService } from "./cycle-analytics.service";

export class CycleOperationsService extends CycleAnalyticsService {
  constructor(BASE_URL?: string, configureAxiosInstance?: (axiosInstance: AxiosInstance) => void) {
    super(BASE_URL || API_BASE_URL, configureAxiosInstance);
  }

  /**
   * Adds a cycle to user favorites.
   * @param {string} workspaceSlug - The workspace identifier
   * @param {string} projectId - The project identifier
   * @param {{cycle: string}} data - The favorite cycle data
   * @returns {Promise<unknown>} The response data
   * @throws {Error} If the request fails
   */
  async addToFavorites(
    workspaceSlug: string,
    projectId: string,
    data: {
      cycle: string;
    }
  ): Promise<unknown> {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/user-favorite-cycles/`, data)
    );
  }

  /**
   * Removes a cycle from user favorites.
   * @param {string} workspaceSlug - The workspace identifier
   * @param {string} projectId - The project identifier
   * @param {string} cycleId - The cycle identifier
   * @returns {Promise<unknown>} The removal response
   * @throws {Error} If the request fails
   */
  async removeFromFavorites(workspaceSlug: string, projectId: string, cycleId: string): Promise<unknown> {
    return this.responseData(
      this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/user-favorite-cycles/${cycleId}/`)
    );
  }

  /**
   * Transfers issues between cycles.
   * @param {string} workspaceSlug - The workspace identifier
   * @param {string} projectId - The project identifier
   * @param {string} cycleId - The source cycle identifier
   * @param {{new_cycle_id: string}} data - The target cycle data
   * @returns {Promise<unknown>} The transfer response
   * @throws {Error} If the request fails
   */
  async transferIssues(
    workspaceSlug: string,
    projectId: string,
    cycleId: string,
    data: {
      new_cycle_id: string;
    }
  ): Promise<unknown> {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}/transfer-issues/`, data)
    );
  }
}
