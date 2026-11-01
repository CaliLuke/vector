/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// types
import { API_BASE_URL } from "@plane/constants";
import type { IProjectBulkAddFormData, TProjectMembership } from "@plane/types";
// services
import { APIService } from "@/services/api.service";

export class ProjectMemberService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async fetchProjectMembers(
    workspaceSlug: string,
    projectId: string,
    signal?: AbortSignal
  ): Promise<TProjectMembership[]> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/`, {}, { signal })
    );
  }

  async bulkAddMembersToProject(
    workspaceSlug: string,
    projectId: string,
    data: IProjectBulkAddFormData
  ): Promise<TProjectMembership[]> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/`, data));
  }

  async getProjectMember(workspaceSlug: string, projectId: string, memberId: string): Promise<TProjectMembership> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/${memberId}/`));
  }

  async updateProjectMember(
    workspaceSlug: string,
    projectId: string,
    memberId: string,
    data: Partial<TProjectMembership>
  ): Promise<TProjectMembership> {
    return this.responseData(
      this.patch(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/${memberId}/`, data)
    );
  }

  async deleteProjectMember(workspaceSlug: string, projectId: string, memberId: string): Promise<void> {
    return this.responseData(
      this.delete(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/${memberId}/`)
    );
  }
}
