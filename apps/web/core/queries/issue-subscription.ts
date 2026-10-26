/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { EIssueServiceType } from "@plane/types";
import { IssueService } from "@/services/issue/issue.service";
import { captureIdentityScope, isIdentityScopeCurrent, isIdentityScopeVerified } from "./identity";

export type IssueSubscriptionScope = {
  userId: string | undefined;
  workspaceSlug: string | undefined;
  projectId: string | undefined;
  issueId: string | undefined;
  serviceType: EIssueServiceType;
};

export function issueSubscriptionOptions(scope: IssueSubscriptionScope) {
  const { userId, workspaceSlug, projectId, issueId, serviceType } = scope;
  return queryOptions({
    queryKey: ["issue-subscription", userId, workspaceSlug, projectId, serviceType, issueId] as const,
    queryFn:
      userId && workspaceSlug && projectId && issueId
        ? async ({ client, signal }) => {
            const identity = captureIdentityScope(client, userId);
            if (!isIdentityScopeVerified(client, identity))
              throw new Error("The account session changed before subscription status could load.");
            const result = await new IssueService(serviceType).getIssueNotificationSubscriptionStatus(
              workspaceSlug,
              projectId,
              issueId,
              signal
            );
            if (!isIdentityScopeVerified(client, identity))
              throw new Error("The account session changed while subscription status loaded.");
            return result.subscribed;
          }
        : skipToken,
    staleTime: 60_000,
    refetchOnMount: false,
  });
}

export function updateIssueSubscriptionOptions(client: QueryClient, scope: IssueSubscriptionScope) {
  const identity = captureIdentityScope(client, scope.userId);
  const { workspaceSlug, projectId, issueId, serviceType } = scope;
  const { queryKey } = issueSubscriptionOptions(scope);
  return mutationOptions({
    mutationKey: [...queryKey, "update"],
    scope: { id: JSON.stringify(queryKey) },
    retry: false,
    onMutate: () => client.getQueryCache().find({ queryKey, exact: true }),
    mutationFn: async (subscribed: boolean) => {
      if (!workspaceSlug || !projectId || !issueId || !isIdentityScopeVerified(client, identity))
        throw new Error("The account or work item scope has changed.");
      const service = new IssueService(serviceType);
      if (subscribed) await service.subscribeToIssueNotifications(workspaceSlug, projectId, issueId);
      else await service.unsubscribeFromIssueNotifications(workspaceSlug, projectId, issueId);
      return subscribed;
    },
    onSuccess: async (subscribed, _, query) => {
      const current = () =>
        isIdentityScopeCurrent(client, identity) &&
        query &&
        client.getQueryCache().find({ queryKey, exact: true }) === query;
      if (!current()) return;
      await client.cancelQueries({ queryKey, exact: true });
      if (current()) client.setQueryData(queryKey, subscribed);
    },
  });
}
