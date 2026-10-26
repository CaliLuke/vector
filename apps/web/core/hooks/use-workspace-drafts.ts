/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { TIssue, TWorkspaceDraftIssue } from "@plane/types";
import { useCurrentUser } from "./use-current-user";
import { useIdentityGeneration } from "./use-identity-generation";
import { useWorkspacePermissions } from "./use-permissions";
import {
  acceptWorkspaceDraftReceipt,
  captureWorkspaceDraftReceiptOwner,
  removePublishedWorkspaceDraft,
  workspaceDraftListOptions,
  workspaceDraftMutationOptions,
} from "@/queries/workspace-drafts";
import type { WorkspaceDraftScope, WorkspaceDraftWrite } from "@/queries/workspace-drafts";

function useWorkspaceDraftScope(workspaceSlug: string | undefined) {
  const client = useQueryClient();
  const account = useCurrentUser();
  const permissions = useWorkspacePermissions(workspaceSlug);
  const sessionGeneration = useIdentityGeneration(client);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const scope = useMemo<WorkspaceDraftScope>(
    () => ({
      accountId: account.isSuccess ? account.data?.id : undefined,
      sessionGeneration,
      workspaceSlug,
      membershipRole: permissions.workspaceRole,
      membershipRevision: membership?.updated_at,
    }),
    [
      account.data?.id,
      account.isSuccess,
      membership?.updated_at,
      permissions.workspaceRole,
      sessionGeneration,
      workspaceSlug,
    ]
  );
  const isError = account.isError || permissions.isError;
  const isPending = !isError && (!account.isSuccess || permissions.isPending);
  const isReady = Boolean(
    !isError &&
    account.isSuccess &&
    scope.accountId &&
    membership?.is_active === true &&
    scope.membershipRole !== undefined &&
    scope.membershipRevision
  );
  return {
    client,
    scope,
    isReady,
    isError,
    isPending,
    isDenied: !isError && !isPending && !isReady,
    retryAccess: async () => {
      if (account.isError) await account.refetch();
      else if (permissions.membershipQuery.isError) await permissions.membershipQuery.refetch();
    },
  };
}

export function useWorkspaceDraftList(workspaceSlug: string | undefined, enabled = true) {
  const access = useWorkspaceDraftScope(workspaceSlug);
  const query = useInfiniteQuery(workspaceDraftListOptions(access.client, access.scope, enabled && access.isReady));
  const pages = access.isReady && query.isSuccess ? query.data.pages : undefined;
  const issues = useMemo(() => pages?.flatMap((page) => page.results ?? []) ?? [], [pages]);
  return {
    ...query,
    issues,
    paginationInfo: pages?.[0],
    accessDenied: access.isDenied,
    isPending: access.isPending || (access.isReady && query.isPending),
    isError: access.isError || (access.isReady && query.isError),
    retry: access.isError ? access.retryAccess : query.refetch,
  };
}

export function useWorkspaceDraftMutations(workspaceSlug: string | undefined) {
  const access = useWorkspaceDraftScope(workspaceSlug);
  const mutation = useMutation(workspaceDraftMutationOptions(access.client, access.scope));
  const mutate = (write: WorkspaceDraftWrite) => mutation.mutateAsync(write);
  return {
    ...mutation,
    createDraft: async (data: Partial<TWorkspaceDraftIssue | TIssue>) => {
      const receipt = await mutate({ kind: "create", data });
      return receipt.kind === "create" ? receipt.saved : undefined;
    },
    updateDraft: async (issueId: string, data: Partial<TWorkspaceDraftIssue>) => {
      const receipt = await mutate({ kind: "update", issueId, data });
      return receipt.kind === "update" ? receipt.saved : undefined;
    },
    deleteDraft: async (issueId: string) => {
      await mutate({ kind: "delete", issueId });
    },
    captureReceiptOwner: (issueId: string) => captureWorkspaceDraftReceiptOwner(access.client, access.scope, issueId),
    completePublication: (issueId: string, owner: ReturnType<typeof captureWorkspaceDraftReceiptOwner>) =>
      removePublishedWorkspaceDraft(access.client, owner, issueId),
    acceptDraftReceipt: (issue: TWorkspaceDraftIssue, owner: ReturnType<typeof captureWorkspaceDraftReceiptOwner>) =>
      acceptWorkspaceDraftReceipt(access.client, owner, issue),
    accessDenied: access.isDenied,
    accessError: access.isError,
    accessPending: access.isPending,
    retryAccess: access.retryAccess,
  };
}
