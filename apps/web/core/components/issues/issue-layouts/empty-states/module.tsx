/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useState } from "react";

// plane imports

import { useTranslation } from "@plane/i18n";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { ISearchIssueResponse } from "@plane/types";
import { EIssuesStoreType, EUserProjectRoles } from "@plane/types";
// components
import { ExistingIssuesListModal } from "@/components/core/modals/existing-issues-list-modal";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useWorkItemFilterUi } from "@/providers/work-item-filter-ui";
import { FilteredIssuesEmptyState } from "./filtered-issues";

export function ModuleEmptyState() {
  // router
  const { workspaceSlug, projectId, moduleId } = useParams({ strict: false });
  // states
  const [moduleIssuesListModal, setModuleIssuesListModal] = useState(false);
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const actions = useIssuesActions(EIssuesStoreType.MODULE);
  const { toggleCreateIssueModal } = useCommandPalette();
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const { state: filterUi } = useWorkItemFilterUi();
  // derived values
  const canPerformEmptyStateActions = canProject([EUserProjectRoles.ADMIN, EUserProjectRoles.MEMBER]);

  const handleAddIssuesToModule = async (data: ISearchIssueResponse[]) => {
    if (!workspaceSlug || !projectId || !moduleId) return;

    const issueIds = data.map((i) => i.id);
    await actions
      .addIssuesToModule(projectId, moduleId, issueIds)
      .then(() =>
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Success!",
          message: "Work items added to the module successfully.",
        })
      )
      .catch(() =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "Selected work items could not be added to the module. Please try again.",
        })
      );
  };

  return (
    <div className="relative h-full w-full overflow-y-auto">
      <ExistingIssuesListModal
        workspaceSlug={workspaceSlug?.toString()}
        projectId={projectId?.toString()}
        isOpen={moduleIssuesListModal}
        handleClose={() => setModuleIssuesListModal(false)}
        searchParams={{ module: moduleId != undefined ? moduleId : "" }}
        handleOnSubmit={handleAddIssuesToModule}
      />
      <div className="grid h-full w-full place-items-center">
        {filterUi.hasActiveFilters ? (
          <FilteredIssuesEmptyState
            clearFilters={filterUi.clearFilters}
            canClearFilters={canPerformEmptyStateActions}
            clearFiltersLabel="Clear filters"
          />
        ) : (
          <EmptyStateDetailed
            assetKey="work-item"
            title={t("project_empty_state.module_work_items.title")}
            description={t("project_empty_state.module_work_items.description")}
            actions={[
              {
                label: t("project_empty_state.module_work_items.cta_primary"),
                onClick: () => {
                  toggleCreateIssueModal(true, EIssuesStoreType.MODULE);
                },
                disabled: !canPerformEmptyStateActions,
                variant: "primary",
              },
              {
                label: t("project_empty_state.module_work_items.cta_secondary"),
                onClick: () => setModuleIssuesListModal(true),
                disabled: !canPerformEmptyStateActions,
                variant: "secondary",
              },
            ]}
          />
        )}
      </div>
    </div>
  );
}
