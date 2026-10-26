/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCurrentWorkspace } from "@/hooks/use-workspaces";
import { useParams } from "@tanstack/react-router";

// plane imports

import { useTranslation } from "@plane/i18n";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
import { EIssuesStoreType, EUserWorkspaceRoles } from "@plane/types";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkspacePermissions } from "@/hooks/use-permissions";

export const GlobalViewEmptyState = function GlobalViewEmptyState() {
  // plane imports
  const { t } = useTranslation();
  // store hooks
  const { data: catalogueWorkspace } = useCurrentWorkspace();
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const workspaceProjectIds = projectRows
    ?.filter((project) => project.workspace === catalogueWorkspace?.id && !project.archived_at)
    .map((project) => project.id);
  const { toggleCreateIssueModal, toggleCreateProjectModal } = useCommandPalette();
  const { workspaceSlug } = useParams({ strict: false });
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);
  // derived values
  const hasMemberLevelPermission = canWorkspace([EUserWorkspaceRoles.ADMIN, EUserWorkspaceRoles.MEMBER]);

  if (workspaceProjectIds?.length === 0) {
    return (
      <EmptyStateDetailed
        title={t("workspace_projects.empty_state.no_projects.title")}
        description={t("workspace_projects.empty_state.no_projects.description")}
        assetKey="project"
        assetClassName="size-40"
        actions={[
          {
            label: t("workspace_projects.empty_state.no_projects.primary_button.text"),
            onClick: () => {
              toggleCreateProjectModal(true);
            },
            disabled: !hasMemberLevelPermission,
            variant: "primary",
          },
        ]}
      />
    );
  }

  return (
    <EmptyStateDetailed
      title={t(`workspace_empty_state.views.title`)}
      description={t(`workspace_empty_state.views.description`)}
      assetKey="project"
      assetClassName="size-40"
      actions={[
        {
          label: t(`workspace_empty_state.views.cta_primary`),
          onClick: () => {
            toggleCreateIssueModal(true, EIssuesStoreType.PROJECT);
          },
          disabled: !hasMemberLevelPermission,
          variant: "primary",
        },
      ]}
    />
  );
};
