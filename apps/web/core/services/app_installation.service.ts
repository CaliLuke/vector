/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// services
import type { ISlackIntegration } from "@plane/types";
import { API_BASE_URL } from "@plane/constants";
import { APIService } from "@/services/api.service";
// helpers

export class AppInstallationService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async addInstallationApp(workspaceSlug: string, provider: string, data: Record<string, unknown>): Promise<unknown> {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/workspace-integrations/${provider}/`, data),
      "response"
    );
  }

  async addSlackChannel(
    workspaceSlug: string,
    projectId: string,
    integrationId: string | null | undefined,
    data: Record<string, unknown>
  ): Promise<unknown> {
    return this.responseData(
      this.post(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/workspace-integrations/${integrationId}/project-slack-sync/`,
        data
      ),
      "response"
    );
  }

  async getSlackChannelDetail(
    workspaceSlug: string,
    projectId: string,
    integrationId: string | null | undefined
  ): Promise<ISlackIntegration[]> {
    return this.responseData(
      this.get(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/workspace-integrations/${integrationId}/project-slack-sync/`
      ),
      "response"
    );
  }

  async removeSlackChannel(
    workspaceSlug: string,
    projectId: string,
    integrationId: string | null | undefined,
    slackSyncId: string | undefined
  ): Promise<unknown> {
    return this.responseData(
      this.delete(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/workspace-integrations/${integrationId}/project-slack-sync/${slackSyncId}`
      ),
      "response"
    );
  }
}
