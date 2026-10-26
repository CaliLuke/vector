/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useLocation } from "@tanstack/react-router";
import { useState, useRef, useEffect } from "react";
import { autoScrollForElements } from "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element";

import { Ellipsis } from "lucide-react";
import { Disclosure, Transition } from "@headlessui/react";
// plane imports
import { EUserPermissions, PROJECT_TRACKER_ELEMENTS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { PlusIcon, ChevronRightIcon } from "@plane/propel/icons";
import { IconButton } from "@plane/propel/icon-button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Tooltip } from "@plane/propel/tooltip";
import { Loader } from "@plane/ui";
import { copyUrlToClipboard } from "@plane/utils/string";
import { cn } from "@plane/utils/common";
// components
import { CreateProjectModal } from "@/components/project/create-project-modal";
import { SidebarNavItem } from "@/components/sidebar/sidebar-navigation";
// hooks
import { useSidebarActions, useExtendedProjectSidebarOpen } from "@/providers/sidebar-state";
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useWorkspaceProjects } from "@/hooks/use-projects";
import { useWorkspaceProjectReordering } from "@/hooks/use-workspace-project-reordering";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useProjectNavigationPreferences } from "@/hooks/use-navigation-preferences";
import { useJoinedProjectIds } from "@/hooks/use-joined-project-ids";
import { findSidebarScrollViewport } from "./projects-list.utils";
// plane web imports
// local imports
import { SidebarProjectsListItem } from "./projects-list-item";

function getProjectListDisplay(projectIds: string[], showLimitedProjects: boolean, limitedProjectsCount: number) {
  return {
    displayedProjects: showLimitedProjects ? projectIds.slice(0, limitedProjectsCount) : projectIds,
    hasMoreProjects: showLimitedProjects && projectIds.length > limitedProjectsCount,
  };
}

