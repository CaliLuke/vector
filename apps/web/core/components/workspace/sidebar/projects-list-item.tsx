/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import { useCallback, useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { draggable, dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { pointerOutsideOfPreview } from "@atlaskit/pragmatic-drag-and-drop/element/pointer-outside-of-preview";
import { setCustomNativeDragPreview } from "@atlaskit/pragmatic-drag-and-drop/element/set-custom-native-drag-preview";
import { attachInstruction, extractInstruction } from "@atlaskit/pragmatic-drag-and-drop-hitbox/tree-item";

import { createRoot } from "react-dom/client";
import scrollIntoView from "smooth-scroll-into-view-if-needed";
import { Settings, Share2, MoreHorizontal } from "lucide-react";
import { Disclosure, Transition } from "@headlessui/react";
// plane imports
import { EUserPermissions } from "@plane/constants";
import { useOutsideClickDetector } from "@plane/hooks";
import { useTranslation } from "@plane/i18n";
import { Logo } from "@plane/propel/emoji-icon-picker";
import { LinkIcon, ArchiveIcon, ChevronRightIcon } from "@plane/propel/icons";
import { IconButton } from "@plane/propel/icon-button";
import { CustomMenu, DropIndicator, ControlLink } from "@plane/ui";
import { cn } from "@plane/utils/common";
// components
import { DEFAULT_TAB_KEY, getTabUrl } from "@/components/navigation/tab-navigation-utils";
import { useTabPreferences } from "@/components/navigation/use-tab-preferences";
import { LeaveProjectModal } from "@/components/project/leave-project-modal";
import { PublishProjectModal } from "@/components/project/publish-project/modal";
// hooks
import { useSidebarActions, useExtendedProjectSidebarOpen } from "@/providers/sidebar-state";
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useWorkspaceProjects } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useProjectNavigationPreferences } from "@/hooks/use-navigation-preferences";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { HIGHLIGHT_CLASS, highlightIssueOnDrop } from "../../issues/issue-layouts/utils";
import { ProjectNavigation } from "./project-navigation";
import { SidebarDragHandle } from "./drag-handle";
import { useNavigationItems } from "@/components/navigation/use-navigation-items";
import { LeaveProjectMenuItem } from "@/components/navigation/leave-project-menu-item";

type Props = {
  projectId: string;
  handleCopyText: () => void;
  handleOnProjectDrop?: (
    sourceId: string | undefined,
    destinationId: string | undefined,
    shouldDropAtEnd: boolean
  ) => void;
  projectListType: "JOINED" | "FAVORITES";
  disableDrag?: boolean;
  disableDrop?: boolean;
  isLastChild: boolean;
  renderInExtendedSidebar?: boolean;
};

