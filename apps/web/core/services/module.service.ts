/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type { IModule } from "@plane/types";
import { ModuleService as SharedModuleService } from "@plane/services";
import { installUnauthorizedRedirectInterceptor } from "@/services/api.service";

/**
 * Web compatibility adapter for the shared module service.
 *
 * The web application still uses its established method names and has one
 * full-update endpoint that uses PUT. The package service owns the common
 * endpoint behavior; its `update` method remains the PATCH operation.
 */
export class ModuleService extends SharedModuleService {
  constructor() {
    super(API_BASE_URL, installUnauthorizedRedirectInterceptor);
  }

  getWorkspaceModules(workspaceSlug: string): Promise<IModule[]> {
    return this.workspaceModulesList(workspaceSlug);
  }

  getModules(workspaceSlug: string, projectId: string): Promise<IModule[]> {
    return this.projectModulesList(workspaceSlug, projectId);
  }

  createModule(workspaceSlug: string, projectId: string, data: any): Promise<IModule> {
    return this.create(workspaceSlug, projectId, data);
  }

  updateModule(workspaceSlug: string, projectId: string, moduleId: string, data: any): Promise<any> {
    return this.put(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/${moduleId}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  getModuleDetails(workspaceSlug: string, projectId: string, moduleId: string): Promise<IModule> {
    return this.retrieve(workspaceSlug, projectId, moduleId);
  }

  patchModule(workspaceSlug: string, projectId: string, moduleId: string, data: Partial<IModule>): Promise<IModule> {
    return this.update(workspaceSlug, projectId, moduleId, data);
  }

  deleteModule(workspaceSlug: string, projectId: string, moduleId: string): Promise<any> {
    return this.destroy(workspaceSlug, projectId, moduleId);
  }
}
