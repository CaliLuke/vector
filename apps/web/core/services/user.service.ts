/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// services
import { API_BASE_URL } from "@plane/constants";
import type {
  TIssue,
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
    params: any
  ): Promise<
    | {
        [key: string]: TIssue[];
      }
    | TIssue[]
  > {
    return this.get(`/api/workspaces/${workspaceSlug}/my-issues/`, {
      params,
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
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
    return this.get("/api/users/me/profile/", { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }
  async updateCurrentUserProfile(data: Partial<TUserProfile>, signal?: AbortSignal): Promise<TUserProfile> {
    return this.patch("/api/users/me/profile/", data, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  async currentUserSettings(bustCache: boolean = false, signal?: AbortSignal): Promise<IUserSettings> {
    const url = bustCache ? `/api/users/me/settings/?t=${Date.now()}` : "/api/users/me/settings/";
    return this.get(url, { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  async currentUserEmailNotificationSettings(signal?: AbortSignal): Promise<IUserEmailNotificationSettings> {
    return this.get("/api/users/me/notification-preferences/", { signal })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response ?? error;
      });
  }

  async updateUser(data: Partial<IUser>): Promise<IUser> {
    return this.patch("/api/users/me/", data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async updateUserOnBoard(signal?: AbortSignal): Promise<{ message: string }> {
    return this.patch(
      "/api/users/me/onboard/",
      {
        is_onboarded: true,
      },
      { signal }
    )
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async updateUserTourCompleted(signal?: AbortSignal): Promise<{ message: string }> {
    return this.patch(
      "/api/users/me/tour-completed/",
      {
        is_tour_completed: true,
      },
      { signal }
    )
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async updateCurrentUserEmailNotificationSettings(
    data: Partial<IUserEmailNotificationSettings>
  ): Promise<IUserEmailNotificationSettings> {
    return this.patch("/api/users/me/notification-preferences/", data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async changePassword(token: string, data: { old_password?: string; new_password: string }): Promise<any> {
    return this.post(`/auth/change-password/`, data, {
      headers: {
        "X-CSRFTOKEN": token,
      },
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data ?? error;
      });
  }

  async getUserProfileData(workspaceSlug: string, userId: string): Promise<IUserProfileData> {
    return this.get(`/api/workspaces/${workspaceSlug}/user-stats/${userId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getUserProfileProjectsSegregation(
    workspaceSlug: string,
    userId: string
  ): Promise<IUserProfileProjectSegregation> {
    return this.get(`/api/workspaces/${workspaceSlug}/user-profile/${userId}/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getUserProfileActivity(
    workspaceSlug: string,
    userId: string,
    params: {
      per_page: number;
      cursor?: string;
    }
  ): Promise<IUserActivityResponse> {
    return this.get(`/api/workspaces/${workspaceSlug}/user-activity/${userId}/`, {
      params,
    })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async downloadProfileActivity(
    workspaceSlug: string,
    userId: string,
    data: {
      date: string;
    }
  ): Promise<any> {
    return this.post(`/api/workspaces/${workspaceSlug}/user-activity/${userId}/export/`, data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async getUserProfileIssues(
    workspaceSlug: string,
    userId: string,
    params: any,
    config = {}
  ): Promise<TIssuesResponse> {
    return this.get(
      `/api/workspaces/${workspaceSlug}/user-issues/${userId}/`,
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

  async deactivateAccount() {
    return this.delete(`/api/users/me/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async leaveWorkspace(workspaceSlug: string): Promise<void> {
    return this.post(`/api/workspaces/${workspaceSlug}/members/leave/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async joinProject(workspaceSlug: string, project_ids: string[]): Promise<{ message: string }> {
    return this.post(`/api/users/me/workspaces/${workspaceSlug}/projects/invitations/`, { project_ids })
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async leaveProject(workspaceSlug: string, projectId: string): Promise<void> {
    return this.post(`/api/workspaces/${workspaceSlug}/projects/${projectId}/members/leave/`)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async checkEmail(token: string, email: string): Promise<IEmailCheckResponse> {
    return this.post(
      "/auth/email-check/",
      { email },
      {
        headers: {
          "X-CSRFTOKEN": token,
        },
      }
    )
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async generateEmailCode(data: { email: string }): Promise<any> {
    return this.post("/api/users/me/email/generate-code/", data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }

  async verifyEmailCode(data: { email: string; code: string }): Promise<any> {
    return this.patch("/api/users/me/email/", data)
      .then((response) => response?.data)
      .catch((error) => {
        throw error?.response?.data;
      });
  }
}

const userService = new UserService();

export default userService;
