/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useLocation } from "@tanstack/react-router";
import { useCallback, useEffect } from "react";

// plane imports
import { useTranslation } from "@plane/i18n";
import type { TProjectAppliedDisplayFilterKeys, TProjectFilters } from "@plane/types";
import { calculateTotalFilters } from "@plane/utils/filter";
import { orderProjects, shouldFilterProject } from "@plane/utils/project";
// components
import { PageHead } from "@/components/core/page-title";
// hooks
import { useCurrentWorkspace } from "@/hooks/use-workspaces";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useProjectListFilters } from "@/hooks/use-project-list-filters";
// local imports
import { ProjectAppliedFiltersList } from "./applied-filters";
import { ProjectCardList } from "./card-list";

export function ProjectRoot() {
  const { data: currentWorkspace } = useCurrentWorkspace();
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const pathname = useLocation({ select: (location) => location.pathname });
  const { t } = useTranslation();
  const catalogue = useWorkspaceProjectDetails(workspaceSlug);
  const {
    displayFilters,
    filters: currentWorkspaceFilters,
    searchQuery,
    appliedDisplayFilters: currentWorkspaceAppliedDisplayFilters,
    clearAllFilters,
    clearAllAppliedDisplayFilters,
    updateFilters,
    updateDisplayFilters,
  } = useProjectListFilters();
  const workspaceProjects = catalogue.projects?.filter((project) => project.workspace === currentWorkspace?.id) ?? [];
  const filteredProjects = orderProjects(
    workspaceProjects.filter(
      (project) =>
        shouldFilterProject(project, displayFilters, currentWorkspaceFilters) &&
        (project.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          project.identifier.toLowerCase().includes(searchQuery.toLowerCase()))
    ),
    displayFilters.order_by
  );
  // derived values
  const pageTitle = currentWorkspace?.name
    ? `${currentWorkspace.name} - ${t("workspace_projects.label", { count: 2 })}`
    : undefined;

  const isArchived = pathname.includes("/archives");

  const allowedDisplayFilters =
    currentWorkspaceAppliedDisplayFilters?.filter((filter) => filter !== "archived_projects") ?? [];

  const handleRemoveFilter = useCallback(
    (key: keyof TProjectFilters, value: string | null) => {
      let newValues = currentWorkspaceFilters?.[key] ?? [];

      if (!value) newValues = [];
      else newValues = newValues.filter((val) => val !== value);

      updateFilters({ [key]: newValues });
    },
    [currentWorkspaceFilters, updateFilters]
  );

  const handleRemoveDisplayFilter = useCallback(
    (key: TProjectAppliedDisplayFilterKeys) => {
      updateDisplayFilters({ [key]: false });
    },
    [updateDisplayFilters]
  );

  const handleClearAllFilters = useCallback(() => {
    clearAllFilters();
    clearAllAppliedDisplayFilters();
    if (isArchived) updateDisplayFilters({ archived_projects: true });
  }, [clearAllFilters, clearAllAppliedDisplayFilters, isArchived, updateDisplayFilters]);

  useEffect(() => {
    updateDisplayFilters({ archived_projects: isArchived });
  }, [isArchived, updateDisplayFilters]);

  return (
    <>
      <PageHead title={pageTitle} />
      <div className="flex h-full w-full flex-col">
        {(calculateTotalFilters(currentWorkspaceFilters ?? {}) !== 0 || allowedDisplayFilters.length > 0) && (
          <ProjectAppliedFiltersList
            appliedFilters={currentWorkspaceFilters ?? {}}
            appliedDisplayFilters={allowedDisplayFilters}
            handleClearAllFilters={handleClearAllFilters}
            handleRemoveFilter={handleRemoveFilter}
            handleRemoveDisplayFilter={handleRemoveDisplayFilter}
            filteredProjects={filteredProjects.length}
            totalProjects={workspaceProjects.length}
            alwaysAllowEditing
          />
        )}
        <ProjectCardList
          projects={filteredProjects}
          workspaceProjectCount={workspaceProjects.filter((project) => !project.archived_at).length}
          archivedOnly={Boolean(displayFilters.archived_projects)}
          hasFilters={calculateTotalFilters(currentWorkspaceFilters ?? {}) > 0}
          isPending={catalogue.isPending}
          isError={catalogue.isError}
          accessDenied={catalogue.accessDenied}
          retry={catalogue.retryAccess}
        />
      </div>
    </>
  );
}
