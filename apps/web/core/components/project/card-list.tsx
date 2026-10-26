/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Button } from "@plane/ui";
import type { TProject } from "@plane/types";
import { useParams } from "@tanstack/react-router";
// plane imports
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
import { ContentWrapper } from "@plane/ui";
// components
import { ProjectsLoader } from "@/components/ui/loader/projects-loader";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
// local imports
import { ProjectCard } from "./card";

type Props = {
  projects: TProject[];
  workspaceProjectCount: number;
  archivedOnly: boolean;
  hasFilters: boolean;
  isPending: boolean;
  isError: boolean;
  accessDenied: boolean;
  retry: () => Promise<unknown>;
};

export function ProjectCardList({
  projects,
  workspaceProjectCount,
  archivedOnly,
  hasFilters,
  isPending,
  isError,
  accessDenied,
  retry,
}: Props) {
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const { toggleCreateProjectModal } = useCommandPalette();
  const { workspaceSlug } = useParams({ strict: false });
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);

  // permissions
  const canPerformEmptyStateActions = canWorkspace([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  if (isPending) return <ProjectsLoader />;

  if (accessDenied)
    return (
      <div className="flex items-center justify-center p-8" role="alert">
        You do not have access to this workspace.
      </div>
    );

  if (isError)
    return (
      <div className="flex items-center justify-center gap-3 p-8" role="alert">
        <span>{t("common.error.generic")}</span>
        <Button variant="neutral-primary" size="sm" onClick={() => void retry()}>
          {t("common.retry")}
        </Button>
      </div>
    );

  if (workspaceProjectCount === 0 && !archivedOnly)
    return (
      <EmptyStateDetailed
        title={t("workspace_projects.empty_state.general.title")}
        description={t("workspace_projects.empty_state.general.description")}
        assetKey="project"
        assetClassName="size-40"
        actions={[
          {
            label: t("workspace_projects.empty_state.general.primary_button.text"),
            onClick: () => {
              toggleCreateProjectModal(true);
            },
            disabled: !canPerformEmptyStateActions,
            variant: "primary",
          },
        ]}
      />
    );

  if (projects.length === 0)
    return (
      <EmptyStateDetailed
        title={
          archivedOnly && !hasFilters
            ? t("workspace_empty_state.projects_archived.title")
            : t("common_empty_state.search.title")
        }
        description={
          archivedOnly && !hasFilters
            ? t("workspace_empty_state.projects_archived.description")
            : t("common_empty_state.search.description")
        }
        assetKey={archivedOnly && !hasFilters ? "archived-work-item" : "search"}
        assetClassName="size-40"
      />
    );

  return (
    <ContentWrapper>
      <div className="grid grid-cols-1 gap-8 md:grid-cols-2 lg:grid-cols-3">
        {projects.map((project) => (
          <ProjectCard key={project.id} project={project} />
        ))}
      </div>
    </ContentWrapper>
  );
}
