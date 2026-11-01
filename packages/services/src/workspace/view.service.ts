/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type { IWorkspaceView, TIssueParams, TIssuesResponse } from "@plane/types";
import { APIService } from "../api.service";

export class WorkspaceViewService extends APIService {
  /**
   * Creates an instance of WorkspaceViewService
   * @param {string} baseUrl - The base URL for API requests
   */
  constructor(BASE_URL?: string) {
    super(BASE_URL || API_BASE_URL);
  }

  async create(workspaceSlug: string, data: Partial<IWorkspaceView>): Promise<IWorkspaceView> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/views/`, data));
  }

  async update(workspaceSlug: string, viewId: string, data: Partial<IWorkspaceView>): Promise<IWorkspaceView> {
    return this.responseData(this.patch(`/api/workspaces/${workspaceSlug}/views/${viewId}/`, data));
  }

  async destroy(workspaceSlug: string, viewId: string): Promise<unknown> {
    return this.responseData(this.delete(`/api/workspaces/${workspaceSlug}/views/${viewId}/`));
  }

  async list(workspaceSlug: string): Promise<IWorkspaceView[]> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/views/`));
  }

  async retrieve(workspaceSlug: string, viewId: string): Promise<IWorkspaceView> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/views/${viewId}/`));
  }

  async getViewIssues(
    workspaceSlug: string,
    params: Partial<Record<TIssueParams, string | boolean>>,
    config = {}
  ): Promise<TIssuesResponse> {
    return this.responseData(
      this.get(
        `/api/workspaces/${workspaceSlug}/issues/`,
        {
          params,
        },
        config
      )
    );
  }
}
