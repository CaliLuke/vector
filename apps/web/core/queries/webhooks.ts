/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file in the project root for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { Query, QueryClient } from "@tanstack/react-query";
import type { IWebhook } from "@plane/types";
import { WebhookService } from "@/services/webhook.service";

const webhookService = new WebhookService();

export type IWebhookRecord = Omit<IWebhook, "secret_key">;
type TWebhookSecretResponse = IWebhook & { secret_key?: string };

export const webhookKeys = {
  all: ["webhooks"] as const,
  lists: () => [...webhookKeys.all, "list"] as const,
  list: (workspaceSlug: string) => [...webhookKeys.lists(), workspaceSlug] as const,
  details: () => [...webhookKeys.all, "detail"] as const,
  detail: (workspaceSlug: string, webhookId: string) => [...webhookKeys.details(), workspaceSlug, webhookId] as const,
};

function withoutSecret(webhook: TWebhookSecretResponse): IWebhookRecord {
  const { secret_key: _secretKey, ...record } = webhook;
  return record;
}

export function webhooksListOptions(workspaceSlug: string | undefined, isWorkspaceAdmin: boolean) {
  return queryOptions({
    queryKey: webhookKeys.list(workspaceSlug ?? ""),
    queryFn:
      workspaceSlug && isWorkspaceAdmin
        ? async ({ signal }) => (await webhookService.fetchWebhooksList(workspaceSlug, signal)).map(withoutSecret)
        : skipToken,
  });
}

export function webhookDetailsOptions(
  workspaceSlug: string | undefined,
  webhookId: string | undefined,
  isWorkspaceAdmin: boolean
) {
  return queryOptions({
    queryKey: webhookKeys.detail(workspaceSlug ?? "", webhookId ?? ""),
    queryFn:
      workspaceSlug && webhookId && isWorkspaceAdmin
        ? async ({ signal }) =>
            withoutSecret(await webhookService.fetchWebhookDetails(workspaceSlug, webhookId, signal))
        : skipToken,
  });
}

type MutationQueries = {
  list: Query | undefined;
  detail: Query | undefined;
};

function captureQueries(client: QueryClient, workspaceSlug: string, webhookId?: string): MutationQueries {
  return {
    list: client.getQueryCache().find({ queryKey: webhookKeys.list(workspaceSlug), exact: true }),
    detail: webhookId
      ? client.getQueryCache().find({ queryKey: webhookKeys.detail(workspaceSlug, webhookId), exact: true })
      : undefined,
  };
}

function isCurrentQuery(client: QueryClient, key: readonly unknown[], query: Query | undefined): query is Query {
  return !!query && client.getQueryCache().find({ queryKey: key, exact: true }) === query;
}

async function cancelCapturedQueries(client: QueryClient, queries: MutationQueries | undefined) {
  const captured = new Set([queries?.list, queries?.detail].filter((query): query is Query => !!query));
  if (captured.size === 0) return;
  await client.cancelQueries({ predicate: (query) => captured.has(query) });
}

async function cacheUpdatedWebhook(
  client: QueryClient,
  workspaceSlug: string,
  webhookId: string,
  updatedWebhook: IWebhookRecord,
  queries: MutationQueries | undefined
) {
  await cancelCapturedQueries(client, queries);
  if (isCurrentQuery(client, webhookKeys.detail(workspaceSlug, webhookId), queries?.detail)) {
    client.setQueryData(webhookKeys.detail(workspaceSlug, webhookId), updatedWebhook);
  }
  if (isCurrentQuery(client, webhookKeys.list(workspaceSlug), queries?.list)) {
    client.setQueryData<IWebhookRecord[]>(webhookKeys.list(workspaceSlug), (webhooks) =>
      webhooks?.map((webhook) => (webhook.id === updatedWebhook.id ? updatedWebhook : webhook))
    );
  }
}

