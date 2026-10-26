/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { forgetItemReads, isItemAccessDenied, itemAccessOptions, retryItemAccess } from "@/queries/work-item-access";
import type { ItemAccessScope } from "@/queries/work-item-access";

/** Shared denial metadata; server records remain in their native collection queries. */
export function useItemAccess(scope: ItemAccessScope) {
  const client = useQueryClient();
  const denied = useQuery(itemAccessOptions(scope));
  const { accountId, generation, workspaceSlug, projectId, issueId, serviceType } = scope;
  useEffect(() => {
    const currentScope = { accountId, generation, workspaceSlug, projectId, issueId, serviceType };
    if (denied.data && isItemAccessDenied(client, currentScope)) forgetItemReads(client, currentScope);
  }, [client, denied.data, accountId, generation, workspaceSlug, projectId, issueId, serviceType]);
  return { isDenied: denied.data === true, retry: () => retryItemAccess(client, scope) };
}
