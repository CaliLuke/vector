/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports

import { useTranslation } from "@plane/i18n";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
import { EIssuesStoreType } from "@plane/types";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { IssueListEmptyState } from "./issue-list-empty-state";

export const ProjectEmptyState = function ProjectEmptyState() {
  // plane imports
  const { t } = useTranslation();
  // store hooks
  const { toggleCreateIssueModal } = useCommandPalette();
  return (
    <IssueListEmptyState clearFiltersLabel={t("project_issues.empty_state.issues_empty_filter.secondary_button.text")}>
      {(canPerformEmptyStateActions) => (
        <EmptyStateDetailed
          assetKey="work-item"
          title={t("project_empty_state.work_items.title")}
          description={t("project_empty_state.work_items.description")}
          actions={[
            {
              label: t("project_empty_state.work_items.cta_primary"),
              onClick: () => {
                toggleCreateIssueModal(true, EIssuesStoreType.PROJECT);
              },
              disabled: !canPerformEmptyStateActions,
              variant: "primary",
            },
          ]}
        />
      )}
    </IssueListEmptyState>
  );
};