export function createWebhookMutationOptions(client: QueryClient, workspaceSlug: string) {
  return mutationOptions({
    mutationKey: [...webhookKeys.list(workspaceSlug), "create"],
    retry: false,
    mutationFn: async ({
      data,
      exposeSecret,
    }: {
      data: Partial<IWebhook>;
      exposeSecret: (secret: string | null, webhook: IWebhookRecord) => void;
    }) => {
      const response = (await webhookService.createWebhook(workspaceSlug, data)) as TWebhookSecretResponse;
      const webhook = withoutSecret(response);
      exposeSecret(response.secret_key ?? null, webhook);
      return webhook;
    },
    onMutate: () => captureQueries(client, workspaceSlug),
    onSuccess: async (createdWebhook, _variables, queries) => {
      await cancelCapturedQueries(client, queries);
      if (!isCurrentQuery(client, webhookKeys.list(workspaceSlug), queries?.list)) return;
      client.setQueryData<IWebhookRecord[]>(webhookKeys.list(workspaceSlug), (webhooks) =>
        webhooks ? [createdWebhook, ...webhooks.filter(({ id }) => id !== createdWebhook.id)] : webhooks
      );
    },
  });
}

export function updateWebhookMutationOptions(client: QueryClient, workspaceSlug: string, webhookId: string) {
  return mutationOptions({
    mutationKey: [...webhookKeys.detail(workspaceSlug, webhookId), "update"],
    scope: { id: `webhook:${workspaceSlug}:${webhookId}` },
    retry: false,
    mutationFn: async (data: Partial<IWebhook>) =>
      withoutSecret((await webhookService.updateWebhook(workspaceSlug, webhookId, data)) as TWebhookSecretResponse),
    onMutate: () => captureQueries(client, workspaceSlug, webhookId),
    onSuccess: (updatedWebhook, _data, queries) =>
      cacheUpdatedWebhook(client, workspaceSlug, webhookId, updatedWebhook, queries),
  });
}

export function deleteWebhookMutationOptions(client: QueryClient, workspaceSlug: string, webhookId: string) {
  return mutationOptions({
    mutationKey: [...webhookKeys.detail(workspaceSlug, webhookId), "delete"],
    scope: { id: `webhook:${workspaceSlug}:${webhookId}` },
    retry: false,
    mutationFn: () => webhookService.deleteWebhook(workspaceSlug, webhookId),
    onMutate: () => ({ ...captureQueries(client, workspaceSlug, webhookId), sessionCurrent: false }),
    onSuccess: async (_result, _variables, queries) => {
      await cancelCapturedQueries(client, queries);
      if (!isCurrentQuery(client, webhookKeys.detail(workspaceSlug, webhookId), queries?.detail)) return;
      if (queries) queries.sessionCurrent = true;
      if (isCurrentQuery(client, webhookKeys.list(workspaceSlug), queries?.list)) {
        client.setQueryData<IWebhookRecord[]>(webhookKeys.list(workspaceSlug), (webhooks) =>
          webhooks?.filter((webhook) => webhook.id !== webhookId)
        );
      }
      if (isCurrentQuery(client, webhookKeys.detail(workspaceSlug, webhookId), queries?.detail)) {
        client.removeQueries({ queryKey: webhookKeys.detail(workspaceSlug, webhookId), exact: true });
      }
    },
  });
}

export function regenerateWebhookSecretMutationOptions(client: QueryClient, workspaceSlug: string, webhookId: string) {
  return mutationOptions({
    mutationKey: [...webhookKeys.detail(workspaceSlug, webhookId), "regenerate-secret"],
    scope: { id: `webhook:${workspaceSlug}:${webhookId}` },
    retry: false,
    mutationFn: async ({
      exposeSecret,
    }: {
      exposeSecret: (secret: string | null, webhook: IWebhookRecord) => void;
    }) => {
      const response = (await webhookService.regenerateSecretKey(workspaceSlug, webhookId)) as TWebhookSecretResponse;
      const webhook = withoutSecret(response);
      exposeSecret(response.secret_key ?? null, webhook);
      return webhook;
    },
    onMutate: () => captureQueries(client, workspaceSlug, webhookId),
    onSuccess: (updatedWebhook, _variables, queries) =>
      cacheUpdatedWebhook(client, workspaceSlug, webhookId, updatedWebhook, queries),
  });
}
