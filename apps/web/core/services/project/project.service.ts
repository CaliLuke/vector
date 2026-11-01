/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type {
  GithubRepositoriesResponse,
  TSyncedGithubRepository,
  IProjectUserPropertiesResponse,
  ISearchIssueResponse,
  TProjectAnalyticsCount,
  TProjectAnalyticsCountParams,
  TProjectIssuesSearchParams,
} from "@plane/types";
// helpers
// plane web types
import type { TProject, TPartialProject } from "@plane/types";
// services
import { APIService } from "@/services/api.service";

export class ProjectService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async createProject(workspaceSlug: string, data: Partial<TProject>): Promise<TProject> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/projects/`, data), "response");
  }

  async checkProjectIdentifierAvailability(workspaceSlug: string, data: string): Promise<{ exists: boolean }> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/project-identifiers`, {
        params: {
          name: data,
        },
      })
    );
  }

  async getProjectsLite(workspaceSlug: string, signal?: AbortSignal): Promise<TPartialProject[]> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/projects/`, { signal }));
  }

  async getProjects(workspaceSlug: string, signal?: AbortSignal): Promise<TProject[]> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/projects/details/`, { signal }));
  }

  async getProject(workspaceSlug: string, projectId: string, signal?: AbortSignal): Promise<TProject> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/`, { signal }),
      "response"
    );
  }

  async getProjectAnalyticsCount(
    workspaceSlug: string,
    params?: TProjectAnalyticsCountParams,
    signal?: AbortSignal
  ): Promise<TProjectAnalyticsCount[]> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/project-stats/`, {
        params,
        signal,
      })
    );
  }

  async updateProject(workspaceSlug: string, projectId: string, data: Partial<TProject>): Promise<TProject> {
    return this.responseData(this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/`, data));
  }

  async deleteProject(workspaceSlug: string, projectId: string): Promise<unknown> {
    return this.responseData(this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/`));
  }

  // User Properties
  async getProjectUserProperties(
    workspaceSlug: string,
    projectId: string,
    signal?: AbortSignal
  ): Promise<IProjectUserPropertiesResponse> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/user-properties/`, { signal }),
      "data-or-error"
    );
  }

  async updateProjectUserProperties(
    workspaceSlug: string,
    projectId: string,
    data: Partial<IProjectUserPropertiesResponse>
  ): Promise<IProjectUserPropertiesResponse> {
    return this.responseData(
      this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/user-properties/`, data)
    );
  }

  async getGithubRepositories(url: string): Promise<GithubRepositoriesResponse> {
    return this.responseData(
      this.request({
        method: "get",
        url,
      })
    );
  }

  async syncGithubRepository(
    workspaceSlug: string,
    projectId: string,
    workspaceIntegrationId: string,
    data: {
      name: string;
      owner: string;
      repository_id: string;
      url: string;
    }
  ): Promise<unknown> {
    return this.responseData(
      this.post(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/workspace-integrations/${workspaceIntegrationId}/github-repository-sync/`,
        data
      )
    );
  }

  async getProjectGithubRepository(
    workspaceSlug: string,
    projectId: string,
    integrationId: string
  ): Promise<TSyncedGithubRepository[]> {
    return this.responseData(
      this.get(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/workspace-integrations/${integrationId}/github-repository-sync/`
      )
    );
  }

  async getUserProjectFavorites(workspaceSlug: string): Promise<unknown[]> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/user-favorite-projects/`));
  }

  async addProjectToFavorites(workspaceSlug: string, project: string): Promise<unknown> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/user-favorite-projects/`, { project }));
  }

  async removeProjectFromFavorites(workspaceSlug: string, projectId: string): Promise<unknown> {
    return this.responseData(this.delete(`/api/workspaces/${workspaceSlug}/user-favorite-projects/${projectId}/`));
  }

  async projectIssuesSearch(
    workspaceSlug: string,
    projectId: string,
    params: TProjectIssuesSearchParams,
    signal?: AbortSignal
  ): Promise<ISearchIssueResponse[]> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/search-issues/`, {
        params,
        signal,
      }),
      "data-or-error"
    );
  }
}