export const SidebarProjectsList = function SidebarProjectsList() {
  // states
  const [projectListDisclosure, setProjectListDisclosure] = useState<{ pathname: string | undefined; isOpen: boolean }>(
    { pathname: undefined, isOpen: true }
  );
  const [isProjectModalOpen, setIsProjectModalOpen] = useState(false);
  const [isScrolled, setIsScrolled] = useState(false); // scroll animation state
  // refs
  const containerRef = useRef<HTMLDivElement | null>(null);
  // store hooks
  const { t } = useTranslation();
  const { toggleCreateProjectModal } = useCommandPalette();
  const { preferences: projectPreferences } = useProjectNavigationPreferences();
  const isExtendedProjectSidebarOpened = useExtendedProjectSidebarOpen();
  const { toggleExtendedProjectSidebar } = useSidebarActions();

  // router params
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const projectCatalogue = useWorkspaceProjects(workspaceSlug);
  const joinedProjects = useJoinedProjectIds(workspaceSlug);
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);
  const handleOnProjectDrop = useWorkspaceProjectReordering({
    workspaceSlug,
    joinedProjectIds: joinedProjects,
  });
  const pathname = useLocation({ select: (location) => location.pathname });
  const isAllProjectsListOpen =
    projectListDisclosure.pathname === pathname
      ? projectListDisclosure.isOpen
      : pathname.includes("projects") || projectListDisclosure.isOpen;

  // auth
  const isAuthorizedUser = canWorkspace([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  // Compute limited projects for main sidebar
  const { displayedProjects, hasMoreProjects } = getProjectListDisplay(
    joinedProjects,
    projectPreferences.showLimitedProjects,
    projectPreferences.limitedProjectsCount
  );

  const handleCopyText = (projectId: string) => {
    copyUrlToClipboard(`${workspaceSlug}/projects/${projectId}/issues`).then(() => {
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("link_copied"),
        message: t("project_link_copied_to_clipboard"),
      });
      return undefined;
    });
  };

  /**
   * Implementing scroll animation styles based on the scroll length of the container
   */
  useEffect(() => {
    const handleScroll = () => {
      if (containerRef.current) {
        const scrollTop = containerRef.current.scrollTop;
        setIsScrolled(scrollTop > 0);
      }
    };
    const currentContainerRef = containerRef.current;
    if (currentContainerRef) {
      currentContainerRef.addEventListener("scroll", handleScroll);
    }
    return () => {
      if (currentContainerRef) {
        currentContainerRef.removeEventListener("scroll", handleScroll);
      }
    };
  }, [containerRef]);

  useEffect(() => {
    const element = findSidebarScrollViewport(containerRef.current);

    if (!element) return;

    return autoScrollForElements({
      element,
      canScroll: ({ source }) => source?.data?.dragInstanceId === "PROJECTS",
      getAllowedAxis: () => "vertical",
    });
  }, [containerRef]);

  const toggleListDisclosure = (isOpen: boolean) => {
    setProjectListDisclosure({ pathname, isOpen });
    localStorage.setItem("isAllProjectsListOpen", isOpen.toString());
  };
  useEffect(() => {
    if (pathname.includes("projects")) localStorage.setItem("isAllProjectsListOpen", "true");
  }, [pathname]);
  return (
    <>
      {workspaceSlug && (
        <CreateProjectModal
          isOpen={isProjectModalOpen}
          onClose={() => setIsProjectModalOpen(false)}
          setToFavorite={false}
          workspaceSlug={workspaceSlug.toString()}
        />
      )}
      <div
        ref={containerRef}
        className={cn({
          "border-t border-strong": isScrolled,
        })}
      >
        <>
          <Disclosure as="div" className="flex flex-col" defaultOpen={isAllProjectsListOpen}>
            <div className="group flex w-full items-center justify-between rounded-sm px-2 py-1.5 text-placeholder hover:bg-layer-transparent-hover">
              <Disclosure.Button
                as="button"
                type="button"
                className="flex w-full items-center gap-1 text-left text-13 font-semibold whitespace-nowrap text-placeholder"
                onClick={() => toggleListDisclosure(!isAllProjectsListOpen)}
                aria-label={t(
                  isAllProjectsListOpen
                    ? "aria_labels.projects_sidebar.close_projects_menu"
                    : "aria_labels.projects_sidebar.open_projects_menu"
                )}
              >
                <span className="text-13 font-semibold">{t("projects")}</span>
              </Disclosure.Button>
              <div className="flex items-center gap-1">
                {isAuthorizedUser && (
                  <Tooltip tooltipHeading={t("create_project")} tooltipContent="">
                    <IconButton
                      variant="ghost"
                      size="sm"
                      icon={PlusIcon}
                      onClick={() => {
                        setIsProjectModalOpen(true);
                      }}
                      data-ph-element={PROJECT_TRACKER_ELEMENTS.SIDEBAR_CREATE_PROJECT_TOOLTIP}
                      className="hidden text-placeholder group-hover:inline-flex"
                      aria-label={t("aria_labels.projects_sidebar.create_new_project")}
                    />
                  </Tooltip>
                )}
                <IconButton
                  variant="ghost"
                  size="sm"
                  icon={ChevronRightIcon}
                  onClick={() => toggleListDisclosure(!isAllProjectsListOpen)}
                  className="text-placeholder"
                  iconClassName={cn("transition-transform", {
                    "rotate-90": isAllProjectsListOpen,
                  })}
                  aria-label={t(
                    isAllProjectsListOpen
                      ? "aria_labels.projects_sidebar.close_projects_menu"
                      : "aria_labels.projects_sidebar.open_projects_menu"
                  )}
                />
              </div>
            </div>
            <Transition
              as="div"
              className="flex flex-col"
              show={isAllProjectsListOpen}
              enter="transition duration-100 ease-out"
              enterFrom="transform scale-95 opacity-0"
              enterTo="transform scale-100 opacity-100"
              leave="transition duration-75 ease-out"
              leaveFrom="transform scale-100 opacity-100"
              leaveTo="transform scale-95 opacity-0"
            >
              {projectCatalogue.isPending && (
                <Loader className="w-full space-y-1.5">
                  {[1, 2, 3, 4].map((placeholder) => (
                    <Loader.Item key={placeholder} height="28px" />
                  ))}
                </Loader>
              )}
              {projectCatalogue.isError && (
                <div className="text-error px-2 py-2 text-11" role="alert">
                  <span>{t("common.error.generic")}</span>
                  <button type="button" className="ml-2 underline" onClick={() => void projectCatalogue.retryAccess()}>
                    {t("common.retry")}
                  </button>
                </div>
              )}
              {isAllProjectsListOpen && (
                <Disclosure.Panel as="div" className="flex flex-col gap-0.5" static>
                  <>
                    {displayedProjects.map((projectId, index) => (
                      <SidebarProjectsListItem
                        key={projectId}
                        projectId={projectId}
                        handleCopyText={() => handleCopyText(projectId)}
                        projectListType={"JOINED"}
                        disableDrag={false}
                        disableDrop={false}
                        isLastChild={index === displayedProjects.length - 1}
                        handleOnProjectDrop={handleOnProjectDrop}
                      />
                    ))}
                    {hasMoreProjects && (
                      <SidebarNavItem>
                        <button
                          type="button"
                          onClick={() => toggleExtendedProjectSidebar()}
                          className="flex flex-grow items-center gap-1.5 text-13 font-medium text-tertiary"
                          id="extended-project-sidebar-toggle"
                          aria-label={t(
                            isExtendedProjectSidebarOpened
                              ? "aria_labels.app_sidebar.close_extended_sidebar"
                              : "aria_labels.app_sidebar.open_extended_sidebar"
                          )}
                        >
                          <Ellipsis className="size-4 flex-shrink-0" />
                          <span>{isExtendedProjectSidebarOpened ? "Hide" : "More"}</span>
                        </button>
                      </SidebarNavItem>
                    )}
                  </>
                </Disclosure.Panel>
              )}
            </Transition>
          </Disclosure>
        </>

        {isAuthorizedUser && joinedProjects.length === 0 && (
          <button
            type="button"
            data-ph-element={PROJECT_TRACKER_ELEMENTS.SIDEBAR_CREATE_PROJECT_BUTTON}
            className="flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-13 leading-5 font-medium text-secondary hover:bg-surface-2"
            onClick={() => {
              toggleCreateProjectModal(true);
            }}
          >
            {t("add_project")}
          </button>
        )}
      </div>
    </>
  );
};
