/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import type {
  TUnreadNotificationsCount,
  TNotificationPaginatedInfo,
  TNotification,
  TNotificationPaginatedInfoQueryParams,
} from "@plane/types";
// services
import { APIService } from "../api.service";

export class WorkspaceNotificationService extends APIService {
  constructor(BASE_URL?: string) {
    super(BASE_URL || API_BASE_URL);
  }

  /**
   * Retrieves the count of unread notifications for a workspace
   * @param {string} workspaceSlug - The unique identifier for the workspace
   * @returns {Promise<TUnreadNotificationsCount | undefined>} The count of unread notifications
   */
  async getUnreadCount(workspaceSlug: string): Promise<TUnreadNotificationsCount | undefined> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/users/notifications/unread/`));
  }

  /**
   * Retrieves paginated notifications for a workspace
   * @param {string} workspaceSlug - The unique identifier for the workspace
   * @param {TNotificationPaginatedInfoQueryParams} params - Query parameters for pagination and filtering
   * @returns {Promise<TNotificationPaginatedInfo | undefined>} Paginated list of notifications
   */
  async list(
    workspaceSlug: string,
    params: TNotificationPaginatedInfoQueryParams
  ): Promise<TNotificationPaginatedInfo | undefined> {
    return this.responseData(this.get(`/api/workspaces/${workspaceSlug}/users/notifications/`, { params }));
  }

  /**
   * Updates a specific notification by ID
   * @param {string} workspaceSlug - The unique identifier for the workspace
   * @param {string} notificationId - The unique identifier for the notification
   * @param {Partial<TNotification>} data - The notification data to update
   * @returns {Promise<TNotification | undefined>} The updated notification
   */
  async update(
    workspaceSlug: string,
    notificationId: string,
    data: Partial<TNotification>
  ): Promise<TNotification | undefined> {
    return this.responseData(
      this.patch(`/api/workspaces/${workspaceSlug}/users/notifications/${notificationId}/`, data)
    );
  }

  /**
   * Marks a notification as read
   * @param {string} workspaceSlug - The unique identifier for the workspace
   * @param {string} notificationId - The unique identifier for the notification
   * @returns {Promise<TNotification | undefined>} The updated notification
   */
  async markAsRead(workspaceSlug: string, notificationId: string): Promise<TNotification | undefined> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/users/notifications/${notificationId}/read/`));
  }

  /**
   * Marks a notification as unread
   * @param {string} workspaceSlug - The unique identifier for the workspace
   * @param {string} notificationId - The unique identifier for the notification
   * @returns {Promise<TNotification | undefined>} The updated notification
   */
  async markAsUnread(workspaceSlug: string, notificationId: string): Promise<TNotification | undefined> {
    return this.responseData(
      this.delete(`/api/workspaces/${workspaceSlug}/users/notifications/${notificationId}/read/`)
    );
  }

  /**
   * Archives a notification
   * @param {string} workspaceSlug - The unique identifier for the workspace
   * @param {string} notificationId - The unique identifier for the notification
   * @returns {Promise<TNotification | undefined>} The updated notification
   */
  async archive(workspaceSlug: string, notificationId: string): Promise<TNotification | undefined> {
    return this.responseData(
      this.post(`/api/workspaces/${workspaceSlug}/users/notifications/${notificationId}/archive/`)
    );
  }

  /**
   * Unarchives a notification
   * @param {string} workspaceSlug - The unique identifier for the workspace
   * @param {string} notificationId - The unique identifier for the notification
   * @returns {Promise<TNotification | undefined>} The updated notification
   */
  async unarchive(workspaceSlug: string, notificationId: string): Promise<TNotification | undefined> {
    return this.responseData(
      this.delete(`/api/workspaces/${workspaceSlug}/users/notifications/${notificationId}/archive/`)
    );
  }

  /**
   * Marks all notifications as read based on filter criteria
   * @param {string} workspaceSlug - The unique identifier for the workspace
   * @param {TNotificationPaginatedInfoQueryParams} data - Filter criteria for notifications to mark as read
   * @returns {Promise<TNotification | undefined>} The result of the operation
   */
  async markAllAsRead(
    workspaceSlug: string,
    data: TNotificationPaginatedInfoQueryParams
  ): Promise<TNotification | undefined> {
    return this.responseData(this.post(`/api/workspaces/${workspaceSlug}/users/notifications/mark-all-read/`, data));
  }
}
