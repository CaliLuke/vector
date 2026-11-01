/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// types
import type { IModule, TIssueParams, TIssuesResponse } from "@plane/types";
// services
import { ModuleOperationService } from "./operations.service";

export class ModuleService extends ModuleOperationService {
  async workspaceModulesList(workspaceSlug: string): Promise<IModule[]> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/modules/`));
  }

  async projectModulesList(workspaceSlug: string, projectId: string): Promise<IModule[]> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/`));
  }

  async create(workspaceSlug: string, projectId: string, data: Partial<IModule>): Promise<IModule> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/`, data));
  }

  async retrieve(workspaceSlug: string, projectId: string, moduleId: string): Promise<IModule> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/${moduleId}/`));
  }

  async update(workspaceSlug: string, projectId: string, moduleId: string, data: Partial<IModule>): Promise<IModule> {
    return this.responseData(
      this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/${moduleId}/`, data)
    );
  }

  async destroy(workspaceSlug: string, projectId: string, moduleId: string): Promise<unknown> {
    return this.responseData(
      this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/${moduleId}/`)
    );
  }

  async getModuleIssues(
    workspaceSlug: string,
    projectId: string,
    moduleId: string,
    queries?: Partial<Record<TIssueParams, string | number | boolean>>,
    config = {}
  ): Promise<TIssuesResponse> {
    return this.responseData(
      this.get(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/${moduleId}/issues/`,
        {
          params: queries,
        },
        config
      )
    );
  }
}
