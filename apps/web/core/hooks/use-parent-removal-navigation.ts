/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useEffect } from "react";
import { useRouter } from "@tanstack/react-router";
import { useCommittedScope } from "@/hooks/use-committed-scope";
/** Route ownership survives a deleted row's dialog; null is the native accepted-removal receipt. */
export function useParentRemovalNavigation(scopeKey: string, removed: boolean, href: string) {
  const router = useRouter(),
    current = useCommittedScope(scopeKey);
  useEffect(() => {
    if (removed && current()) void router.navigate({ href, replace: true });
  }, [current, href, removed, router]);
}