export const SidebarProjectsListItem = function SidebarProjectsListItem(props: Props) {
  const {
    projectId,
    handleCopyText,
    disableDrag,
    disableDrop,
    isLastChild,
    handleOnProjectDrop,
    projectListType,
    renderInExtendedSidebar = false,
  } = props;
  // store hooks
  const { t } = useTranslation();
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const projectCatalogue = useWorkspaceProjects(workspaceSlug);
  const projectRows = projectCatalogue.projects;
  const { isMobile } = usePlatformOS();
  const { getIsProjectListOpen, toggleProjectListOpen } = useCommandPalette();
  const { preferences: projectPreferences } = useProjectNavigationPreferences();
  const isExtendedProjectSidebarOpened = useExtendedProjectSidebarOpen();
  const { toggleExtendedProjectSidebar, toggleAnySidebarDropdown } = useSidebarActions();

  // states
  const [leaveProjectModalOpen, setLeaveProjectModal] = useState(false);
  const [publishModalOpen, setPublishModal] = useState(false);
  const [isMenuActive, setIsMenuActive] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const isProjectListOpen = getIsProjectListOpen(projectId);
  const [instruction, setInstruction] = useState<"DRAG_OVER" | "DRAG_BELOW" | undefined>(undefined);
  // refs
  const actionSectionRef = useRef<HTMLButtonElement | null>(null);
  const projectRef = useRef<HTMLDivElement | null>(null);
  const dragHandleRef = useRef<HTMLButtonElement | null>(null);
  // router
  const { projectId: URLProjectId } = useParams({ strict: false });
  const router = useRouter();
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  // derived values
  const project = projectRows?.find((candidate) => candidate.id === projectId);

  // Get available navigation items for this project
  const navigationItems = useNavigationItems({ workspaceSlug, projectId, project });
  const availableTabKeys = navigationItems.map((item) => item.key);

  // Get preferences from hook
  const { tabPreferences } = useTabPreferences(workspaceSlug, projectId);
  const defaultTabKey = tabPreferences.defaultTab;
  // Validate that the default tab is available
  const validatedDefaultTabKey = availableTabKeys.includes(defaultTabKey) ? defaultTabKey : DEFAULT_TAB_KEY;
  const defaultTabUrl = project ? getTabUrl(workspaceSlug, project.id, validatedDefaultTabKey) : "";

  // toggle project list open
  const setIsProjectListOpen = useCallback(
    (value: boolean) => toggleProjectListOpen(projectId, value),
    [projectId, toggleProjectListOpen]
  );
  // auth
  const isAdmin = canProject([EUserPermissions.ADMIN]);
  const isAuthorized = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const handleLeaveProject = () => setLeaveProjectModal(true);

  useProjectSidebarDragAndDrop({
    projectRef,
    dragHandleRef,
    disableDrag,
    disableDrop,
    handleOnProjectDrop,
    isLastChild,
    project,
    projectId,
    projectListType,
    setInstruction,
    setIsDragging,
  });

  useEffect(() => {
    if (isMenuActive) toggleAnySidebarDropdown(true);
    else toggleAnySidebarDropdown(false);
  }, [isMenuActive, toggleAnySidebarDropdown]);

  useOutsideClickDetector(actionSectionRef, () => setIsMenuActive(false));
  useOutsideClickDetector(projectRef, () => projectRef?.current?.classList?.remove(HIGHLIGHT_CLASS));

  useEffect(() => {
    let timeoutId: ReturnType<typeof setTimeout> | undefined;

    if (URLProjectId === project?.id) {
      setIsProjectListOpen(true);
      // Scroll to active project
      if (projectRef.current) {
        timeoutId = setTimeout(() => {
          if (projectRef.current) {
            void scrollIntoView(projectRef.current, {
              behavior: "smooth",
              block: "center",
              scrollMode: "if-needed",
            });
          }
        }, 200);
      }
    }

    return () => {
      if (timeoutId) {
        clearTimeout(timeoutId);
      }
    };
  }, [URLProjectId, project?.id, setIsProjectListOpen]);

  if (!project) return null;

  const isAccordionMode = projectPreferences.navigationMode === "ACCORDION";

  const handleItemClick = () => {
    if (projectPreferences.navigationMode === "ACCORDION") {
      setIsProjectListOpen(!isProjectListOpen);
    } else {
      void router.navigate({ href: defaultTabUrl });
    }
    // close the extended sidebar if it is open
    if (isExtendedProjectSidebarOpened && !isAccordionMode) {
      toggleExtendedProjectSidebar(false);
    }
  };

  const shouldHighlightProject = URLProjectId === project.id && projectPreferences.navigationMode !== "ACCORDION";

  return (
    <>
      <PublishProjectModal isOpen={publishModalOpen} projectId={projectId} onClose={() => setPublishModal(false)} />
      <LeaveProjectModal project={project} isOpen={leaveProjectModalOpen} onClose={() => setLeaveProjectModal(false)} />
      <Disclosure key={`${project.id}_${URLProjectId}`} defaultOpen={isProjectListOpen} as="div">
        <div
          id={`sidebar-${projectId}-${projectListType}`}
          className={cn("relative", {
            "bg-layer-1 opacity-60": isDragging,
          })}
          ref={projectRef}
        >
          <DropIndicator classNames="absolute top-0" isVisible={instruction === "DRAG_OVER"} />
          <div
            className={cn(
              "group/project-item relative flex w-full items-center rounded-md px-2 py-1.5 text-primary hover:bg-layer-transparent-hover",
              {
                "bg-surface-2": isMenuActive,
                "bg-layer-transparent-active": shouldHighlightProject,
              }
            )}
            id={project.id}
          >
            {!disableDrag && (
              <SidebarDragHandle
                isMobile={isMobile}
                tooltipContent={
                  project.sort_order === null ? t("join_the_project_to_rearrange") : t("drag_to_rearrange")
                }
                position="top-end"
                disabled={isDragging}
                handleClassName={cn(
                  "absolute top-1/2 -left-3 hidden -translate-y-1/2 cursor-grab items-center justify-center rounded-sm text-placeholder group-hover/project-item:flex",
                  {
                    "cursor-not-allowed opacity-60": project.sort_order === null,
                    "cursor-grabbing": isDragging,
                    flex: isMenuActive || renderInExtendedSidebar,
                  }
                )}
                ref={dragHandleRef}
              />
            )}
            <>
              <SidebarProjectLink
                defaultTabUrl={defaultTabUrl}
                isAccordionMode={isAccordionMode}
                isProjectListOpen={isProjectListOpen}
                onClick={handleItemClick}
                project={project}
                t={t}
              />
              <div className="flex items-center gap-1">
                <SidebarProjectQuickActions
                  actionSectionRef={actionSectionRef}
                  isAdmin={isAdmin}
                  isAuthorized={isAuthorized}
                  isMenuActive={isMenuActive}
                  onCopy={handleCopyText}
                  onLeave={handleLeaveProject}
                  onMenuClose={() => setIsMenuActive(false)}
                  onPublish={() => setPublishModal(true)}
                  onToggleMenu={() => setIsMenuActive(!isMenuActive)}
                  onNavigateArchives={() => {
                    void router.navigate({ href: `/${workspaceSlug}/projects/${project.id}/archives/issues` });
                  }}
                  onNavigateSettings={() => {
                    void router.navigate({ href: `/${workspaceSlug}/settings/projects/${project.id}` });
                  }}
                  t={t}
                />
                {isAccordionMode && (
                  <IconButton
                    variant="ghost"
                    size="sm"
                    icon={ChevronRightIcon}
                    onClick={() => setIsProjectListOpen(!isProjectListOpen)}
                    className={cn("hidden text-placeholder group-hover/project-item:inline-flex", {
                      "inline-flex": isMenuActive,
                    })}
                    iconClassName={cn("transition-transform", {
                      "rotate-90": isProjectListOpen,
                    })}
                    aria-label={t(
                      isProjectListOpen
                        ? "aria_labels.projects_sidebar.close_project_menu"
                        : "aria_labels.projects_sidebar.open_project_menu"
                    )}
                  />
                )}
              </div>
            </>
          </div>
          <SidebarProjectNavigation
            isAccordionMode={isAccordionMode}
            isProjectListOpen={isProjectListOpen}
            projectId={projectId}
            workspaceSlug={workspaceSlug}
          />
          {isLastChild && <DropIndicator isVisible={instruction === "DRAG_BELOW"} />}
        </div>
      </Disclosure>
    </>
  );
};

