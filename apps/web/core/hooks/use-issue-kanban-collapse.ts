/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
import type { TSupportedFilterTypeForUpdate } from "@plane/constants";
import { EIssueFilterType } from "@plane/constants";
import type { TIssueKanbanFilters, TSupportedFilterForUpdate } from "@plane/types";
import { EIssuesStoreType } from "@plane/types";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { useProjectKanbanFilters } from "./use-project-kanban-filters";

type TFilterUpdate = (
  projectId: string,
  filterType: TSupportedFilterTypeForUpdate,
  filters: TSupportedFilterForUpdate
) => Promise<void>;

export function useIssueKanbanCollapse(
  storeType: EIssuesStoreType,
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  storedFilters: TIssueKanbanFilters | undefined,
  updateFilters: TFilterUpdate | undefined
) {
  const isProjectStore = storeType === EIssuesStoreType.PROJECT;
  const { filters: projectFilters, toggle: toggleProjectFilter } = useProjectKanbanFilters(
    isProjectStore ? workspaceSlug : undefined,
    isProjectStore ? projectId : undefined
  );
  const filters = isProjectStore ? projectFilters : (storedFilters ?? { group_by: [], sub_group_by: [] });

  const toggle = useCallback(
    (group: keyof TIssueKanbanFilters, value: string) => {
      if (!workspaceSlug) return;
      if (isProjectStore) {
        toggleProjectFilter(group, value);
        return;
      }
      if (!workspaceSlug || !updateFilters) {
        setToast({
          title: "Error!",
          type: TOAST_TYPE.ERROR,
          message: "Issue group preferences are unavailable for this view.",
        });
        return;
      }

      const collapsed = storedFilters?.[group] ?? [];
      const updated = collapsed.includes(value)
        ? collapsed.filter((groupId) => groupId !== value)
        : [...collapsed, value];
      void updateFilters(projectId ?? "", EIssueFilterType.KANBAN_FILTERS, { [group]: updated }).catch((error) => {
        setToast({
          title: "Error!",
          type: TOAST_TYPE.ERROR,
          message: error instanceof Error ? error.message : "Failed to update issue group preferences.",
        });
      });
    },
    [isProjectStore, projectId, storedFilters, toggleProjectFilter, updateFilters, workspaceSlug]
  );

  return { filters, toggle };
}
