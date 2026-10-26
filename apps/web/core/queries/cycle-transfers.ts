/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { mutationOptions } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { ICycle } from "@plane/types";
import type { ProjectCatalogueScope } from "./catalogue-scope";
import { parentCatalogueKeys } from "./cycle-modules";
import { captureParentAccess } from "./cycle-module-scope";
import { cycleIsCompleted } from "./cycle-module-selectors";
import { isAuthorizationError } from "./authorization-errors";
import { CycleService } from "@/services/cycle.service";
const service = new CycleService();
export function cycleTransferOptions(client: QueryClient, scope: ProjectCatalogueScope, sourceId: string) {
  return mutationOptions({
    mutationKey: [...parentCatalogueKeys.scope("cycle", scope), "transfer", sourceId],
    scope: { id: JSON.stringify([...parentCatalogueKeys.owner("cycle", scope), scope.projectId, "write"]) },
    retry: false,
    mutationFn: async (targetId: string) => {
      const access = captureParentAccess(client, scope),
        key = parentCatalogueKeys.list("cycle", scope),
        source = client.getQueryCache().find({ queryKey: key, exact: true });
      const rows = client.getQueryData<ICycle[]>(key),
        target = rows?.find((row) => row.id === targetId);
      if (
        !scope.workspaceSlug ||
        !scope.projectId ||
        !access.isVerified() ||
        ![15, 20].includes(scope.projectRole ?? 0) ||
        source?.state.status !== "success" ||
        sourceId === targetId ||
        !target ||
        target.project_id !== scope.projectId ||
        cycleIsCompleted(target)
      )
        throw new Error("Reload the available cycles before transferring work items.");
      const related = new Set(
        client
          .getQueryCache()
          .findAll({ queryKey: parentCatalogueKeys.scope("cycle", scope) })
          .filter(
            (query) =>
              query.queryKey.at(-2) === "list" || query.queryKey.includes(sourceId) || query.queryKey.includes(targetId)
          )
      );
      await service.transferIssues(scope.workspaceSlug, scope.projectId, sourceId, { new_cycle_id: targetId });
      const isCurrent = () =>
        access.isCurrent() &&
        !isAuthorizationError(source.state.error) &&
        client.getQueryCache().find({ queryKey: key, exact: true }) === source;
      const refresh = async () => {
        if (!isCurrent() || !access.isVerified()) return;
        await client.cancelQueries({ predicate: (query) => related.has(query) });
        if (isCurrent() && access.isVerified())
          await client.invalidateQueries({ predicate: (query) => related.has(query) }, { throwOnError: true });
      };
      return { captured: { isCurrent }, refresh };
    },
  });
}
