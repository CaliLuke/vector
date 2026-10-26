/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { infiniteQueryOptions, mutationOptions } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { InstanceWorkspaceService } from "@plane/services";
import type { IWorkspace } from "@plane/types";
import { instanceKeys } from "./instance";

const service = new InstanceWorkspaceService();
export const instanceWorkspacesOptions = infiniteQueryOptions({
  queryKey: ["instance", "workspaces"],
  initialPageParam: undefined as string | undefined,
  queryFn: ({ pageParam, signal }) => service.list(pageParam, signal),
  getNextPageParam: (lastPage) => (lastPage.next_page_results ? lastPage.next_cursor : undefined),
  staleTime: 30_000,
});
export function createWorkspaceOptions(client: QueryClient) {
  return mutationOptions({
    mutationFn: (data: Partial<IWorkspace>) => service.create(data),
    onSuccess: async () => {
      await Promise.all([
        client.invalidateQueries({ queryKey: instanceWorkspacesOptions.queryKey }),
        client.invalidateQueries({ queryKey: instanceKeys.info }),
      ]);
    },
  });
}
