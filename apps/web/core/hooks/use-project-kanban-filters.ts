/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useState } from "react";
import type { TIssueKanbanFilters } from "@plane/types";
import { getProjectKanbanFilters, saveProjectKanbanFilters } from "../helpers/project-kanban-filters";
import { useCurrentUserId } from "@/hooks/use-current-user";

type TScopedKanbanFilters = {
  scope: string;
  filters: TIssueKanbanFilters;
};

export function useProjectKanbanFilters(workspaceSlug: string | undefined, projectId: string | undefined) {
  const userId = useCurrentUserId();
  const scope = workspaceSlug && projectId && userId ? `${workspaceSlug}:${projectId}:${userId}` : "";
  const read = useCallback(
    () => getProjectKanbanFilters(workspaceSlug, projectId, userId),
    [projectId, userId, workspaceSlug]
  );
  const [state, setState] = useState<TScopedKanbanFilters>(() => ({ scope, filters: read() }));
  let filters = state.filters;
  if (state.scope !== scope) {
    filters = read();
    setState({ scope, filters });
  }

  const update = useCallback(
    (patch: Partial<TIssueKanbanFilters>) => {
      if (!workspaceSlug || !projectId || !userId) return;
      const next = saveProjectKanbanFilters(workspaceSlug, projectId, userId, patch);
      setState({ scope, filters: next });
    },
    [projectId, scope, userId, workspaceSlug]
  );

  const toggle = useCallback(
    (group: keyof TIssueKanbanFilters, value: string) => {
      const current = read();
      const updated = current[group].includes(value)
        ? current[group].filter((groupId) => groupId !== value)
        : [...current[group], value];
      update(group === "group_by" ? { group_by: updated } : { sub_group_by: updated });
    },
    [read, update]
  );

  return { filters, update, toggle };
}
