/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type {
  IWorkspace,
  IWorkspaceMemberMe,
  IWorkspaceMember,
  IWorkspaceMemberInvitation,
  ILastActiveWorkspaceDetails,
  IWorkspaceSearchResults,
  IProductUpdateResponse,
  IWorkspaceBulkInviteFormData,
  IWorkspaceViewProps,
  IUserProjectsRole,
  IWorkspaceView,
  TIssuesResponse,
  TLink,
  TSearchResponse,
  TSearchEntityRequestPayload,
  TWidgetEntityData,
  TActivityEntityData,
  IWorkspaceSidebarNavigation,
  IWorkspaceUserPropertiesResponse,
} from "@plane/types";
// services
import { APIService } from "@/services/api.service";

export class WorkspaceService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async userWorkspaces(signal?: AbortSignal): Promise<IWorkspace[]> {
    return this.get("/api/users/me/workspaces/", { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async createWorkspace(data: Partial<IWorkspace>): Promise<IWorkspace> {
    return this.post("/api/workspaces/", data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async updateWorkspace(workspaceSlug: string, data: Partial<IWorkspace>): Promise<IWorkspace> {
    return this.patch(`/api/workspaces/${workspaceSlug}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async deleteWorkspace(workspaceSlug: string): Promise<void> {
    return this.delete(`/api/workspaces/${workspaceSlug}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async inviteWorkspace(workspaceSlug: string, data: IWorkspaceBulkInviteFormData): Promise<{ message: string }> {
    return this.post(`/api/workspaces/${workspaceSlug}/invitations/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async joinWorkspace(workspaceSlug: string, invitationId: string, data: any): Promise<any> {
    return this.post(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/join/`, data, {
      headers: {},
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async joinWorkspaces(data: any): Promise<any> {
    return this.post("/api/users/me/workspaces/invitations/", data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getLastActiveWorkspaceAndProjects(): Promise<ILastActiveWorkspaceDetails> {
    return this.get("/api/users/last-visited-workspace/")
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async userWorkspaceInvitations(): Promise<IWorkspaceMemberInvitation[]> {
    return this.get("/api/users/me/workspaces/invitations/")
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async workspaceMemberMe(workspaceSlug: string, signal?: AbortSignal): Promise<IWorkspaceMemberMe> {
    return this.get(`/api/workspaces/${workspaceSlug}/workspace-members/me/`, {}, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  async updateWorkspaceView(workspaceSlug: string, data: { view_props: IWorkspaceViewProps }): Promise<any> {
    return this.post(`/api/workspaces/${workspaceSlug}/workspace-views/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async fetchWorkspaceMembers(workspaceSlug: string, signal?: AbortSignal): Promise<IWorkspaceMember[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/members/`, {}, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async updateWorkspaceMember(
    workspaceSlug: string,
    memberId: string,
    data: Partial<IWorkspaceMember>
  ): Promise<IWorkspaceMember> {
    return this.patch(`/api/workspaces/${workspaceSlug}/members/${memberId}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async deleteWorkspaceMember(workspaceSlug: string, memberId: string): Promise<any> {
    return this.delete(`/api/workspaces/${workspaceSlug}/members/${memberId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async workspaceInvitations(workspaceSlug: string, signal?: AbortSignal): Promise<IWorkspaceMemberInvitation[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/invitations/`, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getWorkspaceInvitation(workspaceSlug: string, invitationId: string): Promise<IWorkspaceMemberInvitation> {
    return this.get(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/join/`, { headers: {} })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async updateWorkspaceInvitation(
    workspaceSlug: string,
    invitationId: string,
    data: Pick<IWorkspaceMemberInvitation, "role">
  ): Promise<IWorkspaceMemberInvitation> {
    return this.patch(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async deleteWorkspaceInvitations(workspaceSlug: string, invitationId: string): Promise<any> {
    return this.delete(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async workspaceSlugCheck(slug: string): Promise<any> {
    return this.get(`/api/workspace-slug-check/?slug=${slug}`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async searchWorkspace(
    workspaceSlug: string,
    params: { project_id?: string; search: string; workspace_search: boolean },
    signal?: AbortSignal
  ): Promise<IWorkspaceSearchResults> {
    const response = await this.get(`/api/workspaces/${workspaceSlug}/search/`, { params, signal });
    return response.data;
  }
  async getProductUpdates(): Promise<IProductUpdateResponse[]> {
    return this.get("/api/release-notes/")
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async createView(workspaceSlug: string, data: Partial<IWorkspaceView>): Promise<IWorkspaceView> {
    return this.post(`/api/workspaces/${workspaceSlug}/views/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async updateView(workspaceSlug: string, viewId: string, data: Partial<IWorkspaceView>): Promise<IWorkspaceView> {
    return this.patch(`/api/workspaces/${workspaceSlug}/views/${viewId}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async deleteView(workspaceSlug: string, viewId: string): Promise<any> {
    return this.delete(`/api/workspaces/${workspaceSlug}/views/${viewId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getAllViews(workspaceSlug: string, signal?: AbortSignal): Promise<IWorkspaceView[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/views/`, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getViewIssues(workspaceSlug: string, params: any, config = {}): Promise<TIssuesResponse> {
    const path = params.expand?.includes("issue_relation")
      ? `/api/workspaces/${workspaceSlug}/issues-detail/`
      : `/api/workspaces/${workspaceSlug}/issues/`;
    return this.get(
      path,
      {
        params,
      },
      config
    )
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getWorkspaceUserProjectsRole(workspaceSlug: string, signal?: AbortSignal): Promise<IUserProjectsRole> {
    return this.get(`/api/users/me/workspaces/${workspaceSlug}/project-roles/`, {}, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  // quicklinks
  async fetchWorkspaceLinks(workspaceSlug: string, signal?: AbortSignal): Promise<TLink[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/quick-links/`, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  async createWorkspaceLink(workspaceSlug: string, data: Partial<TLink>): Promise<TLink> {
    return this.post(`/api/workspaces/${workspaceSlug}/quick-links/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  async updateWorkspaceLink(workspaceSlug: string, linkId: string, data: Partial<TLink>): Promise<TLink> {
    return this.patch(`/api/workspaces/${workspaceSlug}/quick-links/${linkId}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  async deleteWorkspaceLink(workspaceSlug: string, linkId: string): Promise<void> {
    return this.delete(`/api/workspaces/${workspaceSlug}/quick-links/${linkId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async searchEntity(workspaceSlug: string, params: TSearchEntityRequestPayload): Promise<TSearchResponse> {
    return this.get(`/api/workspaces/${workspaceSlug}/entity-search/`, {
      params: {
        ...params,
        query_type: params.query_type.join(","),
      },
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  // recents
  async fetchWorkspaceRecents(workspaceSlug: string, entity_name?: string): Promise<TActivityEntityData[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/recent-visits/`, {
      params: {
        entity_name,
      },
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  // widgets
  async fetchWorkspaceWidgets(workspaceSlug: string, signal?: AbortSignal): Promise<TWidgetEntityData[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/home-preferences/`, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  async updateWorkspaceWidget(
    workspaceSlug: string,
    widgetKey: string,
    data: Partial<TWidgetEntityData>
  ): Promise<TWidgetEntityData> {
    return this.patch(`/api/workspaces/${workspaceSlug}/home-preferences/${widgetKey}/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  async fetchSidebarNavigationPreferences(
    workspaceSlug: string,
    signal?: AbortSignal
  ): Promise<IWorkspaceSidebarNavigation> {
    return this.get(`/api/workspaces/${workspaceSlug}/sidebar-preferences/`, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  async updateBulkSidebarPreferences(
    workspaceSlug: string,
    data: Array<{ key: string; is_pinned?: boolean; sort_order?: number }>
  ): Promise<void> {
    return this.patch(`/api/workspaces/${workspaceSlug}/sidebar-preferences/`, data)
      .then(() => undefined)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  async fetchWorkspaceFilters(workspaceSlug: string, signal?: AbortSignal): Promise<IWorkspaceUserPropertiesResponse> {
    return this.get(`/api/workspaces/${workspaceSlug}/user-properties/`, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async patchWorkspaceFilters(
    workspaceSlug: string,
    data: Partial<IWorkspaceUserPropertiesResponse>
  ): Promise<IWorkspaceUserPropertiesResponse> {
    return this.patch(`/api/workspaces/${workspaceSlug}/user-properties/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
}
