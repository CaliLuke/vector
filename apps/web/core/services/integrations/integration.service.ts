/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type { IAppIntegration, IImporterService, IWorkspaceIntegration, IExportServiceResponse } from "@plane/types";
import { APIService } from "@/services/api.service";
// types
// helper

export class IntegrationService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async getAppIntegrationsList(): Promise<IAppIntegration[]> {
    return this.responseData(this.get(`/api/integrations/`));
  }

  async getWorkspaceIntegrationsList(workspaceSlug: string): Promise<IWorkspaceIntegration[]> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/workspace-integrations/`));
  }

  async deleteWorkspaceIntegration(workspaceSlug: string, integrationId: string): Promise<unknown> {
    return this.responseData(
      this.delete(`/api/workspaces/${workspaceSlug}/workspace-integrations/${integrationId}/provider/`)
    );
  }

  async getImporterServicesList(workspaceSlug: string): Promise<IImporterService[]> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/importers/`));
  }
  async getExportsServicesList(
    workspaceSlug: string,
    cursor: string,
    per_page: number
  ): Promise<IExportServiceResponse> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/export-issues`, {
        params: {
          per_page,
          cursor,
        },
      })
    );
  }

  async deleteImporterService(workspaceSlug: string, service: string, importerId: string): Promise<unknown> {
    return this.responseData(this.delete(`/api/workspaces/${workspaceSlug}/importers/${service}/${importerId}/`));
  }
}
