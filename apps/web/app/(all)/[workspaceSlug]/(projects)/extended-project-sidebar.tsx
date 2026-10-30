/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useCallback, useRef, useState } from "react";

// plane imports
import { EUserPermissions, PROJECT_TRACKER_ELEMENTS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { EmptyStateCompact } from "@plane/propel/empty-state";
import { PlusIcon, SearchIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Tooltip } from "@plane/propel/tooltip";
import { copyUrlToClipboard } from "@plane/utils/string";
// components
import { CreateProjectModal } from "@/components/project/create-project-modal";
import { SidebarProjectsListItem } from "@/components/workspace/sidebar/projects-list-item";
// hooks
import { useSidebarActions, useExtendedProjectSidebarOpen } from "@/providers/sidebar-state";
import { useWorkspaceProjects } from "@/hooks/use-projects";
import { useJoinedProjectIds } from "@/hooks/use-joined-project-ids";
import { useWorkspaceProjectReordering } from "@/hooks/use-workspace-project-reordering";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { ExtendedSidebarWrapper } from "./extended-sidebar-wrapper";

export const ExtendedProjectSidebar = function ExtendedProjectSidebar() {
  // refs
  const extendedProjectSidebarRef = useRef<HTMLDivElement | null>(null);
  const [searchQuery, setSearchQuery] = useState<string>("");
  // states
  const [isProjectModalOpen, setIsProjectModalOpen] = useState(false);
  // routers
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug/_projects" });
  // store hooks
  const { t } = useTranslation();
  const isExtendedProjectSidebarOpened = useExtendedProjectSidebarOpen();
  const { toggleExtendedProjectSidebar } = useSidebarActions();
  const projectCatalogue = useWorkspaceProjects(workspaceSlug);
  const projectRows = projectCatalogue.projects;
  const joinedProjects = useJoinedProjectIds(workspaceSlug);
  const handleOnProjectDrop = useWorkspaceProjectReordering({ workspaceSlug, joinedProjectIds: joinedProjects });
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);

  // filter projects based on search query
  const filteredProjects = joinedProjects.filter((projectId) => {
    const project = projectRows?.find((candidate) => candidate.id === projectId);
    if (!project) return false;
    return project.name.toLowerCase().includes(searchQuery.toLowerCase()) || project.identifier.includes(searchQuery);
  });

  // auth
  const isAuthorizedUser = canWorkspace([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const handleClose = useCallback(() => toggleExtendedProjectSidebar(false), [toggleExtendedProjectSidebar]);

  const handleCopyText = async (projectId: string) => {
    try {
      await copyUrlToClipboard(`${workspaceSlug}/projects/${projectId}/issues`);
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("link_copied"),
        message: t("project_link_copied_to_clipboard"),
      });
    } catch {
      setToast({ type: TOAST_TYPE.ERROR, title: "Copy failed", message: "The project link could not be copied." });
    }
  };
  return (
    <>
      {workspaceSlug && (
        <CreateProjectModal
          isOpen={isProjectModalOpen}
          onClose={() => setIsProjectModalOpen(false)}
          setToFavorite={false}
          workspaceSlug={workspaceSlug}
        />
      )}
      <ExtendedSidebarWrapper
        isExtendedSidebarOpened={!!isExtendedProjectSidebarOpened}
        extendedSidebarRef={extendedProjectSidebarRef}
        handleClose={handleClose}
        excludedElementId="extended-project-sidebar-toggle"
        className="px-0"
      >
        <div className="sticky top-4 flex w-full flex-col gap-1 px-4">
          <div className="flex items-center justify-between">
            <span className="py-1.5 text-13 font-semibold text-tertiary">Projects</span>
            {isAuthorizedUser && (
              <Tooltip tooltipHeading={t("create_project")} tooltipContent="">
                <button
                  type="button"
                  data-ph-element={PROJECT_TRACKER_ELEMENTS.EXTENDED_SIDEBAR_ADD_BUTTON}
                  className="flex-shrink-0 rounded-sm p-0.5 text-tertiary transition-colors hover:bg-layer-1 hover:text-secondary"
                  onClick={() => {
                    setIsProjectModalOpen(true);
                  }}
                >
                  <PlusIcon className="size-3" />
                </button>
              </Tooltip>
            )}
          </div>
          <div className="ml-auto flex w-full items-center gap-1.5 rounded-md border border-subtle bg-surface-1 px-2.5 py-1">
            <SearchIcon className="h-3.5 w-3.5 text-placeholder" />
            <input
              className="w-full max-w-[234px] border-none bg-transparent text-13 outline-none placeholder:text-placeholder"
              placeholder={t("search")}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>
        </div>
        {filteredProjects.length === 0 ? (
          <div className="mt-4 flex flex-col items-center p-10">
            <EmptyStateCompact
              title={t("common_empty_state.search.title")}
              description={t("common_empty_state.search.description")}
              assetKey="search"
              assetClassName="size-20"
              align="center"
            />
          </div>
        ) : (
          <div className="vertical-scrollbar mt-4 flex scrollbar-sm flex-grow flex-col gap-0.5 overflow-x-hidden overflow-y-auto pr-2 pl-9">
            {filteredProjects.map((projectId, index) => (
              <SidebarProjectsListItem
                key={projectId}
                projectId={projectId}
                handleCopyText={() => void handleCopyText(projectId)}
                projectListType={"JOINED"}
                disableDrag={false}
                disableDrop={false}
                isLastChild={index === filteredProjects.length - 1}
                handleOnProjectDrop={handleOnProjectDrop}
                renderInExtendedSidebar
              />
            ))}
          </div>
        )}
      </ExtendedSidebarWrapper>
    </>
  );
};
