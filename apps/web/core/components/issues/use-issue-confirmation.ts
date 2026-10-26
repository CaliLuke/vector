/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useRef, useState } from "react";
import { useLocation, useParams } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import {
  captureIdentityScope,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "@/queries/identity";
import { isIssuePermissionScopeReady, submitIssueDeleteConfirmation } from "./issue-delete-modal-feedback";

/** Scope a mounted confirmation to the selected entity, route, and account session. */
export function useIssueConfirmationScope(entity: readonly unknown[]) {
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const pathname = useLocation({ select: (location) => location.pathname });
  const client = useQueryClient();
  const { data: currentUser } = useCurrentUser();
  const generation = useIdentityGeneration(client);
  return { workspaceSlug, scopeKey: JSON.stringify([workspaceSlug, pathname, ...entity, currentUser?.id, generation]) };
}

/** Keep confirmation feedback and dismissal attached to its committed identity and dialog. */
export function useIssueConfirmation(
  scopeKey: string,
  permissions: { isPending: boolean; isError: boolean },
  onClose: () => void
) {
  const client = useQueryClient();
  const currentUserQuery = useCurrentUser();
  const currentUser = currentUserQuery.data;
  const generation = useIdentityGeneration(client);
  const mounted = useCommittedScope(scopeKey);
  const identity = captureIdentityScope(client, currentUser?.id);
  const [isPending, setIsPending] = useState(false);
  const pending = useRef(false);
  const isCurrent = () =>
    mounted() &&
    Boolean(currentUser?.id && identity) &&
    generation === getIdentitySessionGeneration(client) &&
    isIdentityScopeCurrent(client, identity);
  const canStart = isIssuePermissionScopeReady({
    hasAccount: Boolean(currentUser?.id),
    identityVerified: isIdentityScopeVerified(client, identity),
    currentUserLoaded: currentUserQuery.isSuccess,
    permissionsPending: permissions.isPending,
    permissionsError: permissions.isError,
  });
  const setPending = (value: boolean) => {
    pending.current = value;
    setIsPending(value);
  };
  const closeWhenIdle = () => {
    if (pending.current || !isCurrent()) return;
    onClose();
  };
  const submitDelete = (
    options: Omit<
      Parameters<typeof submitIssueDeleteConfirmation>[0],
      "canStart" | "isCurrent" | "onClose" | "setIsDeleting"
    >
  ) => {
    if (pending.current) return;
    void submitIssueDeleteConfirmation({
      ...options,
      canStart,
      isCurrent,
      onClose: closeWhenIdle,
      setIsDeleting: setPending,
    });
  };
  return {
    submitDelete,
    currentUserQuery,
    currentUser,
    isCurrent,
    canStart,
    accessError: currentUserQuery.isError || permissions.isError,
    isPending,
    pending,
    setPending,
    closeWhenIdle,
  };
}
