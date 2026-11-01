/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// services
import { API_BASE_URL } from "@plane/constants";
import type {
  TIssue,
  TIssueParams,
  IUser,
  IUserActivityResponse,
  IUserProfileData,
  IUserProfileProjectSegregation,
  IUserSettings,
  IUserEmailNotificationSettings,
  TIssuesResponse,
  TUserProfile,
  IEmailCheckResponse,
} from "@plane/types";
import { APIService } from "@/services/api.service";
// types
// helpers

export class UserService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  currentUserConfig() {
    return {
      url: `${this.baseURL}/api/users/me/`,
    };
  }

  async userIssues(
    workspaceSlug: string,
    params: Partial<Record<TIssueParams, string | boolean>>
  ): Promise<
    | {
        [key: string]: TIssue[];
      }
    | TIssue[]
  > {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/my-issues/`, {
        params,
      })
    );
  }

  async currentUser(signal?: AbortSignal): Promise<IUser | null> {
    // A signed-out response is data for the route gate, not an interceptor redirect.
    const response = await this.get("/api/users/me/", { signal, validateStatus: null });
    if (response.status === 401) return null;
    if (response.status < 200 || response.status >= 300) throw response;
    if (!response.data?.id) throw new Error("The server returned an invalid account response.");
    return response.data;
  }

  async getCurrentUserProfile(signal?: AbortSignal): Promise<TUserProfile> {
    return this.responseData(this.get("/api/users/me/profile/", { signal }), "response-or-error");
  }
  async updateCurrentUserProfile(data: Partial<TUserProfile>, signal?: AbortSignal): Promise<TUserProfile> {
    return this.responseData(this.patch("/api/users/me/profile/", data, { signal }), "response-or-error");
  }

  async currentUserSettings(bustCache: boolean = false, signal?: AbortSignal): Promise<IUserSettings> {
    const url = bustCache ? `/api/users/me/settings/?t=${Date.now()}` : "/api/users/me/settings/";
    return this.responseData(this.get(url, { signal }), "response-or-error");
  }

  async currentUserEmailNotificationSettings(signal?: AbortSignal): Promise<IUserEmailNotificationSettings> {
    return this.responseData(this.get("/api/users/me/notification-preferences/", { signal }), "response-or-error");
  }

  async updateUser(data: Partial<IUser>): Promise<IUser> {
    return this.responseData(this.patch("/api/users/me/", data), "data-or-error");
  }

  async updateUserOnBoard(signal?: AbortSignal): Promise<{ message: string }> {
    return this.responseData(
      this.patch(
        "/api/users/me/onboard/",
        {
          is_onboarded: true,
        },
        { signal }
      ),
      "data-or-error"
    );
  }

  async updateUserTourCompleted(signal?: AbortSignal): Promise<{ message: string }> {
    return this.responseData(
      this.patch(
        "/api/users/me/tour-completed/",
        {
          is_tour_completed: true,
        },
        { signal }
      ),
      "data-or-error"
    );
  }

  async updateCurrentUserEmailNotificationSettings(
    data: Partial<IUserEmailNotificationSettings>
  ): Promise<IUserEmailNotificationSettings> {
    return this.responseData(this.patch("/api/users/me/notification-preferences/", data), "data-or-error");
  }

  async changePassword(token: string, data: { old_password?: string; new_password: string }): Promise<unknown> {
    return this.responseData(
      this.post(`/auth/change-password/`, data, {
        headers: {
          "X-CSRFTOKEN": token,
        },
      }),
      "data-or-error"
    );
  }

  async getUserProfileData(workspaceSlug: string, userId: string): Promise<IUserProfileData> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/user-stats/${userId}/`));
  }

  async getUserProfileProjectsSegregation(
    workspaceSlug: string,
    userId: string
  ): Promise<IUserProfileProjectSegregation> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/user-profile/${userId}/`));
  }

  async getUserProfileActivity(
    workspaceSlug: string,
    userId: string,
    params: {
      per_page: number;
      cursor?: string;
    }
  ): Promise<IUserActivityResponse> {
    return this.responseData(
      this.get(`/api/workspaces/${workspaceSlug}/user-activity/${userId}/`, {
        params,
      })
    );
  }

  async downloadProfileActivity(
    workspaceSlug: string,
    userId: string,
    data: {
      date: string;
    }
  ): Promise<string> {
    const response = await this.post(`/api/workspaces/${workspaceSlug}/user-activity/${userId}/export/`, data, {
      responseType: "text",
    });
    if (typeof response.data !== "string") throw new Error("Invalid activity export response");
    return response.data;
  }

  async getUserProfileIssues(
    workspaceSlug: string,
    userId: string,
    params: Partial<Record<TIssueParams, string | boolean>>,
    config = {}
  ): Promise<TIssuesResponse> {
    return this.responseData(
      this.get(
        `/api/workspaces/${workspaceSlug}/user-issues/${userId}/`,
        {
          params,
        },
        config
      )
    );
  }

  async deactivateAccount() {
    return this.responseData(this.delete(`/api/users/me/`));
  }

  async leaveWorkspace(workspaceSlug: string): Promise<void> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/members/leave/`));
  }

  async joinProject(workspaceSlug: string, project_ids: string[]): Promise<{ message: string }> {
    return this.responseData(
      this.post(`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/`, { project_ids })
    );
  }

  async leaveProject(workspaceSlug: string, projectId: string): Promise<void> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/leave/`));
  }

  async checkEmail(token: string, email: string): Promise<IEmailCheckResponse> {
    return this.responseData(
      this.post(
        "/auth/email-check/",
        { email },
        {
          headers: {
            "X-CSRFTOKEN": token,
          },
        }
      )
    );
  }

  async generateEmailCode(data: { email: string }): Promise<unknown> {
    return this.responseData(this.post("/api/users/me/email/generate-code/", data));
  }

  async verifyEmailCode(data: { email: string; code: string }): Promise<unknown> {
    return this.responseData(this.patch("/api/users/me/email/", data));
  }
}

const userService = new UserService();

export default userService;
