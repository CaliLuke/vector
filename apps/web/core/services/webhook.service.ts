/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// api services
import { API_BASE_URL } from "@plane/constants";
import type { IWebhook } from "@plane/types";
import { APIService } from "@/services/api.service";
// helpers
// types

export class WebhookService extends APIService {
  constructor() {
    super(API_BASE_URL);
  }

  async fetchWebhooksList(workspaceSlug: string, signal?: AbortSignal): Promise<IWebhook[]> {
    return this.get(`/api/workspaces/${workspaceSlug}/webhooks/`, { signal })
      .then((response) => response?.data)
      .catch(throwWebhookError);
  }

  async fetchWebhookDetails(workspaceSlug: string, webhookId: string, signal?: AbortSignal): Promise<IWebhook> {
    return this.get(`/api/workspaces/${workspaceSlug}/webhooks/${webhookId}/`, { signal })
      .then((response) => response?.data)
      .catch(throwWebhookError);
  }

  async createWebhook(workspaceSlug: string, data = {}): Promise<IWebhook> {
    return this.post(`/api/workspaces/${workspaceSlug}/webhooks/`, data)
      .then((response) => response?.data)
      .catch(throwWebhookError);
  }

  async updateWebhook(workspaceSlug: string, webhookId: string, data = {}): Promise<IWebhook> {
    return this.patch(`/api/workspaces/${workspaceSlug}/webhooks/${webhookId}/`, data)
      .then((response) => response?.data)
      .catch(throwWebhookError);
  }

  async deleteWebhook(workspaceSlug: string, webhookId: string): Promise<void> {
    return this.delete(`/api/workspaces/${workspaceSlug}/webhooks/${webhookId}/`)
      .then((response) => response?.data)
      .catch(throwWebhookError);
  }

  async regenerateSecretKey(workspaceSlug: string, webhookId: string): Promise<IWebhook> {
    return this.post(`/api/workspaces/${workspaceSlug}/webhooks/${webhookId}/regenerate/`)
      .then((response) => response?.data)
      .catch(throwWebhookError);
  }
}

function throwWebhookError(error: unknown): never {
  if (error && typeof error === "object" && "response" in error) {
    const response = Reflect.get(error, "response");
    if (response && typeof response === "object") {
      const data = Reflect.get(response, "data");
      if (data !== undefined && data !== null) throw data;
    }
  }
  if (error instanceof Error) throw error;
  throw new Error("Webhook request failed");
}
