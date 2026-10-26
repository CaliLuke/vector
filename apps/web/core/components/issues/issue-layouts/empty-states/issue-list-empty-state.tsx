/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useParams } from "@tanstack/react-router";
import { EUserProjectRoles } from "@plane/types";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useWorkItemFilterUi } from "@/providers/work-item-filter-ui";
import { FilteredIssuesEmptyState } from "./filtered-issues";

type Props = {
  clearFiltersLabel: string;
  children: ReactNode | ((canPerformActions: boolean) => ReactNode);
};

export function IssueListEmptyState(props: Props) {
  const { clearFiltersLabel, children } = props;
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const { can } = useProjectPermissions(workspaceSlug, projectId);
  const { state } = useWorkItemFilterUi();
  const canClearFilters = can([EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER]);
  const content = typeof children === "function" ? children(canClearFilters) : children;

  return (
    <div className="relative h-full w-full overflow-y-auto">
      {state.hasActiveFilters ? (
        <FilteredIssuesEmptyState
          clearFilters={state.clearFilters}
          canClearFilters={canClearFilters}
          clearFiltersLabel={clearFiltersLabel}
        />
      ) : (
        content
      )}
    </div>
  );
}
