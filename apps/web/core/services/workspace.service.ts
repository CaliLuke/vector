/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type {
  TWorkspaceInvitationDecision,
  TWorkspaceInvitationsAccept,
  TIssueParams,
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
    return this.responseData(this.get("/api/users/me/workspaces/", { signal }), "data-or-error");
  }

  async createWorkspace(data: Partial<IWorkspace>): Promise<IWorkspace> {
    return this.responseData(this.post("/api/workspaces/", data), "data-or-error");
  }

  async updateWorkspace(workspaceSlug: string, data: Partial<IWorkspace>): Promise<IWorkspace> {
    return this.responseData(this.patch(`/api/workspaces/${workspaceSlug}/`, data), "data-or-error");
  }

  async deleteWorkspace(workspaceSlug: string): Promise<void> {
    return this.responseData(this.delete(`/api/workspaces/${workspaceSlug}/`), "data-or-error");
  }

  async inviteWorkspace(workspaceSlug: string, data: IWorkspaceBulkInviteFormData): Promise<{ message: string }> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/invitations/`, data));
  }

  async joinWorkspace(
    workspaceSlug: string,
    invitationId: string,
    data: TWorkspaceInvitationDecision
  ): Promise<unknown> {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/join/`, data, {
        headers: {},
      })
    );
  }

  async joinWorkspaces(data: TWorkspaceInvitationsAccept): Promise<unknown> {
    return this.responseData(this.post("/api/users/me/workspaces/invitations/", data));
  }

  async getLastActiveWorkspaceAndProjects(): Promise<ILastActiveWorkspaceDetails> {
    return this.responseData(this.get("/api/users/last-visited-workspace/"));
  }

  async userWorkspaceInvitations(): Promise<IWorkspaceMemberInvitation[]> {
    return this.responseData(this.get("/api/users/me/workspaces/invitations/"));
  }

  async workspaceMemberMe(workspaceSlug: string, signal?: AbortSignal): Promise<IWorkspaceMemberMe> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/workspace-members/me/`, {}, { signal }),
      "response-or-error"
    );
  }

  async updateWorkspaceView(workspaceSlug: string, data: { view_props: IWorkspaceViewProps }): Promise<unknown> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/workspace-views/`, data));
  }

  async fetchWorkspaceMembers(workspaceSlug: string, signal?: AbortSignal): Promise<IWorkspaceMember[]> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/members/`, {}, { signal }));
  }

  async updateWorkspaceMember(
    workspaceSlug: string,
    memberId: string,
    data: Pick<IWorkspaceMember, "role">
  ): Promise<IWorkspaceMember> {
    return this.responseData(this.patch(`/api/workspaces/${workspaceSlug}/members/${memberId}/`, data));
  }

  async deleteWorkspaceMember(workspaceSlug: string, memberId: string): Promise<unknown> {
    return this.responseData(this.delete(`/api/workspaces/${workspaceSlug}/members/${memberId}/`));
  }

  async workspaceInvitations(workspaceSlug: string, signal?: AbortSignal): Promise<IWorkspaceMemberInvitation[]> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/invitations/`, { signal }));
  }

  async getWorkspaceInvitation(workspaceSlug: string, invitationId: string): Promise<IWorkspaceMemberInvitation> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/join/`, { headers: {} })
    );
  }

  async updateWorkspaceInvitation(
    workspaceSlug: string,
    invitationId: string,
    data: Pick<IWorkspaceMemberInvitation, "role">
  ): Promise<IWorkspaceMemberInvitation> {
    return this.responseData(this.patch(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/`, data));
  }

  async deleteWorkspaceInvitations(workspaceSlug: string, invitationId: string): Promise<unknown> {
    return this.responseData(this.delete(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/`));
  }

  async workspaceSlugCheck(slug: string): Promise<{ status: boolean }> {
    return this.responseData(this.get(`/api/workspace-slug-check/?slug=${slug}`));
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
    return this.responseData(this.get("/api/release-notes/"));
  }

  async createView(workspaceSlug: string, data: Partial<IWorkspaceView>): Promise<IWorkspaceView> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/views/`, data));
  }

  async updateView(workspaceSlug: string, viewId: string, data: Partial<IWorkspaceView>): Promise<IWorkspaceView> {
    return this.responseData(this.patch(`/api/workspaces/${workspaceSlug}/views/${viewId}/`, data));
  }

  async deleteView(workspaceSlug: string, viewId: string): Promise<unknown> {
    return this.responseData(this.delete(`/api/workspaces/${workspaceSlug}/views/${viewId}/`));
  }

  async getAllViews(workspaceSlug: string, signal?: AbortSignal): Promise<IWorkspaceView[]> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/views/`, { signal }));
  }

  async getViewIssues(
    workspaceSlug: string,
    params: Partial<Record<TIssueParams, string | boolean>>,
    config = {}
  ): Promise<TIssuesResponse> {
    const path =
      typeof params.expand === "string" && params.expand.includes("issue_relation")
        ? `/api/workspaces/${workspaceSlug}/issues-detail/`
        : `/api/workspaces/${workspaceSlug}/issues/`;
    return this.responseData(
      this.get(
        path,
        {
          params,
        },
        config
      )
    );
  }

  async getWorkspaceUserProjectsRole(workspaceSlug: string, signal?: AbortSignal): Promise<IUserProjectsRole> {
    return this.responseData(
      this.get(`/api/users/me/workspaces/${workspaceSlug}/project-roles/`, {}, { signal }),
      "response-or-error"
    );
  }

  // quicklinks
  async fetchWorkspaceLinks(workspaceSlug: string, signal?: AbortSignal): Promise<TLink[]> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/quick-links/`, { signal }),
      "response-or-error"
    );
  }

  async createWorkspaceLink(workspaceSlug: string, data: Partial<TLink>): Promise<TLink> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/quick-links/`, data), "response-or-error");
  }

  async updateWorkspaceLink(workspaceSlug: string, linkId: string, data: Partial<TLink>): Promise<TLink> {
    return this.responseData(
      this.patch(`/api/workspaces/${workspaceSlug}/quick-links/${linkId}/`, data),
      "response-or-error"
    );
  }

  async deleteWorkspaceLink(workspaceSlug: string, linkId: string): Promise<void> {
    return this.responseData(this.delete(`/api/workspaces/${workspaceSlug}/quick-links/${linkId}/`));
  }

  async searchEntity(workspaceSlug: string, params: TSearchEntityRequestPayload): Promise<TSearchResponse> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/entity-search/`, {
        params: {
          ...params,
          query_type: params.query_type.join(","),
        },
      })
    );
  }

  // recents
  async fetchWorkspaceRecents(workspaceSlug: string, entity_name?: string): Promise<TActivityEntityData[]> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/recent-visits/`, {
        params: {
          entity_name,
        },
      }),
      "response-or-error"
    );
  }

  // widgets
  async fetchWorkspaceWidgets(workspaceSlug: string, signal?: AbortSignal): Promise<TWidgetEntityData[]> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/home-preferences/`, { signal }),
      "response-or-error"
    );
  }

  async updateWorkspaceWidget(
    workspaceSlug: string,
    widgetKey: string,
    data: Partial<TWidgetEntityData>
  ): Promise<TWidgetEntityData> {
    return this.responseData(
      this.patch(`/api/workspaces/${workspaceSlug}/home-preferences/${widgetKey}/`, data),
      "response-or-error"
    );
  }

  async fetchSidebarNavigationPreferences(
    workspaceSlug: string,
    signal?: AbortSignal
  ): Promise<IWorkspaceSidebarNavigation> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/sidebar-preferences/`, { signal }),
      "response-or-error"
    );
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
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/user-properties/`, { signal }));
  }

  async patchWorkspaceFilters(
    workspaceSlug: string,
    data: Partial<IWorkspaceUserPropertiesResponse>
  ): Promise<IWorkspaceUserPropertiesResponse> {
    return this.responseData(this.patch(`/api/workspaces/${workspaceSlug}/user-properties/`, data));
  }
}
