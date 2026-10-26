/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";

// plane imports

import { useTranslation } from "@plane/i18n";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
// hooks
import { IssueListEmptyState } from "./issue-list-empty-state";

export const ProjectArchivedEmptyState = function ProjectArchivedEmptyState() {
  // router
  const router = useRouter();
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  return (
    <IssueListEmptyState clearFiltersLabel="Clear filters">
      {(canPerformEmptyStateActions) => (
        <EmptyStateDetailed
          assetKey="archived-work-item"
          title={t("workspace_empty_state.archive_work_items.title")}
          description={t("workspace_empty_state.archive_work_items.description")}
          actions={[
            {
              label: t("workspace_empty_state.archive_work_items.cta_primary"),
              onClick: () =>
                void router.navigate({ href: `/${workspaceSlug}/settings/projects/${projectId}/automations` }),
              disabled: !canPerformEmptyStateActions,
              variant: "primary",
            },
          ]}
        />
      )}
    </IssueListEmptyState>
  );
};
