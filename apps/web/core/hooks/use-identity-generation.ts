/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useSyncExternalStore } from "react";
import type { QueryClient } from "@tanstack/react-query";
import { getIdentitySessionGeneration, subscribeIdentitySession } from "@/queries/identity";

export function useIdentityGeneration(client: QueryClient) {
  const subscribe = useCallback((listener: () => void) => subscribeIdentitySession(client, listener), [client]);
  const getSnapshot = useCallback(() => getIdentitySessionGeneration(client), [client]);
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