type TSidebarProjectQuickActionsProps = {
  actionSectionRef: React.RefObject<HTMLButtonElement | null>;
  isAdmin: boolean;
  isAuthorized: boolean;
  isMenuActive: boolean;
  onCopy: () => void;
  onLeave: () => void;
  onMenuClose: () => void;
  onNavigateArchives: () => void;
  onNavigateSettings: () => void;
  onPublish: () => void;
  onToggleMenu: () => void;
  t: ReturnType<typeof useTranslation>["t"];
};

function SidebarProjectQuickActions(props: TSidebarProjectQuickActionsProps) {
  const {
    actionSectionRef,
    isAdmin,
    isAuthorized,
    isMenuActive,
    onCopy,
    onLeave,
    onMenuClose,
    onNavigateArchives,
    onNavigateSettings,
    onPublish,
    onToggleMenu,
    t,
  } = props;
  return (
    <CustomMenu
      customButtonAsChild
      customButton={
        <IconButton
          ref={actionSectionRef}
          variant="ghost"
          size="sm"
          icon={MoreHorizontal}
          onClick={onToggleMenu}
          className="text-placeholder"
        />
      }
      className={cn(
        "pointer-events-none flex-shrink-0 opacity-0 group-hover/project-item:pointer-events-auto group-hover/project-item:opacity-100",
        { "pointer-events-auto opacity-100": isMenuActive }
      )}
      customButtonClassName="grid place-items-center"
      placement="bottom-start"
      ariaLabel={t("aria_labels.projects_sidebar.toggle_quick_actions_menu")}
      useCaptureForOutsideClick
      closeOnSelect
      onMenuClose={onMenuClose}
    >
      {isAdmin && (
        <CustomMenu.MenuItem onClick={onPublish}>
          <div className="relative flex flex-shrink-0 items-center justify-start gap-2">
            <div className="flex h-4 w-4 cursor-pointer items-center justify-center rounded-sm text-secondary transition-all duration-300 hover:bg-layer-1">
              <Share2 className="h-3.5 w-3.5 stroke-[1.5]" />
            </div>
            <div>{t("publish_project")}</div>
          </div>
        </CustomMenu.MenuItem>
      )}
      <CustomMenu.MenuItem onClick={onCopy}>
        <span className="flex items-center justify-start gap-2">
          <LinkIcon className="h-3.5 w-3.5 stroke-[1.5]" />
          <span>{t("copy_link")}</span>
        </span>
      </CustomMenu.MenuItem>
      {isAuthorized && (
        <CustomMenu.MenuItem onClick={onNavigateArchives}>
          <div className="flex cursor-pointer items-center justify-start gap-2">
            <ArchiveIcon className="h-3.5 w-3.5 stroke-[1.5]" />
            <span>{t("archives")}</span>
          </div>
        </CustomMenu.MenuItem>
      )}
      <CustomMenu.MenuItem onClick={onNavigateSettings}>
        <div className="flex cursor-pointer items-center justify-start gap-2">
          <Settings className="h-3.5 w-3.5 stroke-[1.5]" />
          <span>{t("settings")}</span>
        </div>
      </CustomMenu.MenuItem>
      {!isAuthorized && <LeaveProjectMenuItem onLeave={onLeave} />}
    </CustomMenu>
  );
}

