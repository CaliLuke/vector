/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// type
import { API_BASE_URL } from "@plane/constants";
import type { IModule } from "@plane/types";
// helpers
// services
import { APIService } from "@/services/api.service";

export class ModuleArchiveService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async getArchivedModules(workspaceSlug: string, projectId: string, signal?: AbortSignal): Promise<IModule[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/archived-modules/`, { signal }).then(
      (response) => response?.data
    );
  }

  async getArchivedModuleDetails(
    workspaceSlug: string,
    projectId: string,
    moduleId: string,
    signal?: AbortSignal
  ): Promise<IModule> {
    return this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/archived-modules/${moduleId}/`, {
      signal,
    }).then((response) => response?.data);
  }

  async archiveModule(
    workspaceSlug: string,
    projectId: string,
    moduleId: string
  ): Promise<{
    archived_at: string;
  }> {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/${moduleId}/archive/`).then(
      (response) => response?.data
    );
  }

  async restoreModule(workspaceSlug: string, projectId: string, moduleId: string): Promise<void> {
    return this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/${moduleId}/archive/`).then(
      (response) => response?.data
    );
  }
}
