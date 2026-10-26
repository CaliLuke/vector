/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type {
  TNotificationPaginatedInfo,
  TNotificationPaginatedInfoQueryParams,
  TNotification,
  TUnreadNotificationsCount,
} from "@plane/types";
// helpers
// services
import { APIService } from "@/services/api.service";

export class WorkspaceNotificationService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async fetchUnreadNotificationsCount(workspaceSlug: string, signal?: AbortSignal): Promise<TUnreadNotificationsCount> {
    const { data } = await this.get(`/api/workspaces/${workspaceSlug}/users/notifications/unread/`, { signal });
    return data;
  }

  async fetchNotifications(
    workspaceSlug: string,
    params: TNotificationPaginatedInfoQueryParams,
    signal?: AbortSignal
  ): Promise<TNotificationPaginatedInfo> {
    const { data } = await this.get(`/api/workspaces/${workspaceSlug}/users/notifications/`, {
      params,
      signal,
    });
    return data;
  }

  async updateNotificationById(
    workspaceSlug: string,
    notificationId: string,
    payload: Partial<TNotification>
  ): Promise<TNotification | undefined> {
    const { data } = await this.patch(
      `/api/workspaces/${workspaceSlug}/users/notifications/${notificationId}/`,
      payload
    );
    return data || undefined;
  }

  async markNotificationAsRead(workspaceSlug: string, notificationId: string): Promise<TNotification | undefined> {
    const { data } = await this.post(`/api/workspaces/${workspaceSlug}/users/notifications/${notificationId}/read/`);
    return data || undefined;
  }

  async markNotificationAsUnread(workspaceSlug: string, notificationId: string): Promise<TNotification | undefined> {
    const { data } = await this.delete(`/api/workspaces/${workspaceSlug}/users/notifications/${notificationId}/read/`);
    return data || undefined;
  }

  async markNotificationAsArchived(workspaceSlug: string, notificationId: string): Promise<TNotification | undefined> {
    const { data } = await this.post(`/api/workspaces/${workspaceSlug}/users/notifications/${notificationId}/archive/`);
    return data || undefined;
  }

  async markNotificationAsUnArchived(
    workspaceSlug: string,
    notificationId: string
  ): Promise<TNotification | undefined> {
    const { data } = await this.delete(
      `/api/workspaces/${workspaceSlug}/users/notifications/${notificationId}/archive/`
    );
    return data || undefined;
  }

  async markAllNotificationsAsRead(
    workspaceSlug: string,
    payload: TNotificationPaginatedInfoQueryParams
  ): Promise<TNotification | undefined> {
    const { data } = await this.post(`/api/workspaces/${workspaceSlug}/users/notifications/mark-all-read/`, payload);
    return data || undefined;
  }
}

const workspaceNotificationService = new WorkspaceNotificationService();

export default workspaceNotificationService;