type TSidebarProject = NonNullable<ReturnType<typeof useWorkspaceProjects>["projects"]>[number];
type TSidebarInstruction = "DRAG_OVER" | "DRAG_BELOW" | undefined;

function SidebarProjectLink(props: {
  defaultTabUrl: string;
  isAccordionMode: boolean;
  isProjectListOpen: boolean;
  onClick: () => void;
  project: TSidebarProject;
  t: ReturnType<typeof useTranslation>["t"];
}) {
  const content = (
    <>
      <div className="grid size-4 flex-shrink-0 place-items-center">
        <Logo logo={props.project.logo_props} size={16} />
      </div>
      <p className="truncate text-13 font-medium text-secondary">{props.project.name}</p>
    </>
  );
  return (
    <ControlLink href={props.defaultTabUrl} className="flex flex-grow truncate" onClick={props.onClick}>
      {props.isAccordionMode ? (
        <Disclosure.Button
          as="button"
          type="button"
          className="flex w-full flex-grow items-center gap-1.5 text-left select-none"
          aria-label={
            props.isProjectListOpen
              ? props.t("aria_labels.projects_sidebar.close_project_menu")
              : props.t("aria_labels.projects_sidebar.open_project_menu")
          }
        >
          {content}
        </Disclosure.Button>
      ) : (
        <div className="flex w-full flex-grow items-center gap-1.5 text-left select-none">{content}</div>
      )}
    </ControlLink>
  );
}

function SidebarProjectNavigation(props: {
  isAccordionMode: boolean;
  isProjectListOpen: boolean;
  projectId: string;
  workspaceSlug: string | string[] | undefined;
}) {
  if (!props.isAccordionMode) return null;
  return (
    <Transition
      show={props.isProjectListOpen}
      enter="transition duration-100 ease-out"
      enterFrom="transform scale-95 opacity-0"
      enterTo="transform scale-100 opacity-100"
      leave="transition duration-75 ease-out"
      leaveFrom="transform scale-100 opacity-100"
      leaveTo="transform scale-95 opacity-0"
    >
      <Disclosure.Panel as="div" className="relative mt-1 mb-1.5 flex flex-col gap-0.5 pl-6" static>
        <div className="absolute top-0 bottom-1 left-[15px] w-[1px] bg-layer-3" />
        <ProjectNavigation workspaceSlug={props.workspaceSlug?.toString() ?? ""} projectId={props.projectId} />
      </Disclosure.Panel>
    </Transition>
  );
}

