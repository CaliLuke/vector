/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type { IModule, ILinkDetails, ModuleLink } from "@plane/types";
import { ModuleService as SharedModuleService } from "@plane/services";
import { installUnauthorizedRedirectInterceptor } from "@/services/api.service";

export class ModuleService extends SharedModuleService {
  constructor() {
    super(API_BASE_URL, installUnauthorizedRedirectInterceptor);
  }
  async getWorkspaceModules(workspaceSlug: string, signal?: AbortSignal): Promise<IModule[]> {
    return (await this.get(`/api/workspaces/${workspaceSlug}/modules/`, { signal })).data;
  }
  async getModules(workspaceSlug: string, projectId: string, signal?: AbortSignal): Promise<IModule[]> {
    return (await this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/`, { signal })).data;
  }
  async getModuleDetails(
    workspaceSlug: string,
    projectId: string,
    moduleId: string,
    signal?: AbortSignal
  ): Promise<IModule> {
    return (await this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/${moduleId}/`, { signal }))
      .data;
  }
  async createModule(workspaceSlug: string, projectId: string, data: Partial<IModule>): Promise<IModule> {
    return (await this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/`, data)).data;
  }
  async updateModule(
    workspaceSlug: string,
    projectId: string,
    moduleId: string,
    data: Partial<IModule>
  ): Promise<IModule> {
    return (await this.put(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/${moduleId}/`, data)).data;
  }
  async patchModule(
    workspaceSlug: string,
    projectId: string,
    moduleId: string,
    data: Partial<IModule>
  ): Promise<IModule> {
    return (await this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/${moduleId}/`, data)).data;
  }
  async deleteModule(workspaceSlug: string, projectId: string, moduleId: string): Promise<void> {
    await this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/${moduleId}/`);
  }

  async createModuleLink(
    workspaceSlug: string,
    projectId: string,
    moduleId: string,
    data: Partial<ModuleLink> & { metadata?: Record<string, unknown> }
  ): Promise<ILinkDetails> {
    return (
      await this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/${moduleId}/module-links/`, data)
    ).data;
  }
  async updateModuleLink(
    workspaceSlug: string,
    projectId: string,
    moduleId: string,
    linkId: string,
    data: Partial<ModuleLink> & { metadata?: Record<string, unknown> }
  ): Promise<ILinkDetails> {
    return (
      await this.patch(
        `/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/${moduleId}/module-links/${linkId}/`,
        data
      )
    ).data;
  }
  async deleteModuleLink(workspaceSlug: string, projectId: string, moduleId: string, linkId: string): Promise<void> {
    await this.delete(
      `/api/workspaces/${workspaceSlug}/projects/${projectId}/modules/${moduleId}/module-links/${linkId}/`
    );
  }
}
