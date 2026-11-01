/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type {
  TWorkspaceInvitationDecision,
  TWorkspaceInvitationsAccept,
  IWorkspaceMemberInvitation,
  IWorkspaceBulkInviteFormData,
  IWorkspaceMember,
} from "@plane/types";
import { APIService } from "../api.service";

/**
 * Service class for managing workspace invitations
 * Handles operations related to inviting users to workspaces and managing invitations
 * @extends {APIService}
 */
export class WorkspaceInvitationService extends APIService {
  constructor(BASE_URL?: string) {
    super(BASE_URL || API_BASE_URL);
  }

  /**
   * Retrieves all workspace invitations for the current user
   * @returns {Promise<IWorkspaceMemberInvitation[]>} Promise resolving to array of workspace invitations
   * @throws {Error} If the API request fails
   */
  async userInvitations(): Promise<IWorkspaceMemberInvitation[]> {
    return this.responseData(this.get("/api/users/me/workspaces/invitations/"));
  }

  /**
   * Retrieves all invitations for a specific workspace
   * @param {string} workspaceSlug - The unique slug identifier for the workspace
   * @returns {Promise<IWorkspaceMemberInvitation[]>} Promise resolving to array of workspace invitations
   * @throws {Error} If the API request fails
   */
  async workspaceInvitations(workspaceSlug: string): Promise<IWorkspaceMemberInvitation[]> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/invitations/`));
  }

  /**
   * Sends bulk invitations to users for a workspace
   * @param {string} workspaceSlug - The unique slug identifier for the workspace
   * @param {IWorkspaceBulkInviteFormData} data - Bulk invitation data containing user information
   * @returns {Promise<unknown>} Promise resolving to the invitation response
   * @throws {Error} If the API request fails
   */
  async invite(workspaceSlug: string, data: IWorkspaceBulkInviteFormData): Promise<{ message: string }> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/invitations/`, data));
  }

  /**
   * Update Invitation
   * @param workspaceSlug
   * @param invitationId
   * @param data
   * @returns
   */
  async update(
    workspaceSlug: string,
    invitationId: string,
    data: Partial<IWorkspaceMember>
  ): Promise<IWorkspaceMemberInvitation> {
    return this.responseData(this.patch(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/`, data));
  }

  /**
   * Delete Workspace invitation
   * @param workspaceSlug
   * @param invitationId
   * @returns
   */
  async destroy(workspaceSlug: string, invitationId: string): Promise<unknown> {
    return this.responseData(this.delete(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/`));
  }

  /**
   * Accepts an invitation to join a workspace
   * @param {string} workspaceSlug - The unique slug identifier for the workspace
   * @param {string} invitationId - The unique identifier for the invitation
   * @param {object} data - Additional data required for joining the workspace
   * @returns {Promise<unknown>} Promise resolving to the join response
   * @throws {Error} If the API request fails
   */
  async join(workspaceSlug: string, invitationId: string, data: TWorkspaceInvitationDecision): Promise<unknown> {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/invitations/${invitationId}/join/`, data, {
        headers: {},
      })
    );
  }

  /**
   * Accepts multiple workspace invitations at once
   * @param {object} data - Data containing information about invitations to accept
   * @returns {Promise<unknown>} Promise resolving to the bulk join response
   * @throws {Error} If the API request fails
   */
  async joinMany(data: TWorkspaceInvitationsAccept): Promise<unknown> {
    return this.responseData(this.post("/api/users/me/workspaces/invitations/", data));
  }
}