type TProjectSidebarDragAndDropProps = {
  projectRef: React.RefObject<HTMLDivElement | null>;
  dragHandleRef: React.RefObject<HTMLButtonElement | null>;
  disableDrag?: boolean;
  disableDrop?: boolean;
  handleOnProjectDrop?: Props["handleOnProjectDrop"];
  isLastChild: boolean;
  project: TSidebarProject | undefined;
  projectId: string;
  projectListType: Props["projectListType"];
  setInstruction: React.Dispatch<React.SetStateAction<TSidebarInstruction>>;
  setIsDragging: React.Dispatch<React.SetStateAction<boolean>>;
};

function useProjectSidebarDragAndDrop(props: TProjectSidebarDragAndDropProps) {
  const {
    projectRef,
    dragHandleRef,
    disableDrag,
    disableDrop,
    handleOnProjectDrop,
    isLastChild,
    project,
    projectId,
    projectListType,
    setInstruction,
    setIsDragging,
  } = props;

  useEffect(() => {
    const element = projectRef.current;
    const dragHandleElement = dragHandleRef.current;
    if (!element) return undefined;

    return combine(
      draggable({
        element,
        canDrag: () => !disableDrag,
        dragHandle: dragHandleElement ?? undefined,
        getInitialData: () => ({ id: projectId, dragInstanceId: "PROJECTS" }),
        onDragStart: () => setIsDragging(true),
        onDrop: () => setIsDragging(false),
        onGenerateDragPreview: ({ nativeSetDragImage }) => {
          setCustomNativeDragPreview({
            getOffset: pointerOutsideOfPreview({ x: "0px", y: "0px" }),
            render: ({ container }) => {
              const root = createRoot(container);
              root.render(
                <div className="flex items-center rounded-sm bg-surface-1 p-1 pr-2 text-13">
                  <div className="grid size-4 flex-shrink-0 place-items-center">
                    {project && <Logo logo={project.logo_props} />}
                  </div>
                  <p className="truncate text-secondary">{project?.name}</p>
                </div>
              );
              return () => root.unmount();
            },
            nativeSetDragImage,
          });
        },
      }),
      dropTargetForElements({
        element,
        canDrop: ({ source }) =>
          !disableDrop && source?.data?.id !== projectId && source?.data?.dragInstanceId === "PROJECTS",
        // oxlint-disable-next-line no-shadow
        getData: ({ input, element }) =>
          attachInstruction(
            { id: projectId },
            {
              input,
              element,
              currentLevel: 0,
              indentPerLevel: 0,
              mode: isLastChild ? "last-in-group" : "standard",
            }
          ),
        onDrag: ({ self }) => setInstruction(getSidebarDropInstruction(self?.data, isLastChild)),
        onDragLeave: () => setInstruction(undefined),
        onDrop: ({ self, source }) => {
          setInstruction(undefined);
          const instruction = getSidebarDropInstruction(self?.data, isLastChild);
          if (!instruction) return;
          const sourceId = getSidebarDataId(source?.data);
          const destinationId = getSidebarDataId(self?.data);
          handleOnProjectDrop?.(sourceId, destinationId, instruction === "DRAG_BELOW");
          highlightIssueOnDrop(`sidebar-${sourceId}-${projectListType}`);
        },
      })
    );
  }, [
    disableDrag,
    disableDrop,
    dragHandleRef,
    handleOnProjectDrop,
    isLastChild,
    project,
    projectId,
    projectListType,
    projectRef,
    setInstruction,
    setIsDragging,
  ]);
}

function getSidebarDataId(data: Record<string, unknown> | undefined): string | undefined {
  const id = data?.id;
  return typeof id === "string" ? id : undefined;
}

function getSidebarDropInstruction(
  data: Record<string, unknown> | undefined,
  isLastChild: boolean
): TSidebarInstruction {
  if (!data) return undefined;
  const instruction = extractInstruction(data)?.type;
  if (!instruction) return undefined;
  return instruction === "reorder-below" && isLastChild ? "DRAG_BELOW" : "DRAG_OVER";
}
