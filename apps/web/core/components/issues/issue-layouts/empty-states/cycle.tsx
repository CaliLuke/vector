/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useState } from "react";
import isEmpty from "lodash-es/isEmpty.js";

// plane imports
import { WORK_ITEM_TRACKER_ELEMENTS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { ISearchIssueResponse } from "@plane/types";
import { EIssuesStoreType, EUserProjectRoles } from "@plane/types";
// components
import { ExistingIssuesListModal } from "@/components/core/modals/existing-issues-list-modal";
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useCycleDetails } from "@/hooks/use-cycles";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useWorkItemFilterUi } from "@/providers/work-item-filter-ui";
import { FilteredIssuesEmptyState } from "./filtered-issues";

export function CycleEmptyState() {
  // router
  const { workspaceSlug, projectId, cycleId } = useParams({ strict: false });
  // states
  const [cycleIssuesListModal, setCycleIssuesListModal] = useState(false);
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const { cycle: cycleDetails } = useCycleDetails(workspaceSlug, projectId, cycleId);
  const actions = useIssuesActions(EIssuesStoreType.CYCLE);
  const { toggleCreateIssueModal } = useCommandPalette();
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const { state: filterUi } = useWorkItemFilterUi();
  // derived values
  const isCompletedCycleSnapshotAvailable = !isEmpty(cycleDetails?.progress_snapshot ?? {});
  const isCompletedAndEmpty = isCompletedCycleSnapshotAvailable || cycleDetails?.status?.toLowerCase() === "completed";
  const canPerformEmptyStateActions = canProject([EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER]);

  const handleAddIssuesToCycle = async (data: ISearchIssueResponse[]) => {
    if (!workspaceSlug || !projectId || !cycleId) return;

    const issueIds = data.map((i) => i.id);

    await actions
      .addIssuesToCycle(projectId, cycleId, issueIds)
      .then(() =>
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Success!",
          message: "Work items added to the cycle successfully.",
        })
      )
      .catch(() =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "Selected work items could not be added to the cycle. Please try again.",
        })
      );
  };

  return (
    <div className="relative h-full w-full overflow-y-auto">
      <ExistingIssuesListModal
        workspaceSlug={workspaceSlug?.toString()}
        projectId={projectId?.toString()}
        isOpen={cycleIssuesListModal}
        handleClose={() => setCycleIssuesListModal(false)}
        searchParams={{ cycle: true }}
        handleOnSubmit={handleAddIssuesToCycle}
      />
      <div className="grid h-full w-full place-items-center">
        {isCompletedAndEmpty ? (
          // TODO: Empty state ux copy needs to be updated
          <EmptyStateDetailed
            assetKey="work-item"
            title={t("project_cycles.empty_state.completed_no_issues.title")}
            description={t("project_cycles.empty_state.completed_no_issues.description")}
          />
        ) : filterUi.hasActiveFilters ? (
          <FilteredIssuesEmptyState
            clearFilters={filterUi.clearFilters}
            canClearFilters={canPerformEmptyStateActions}
            clearFiltersLabel="Clear filters"
          />
        ) : (
          <EmptyStateDetailed
            assetKey="work-item"
            title={t("project_empty_state.cycle_work_items.title")}
            description={t("project_empty_state.cycle_work_items.description")}
            actions={[
              {
                label: t("project_empty_state.cycle_work_items.cta_primary"),
                onClick: () => {
                  toggleCreateIssueModal(true, EIssuesStoreType.CYCLE);
                },
                disabled: !canPerformEmptyStateActions,
                variant: "primary",
                "data-ph-element": WORK_ITEM_TRACKER_ELEMENTS.EMPTY_STATE_ADD_BUTTON.CYCLE,
              },
              {
                label: t("project_empty_state.cycle_work_items.cta_secondary"),
                onClick: () => setCycleIssuesListModal(true),
                disabled: !canPerformEmptyStateActions,
                variant: "secondary",
                "data-ph-element": WORK_ITEM_TRACKER_ELEMENTS.EMPTY_STATE_ADD_BUTTON.CYCLE,
              },
            ]}
          />
        )}
      </div>
    </div>
  );
}
