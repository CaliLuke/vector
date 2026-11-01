/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// types
import { API_BASE_URL } from "@plane/constants";
import type { TProjectPublishSettings } from "@plane/types";
// helpers
// services
import { APIService } from "@/services/api.service";

export class ProjectPublishService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async fetchPublishSettings(
    workspaceSlug: string,
    projectID: string,
    signal?: AbortSignal
  ): Promise<TProjectPublishSettings> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/projects/${projectID}/project-deploy-boards/`, { signal }),
      "response"
    );
  }

  async publishProject(
    workspaceSlug: string,
    projectID: string,
    data: Partial<TProjectPublishSettings>
  ): Promise<TProjectPublishSettings> {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/projects/${projectID}/project-deploy-boards/`, data),
      "response"
    );
  }

  async updatePublishSettings(
    workspaceSlug: string,
    projectID: string,
    project_publish_id: string,
    data: Partial<TProjectPublishSettings>
  ): Promise<TProjectPublishSettings> {
    return this.responseData(
      this.patch(
        `/api/workspaces/${workspaceSlug}/projects/${projectID}/project-deploy-boards/${project_publish_id}/`,
        data
      ),
      "response"
    );
  }

  async unpublishProject(workspaceSlug: string, projectID: string, project_publish_id: string): Promise<unknown> {
    return this.responseData(
      this.delete(
        `/api/workspaces/${workspaceSlug}/projects/${projectID}/project-deploy-boards/${project_publish_id}/`
      ),
      "response"
    );
  }
}
