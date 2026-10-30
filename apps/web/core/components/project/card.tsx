/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link, useParams, useRouter } from "@tanstack/react-router";
import React, { useRef, useState } from "react";

import { ArchiveRestoreIcon, Settings, UserPlus } from "lucide-react";
// plane imports
import { EUserPermissions, IS_FAVORITE_MENU_OPEN } from "@plane/constants";
import { useLocalStorage } from "@plane/hooks";
import { Button } from "@plane/propel/button";
import { Logo } from "@plane/propel/emoji-icon-picker";
import { LinkIcon, LockIcon, NewTabIcon, TrashIcon, CheckIcon } from "@plane/propel/icons";
import { setPromiseToast, setToast, TOAST_TYPE } from "@plane/propel/toast";
import { Tooltip } from "@plane/propel/tooltip";
import type { IProject } from "@plane/types";
import type { TContextMenuItem } from "@plane/ui";
import { Avatar, AvatarGroup, ContextMenu } from "@plane/ui";
import { copyUrlToClipboard } from "@plane/utils/string";
import { cn } from "@plane/utils/common";
import { getFileURL } from "@plane/utils/file";
import { renderFormattedDate } from "@plane/utils/datetime";
// components
// hooks
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useProjectPermissions, useWorkspacePermissions } from "@/hooks/use-permissions";
import { useFavorites } from "@/hooks/use-favorites";

import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { CoverImage } from "@/components/common/cover-image";
import { AuthorizedFavoriteStar } from "@/components/common/authorized-favorite-star";
import { DeleteProjectModal } from "./delete-project-modal";
import { JoinProjectModal } from "./join-project-modal";
import { ArchiveRestoreProjectModal } from "./archive-restore-modal";

type Props = {
  project: IProject;
};

type TProjectCardMenuActions = {
  isArchived: boolean;
  isMemberOfProject: boolean;
  hasAdminRole: boolean;
  hasMemberRole: boolean;
  openSettings: () => void;
  joinProject: () => void;
  openInNewTab: () => void;
  copyLink: () => void;
  restoreProject: () => void;
  deleteProject: () => void;
};

function getProjectCardMenuItems(actions: TProjectCardMenuActions): TContextMenuItem[] {
  const {
    isArchived,
    isMemberOfProject,
    hasAdminRole,
    hasMemberRole,
    openSettings,
    joinProject,
    openInNewTab,
    copyLink,
    restoreProject,
    deleteProject,
  } = actions;

  return [
    {
      key: "settings",
      action: openSettings,
      title: "Settings",
      icon: Settings,
      shouldRender: !isArchived && (hasAdminRole || hasMemberRole),
    },
    {
      key: "join",
      action: joinProject,
      title: "Join",
      icon: UserPlus,
      shouldRender: !isMemberOfProject && !isArchived,
    },
    {
      key: "open-new-tab",
      action: openInNewTab,
      title: "Open in new tab",
      icon: NewTabIcon,
      shouldRender: !isMemberOfProject && !isArchived,
    },
    {
      key: "copy-link",
      action: copyLink,
      title: "Copy link",
      icon: LinkIcon,
      shouldRender: !isArchived,
    },
    {
      key: "restore",
      action: restoreProject,
      title: "Restore",
      icon: ArchiveRestoreIcon,
      shouldRender: isArchived && hasAdminRole,
    },
    {
      key: "delete",
      action: deleteProject,
      title: "Delete",
      icon: TrashIcon,
      shouldRender: isArchived && hasAdminRole,
    },
  ];
}

type ProjectCardFooterProps = {
  project: IProject;
  workspaceSlug: string;
  memberById: ReturnType<typeof useWorkspaceMemberRoster>["memberById"];
  isMobile: boolean;
  isArchived: boolean;
  isMemberOfProject: boolean;
  hasAdminRole: boolean;
  hasMemberRole: boolean;
  onProjectNavigation: (event: React.MouseEvent<HTMLAnchorElement>) => void;
  onRestore: () => void;
  onDelete: () => void;
  onJoin: () => void;
};

function ProjectCardFooter(props: ProjectCardFooterProps) {
  const {
    project,
    workspaceSlug,
    memberById,
    isMobile,
    isArchived,
    isMemberOfProject,
    hasAdminRole,
    hasMemberRole,
    onProjectNavigation,
    onRestore,
    onDelete,
    onJoin,
  } = props;
  const projectMembersIds = project.members;
  return (
    <div
      className={cn(
        "pointer-events-none relative z-10 flex h-[104px] w-full flex-col justify-between rounded-b-sm p-4",
        {
          "opacity-90": isArchived,
        }
      )}
    >
      <p className="line-clamp-2 text-13 break-words text-tertiary">
        {project.description && project.description.trim() !== ""
          ? project.description
          : `Created on ${renderFormattedDate(project.created_at)}`}
      </p>
      <div className="item-center flex justify-between">
        <div className="flex items-center justify-center gap-2">
          <Link
            to="/$workspaceSlug/projects/$projectId/issues"
            params={{ workspaceSlug, projectId: project.id }}
            aria-label={`Open project ${project.name}`}
            onClick={onProjectNavigation}
            data-prevent-progress={!isMemberOfProject || isArchived}
            className="pointer-events-auto"
          >
            <Tooltip
              isMobile={isMobile}
              tooltipHeading="Members"
              tooltipContent={projectMembersIds?.length ? `${projectMembersIds.length} Members` : "No Member"}
              position="top"
            >
              {projectMembersIds?.length ? (
                <div className="flex cursor-pointer items-center gap-2 text-secondary">
                  <AvatarGroup showTooltip={false}>
                    {projectMembersIds.map((memberId) => {
                      const member = memberById(memberId)?.member;
                      if (!member) return null;
                      return <Avatar key={member.id} name={member.display_name} src={getFileURL(member.avatar_url)} />;
                    })}
                  </AvatarGroup>
                </div>
              ) : (
                <span className="text-13 text-placeholder italic">No Member Yet</span>
              )}
            </Tooltip>
          </Link>
          {isArchived && <div className="text-11 font-medium text-placeholder">Archived</div>}
        </div>
        <ProjectCardMembershipActions
          project={project}
          workspaceSlug={workspaceSlug}
          isArchived={isArchived}
          isMemberOfProject={isMemberOfProject}
          hasAdminRole={hasAdminRole}
          hasMemberRole={hasMemberRole}
          onRestore={onRestore}
          onDelete={onDelete}
          onJoin={onJoin}
        />
      </div>
    </div>
  );
}

type ProjectCardMembershipActionsProps = Pick<
  ProjectCardFooterProps,
  | "project"
  | "workspaceSlug"
  | "isArchived"
  | "isMemberOfProject"
  | "hasAdminRole"
  | "hasMemberRole"
  | "onRestore"
  | "onDelete"
  | "onJoin"
>;

function ProjectCardMembershipActions(props: ProjectCardMembershipActionsProps) {
  const {
    project,
    workspaceSlug,
    isArchived,
    isMemberOfProject,
    hasAdminRole,
    hasMemberRole,
    onRestore,
    onDelete,
    onJoin,
  } = props;
  if (isArchived) {
    if (!hasAdminRole) return null;
    return (
      <div className="pointer-events-auto relative z-20 flex items-center justify-center gap-2">
        <button
          type="button"
          className="flex items-center justify-center text-11 font-medium text-placeholder hover:text-secondary"
          onClick={onRestore}
        >
          <span className="flex items-center gap-1.5">
            <ArchiveRestoreIcon className="h-3.5 w-3.5" />
            Restore
          </span>
        </button>
        <button
          type="button"
          aria-label="Delete project"
          className="flex items-center justify-center text-11 font-medium text-placeholder hover:text-secondary"
          onClick={onDelete}
        >
          <TrashIcon className="h-3.5 w-3.5" />
        </button>
      </div>
    );
  }
  if (!isMemberOfProject)
    return (
      <div className="pointer-events-auto relative z-20 flex items-center">
        <Button variant="link" className="!p-0 font-semibold" onClick={onJoin}>
          Join
        </Button>
      </div>
    );
  if (hasAdminRole || hasMemberRole)
    return (
      <Link
        aria-label={`Project settings for ${project.name}`}
        className="pointer-events-auto relative z-20 flex items-center justify-center rounded-sm p-1 text-placeholder hover:bg-layer-1 hover:text-secondary"
        onClick={(event) => event.stopPropagation()}
        to="/$workspaceSlug/settings/projects/$projectId/"
        params={{ workspaceSlug, projectId: project.id }}
      >
        <Settings className="h-3.5 w-3.5" />
      </Link>
    );
  return (
    <span className="flex items-center gap-1 text-13 text-placeholder">
      <CheckIcon className="h-3.5 w-3.5" />
      Joined
    </span>
  );
}

export const ProjectCard = function ProjectCard(props: Props) {
  const { project } = props;
  // states
  const [deleteProjectModalOpen, setDeleteProjectModal] = useState(false);
  const [joinProjectModalOpen, setJoinProjectModal] = useState(false);
  const [restoreProject, setRestoreProject] = useState(false);
  // refs
  const projectCardRef = useRef(null);
  // router
  const router = useRouter();
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  const { favorites, favoritesByEntity, toggleFavorite } = useFavorites(workspaceSlug);
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);
  const { projectRole } = useProjectPermissions(workspaceSlug, project.id);
  // hooks
  const { isMobile } = usePlatformOS();
  // derived values
  const isProjectFavorite = favoritesByEntity?.has(`project:${project.id}`) ?? false;
  const isMemberOfProject = projectRole !== undefined;
  const canCurrentUserFavoriteProject = canWorkspace([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);
  // auth
  const hasAdminRole = projectRole === EUserPermissions.ADMIN;
  const hasMemberRole = projectRole === EUserPermissions.MEMBER;
  // archive
  const isArchived = !!project.archived_at;
  // local storage
  const { setValue: toggleFavoriteMenu, storedValue: isFavoriteMenuOpen } = useLocalStorage<boolean>(
    IS_FAVORITE_MENU_OPEN,
    false
  );

  const handleAddToFavorites = () => {
    if (!workspaceSlug) return;

    const addToFavoritePromise = toggleFavorite("project", project.id, project.id);
    setPromiseToast(addToFavoritePromise, {
      loading: "Adding project to favorites...",
      success: {
        title: "Success!",
        message: () => "Project added to favorites.",
        actionItems: () => {
          if (!isFavoriteMenuOpen) toggleFavoriteMenu(true);
          return null;
        },
      },
      error: {
        title: "Error!",
        message: () => "Couldn't add the project to favorites. Please try again.",
      },
    });
  };

  const handleRemoveFromFavorites = () => {
    if (!workspaceSlug) return;

    const removeFromFavoritePromise = toggleFavorite("project", project.id, project.id);
    setPromiseToast(removeFromFavoritePromise, {
      loading: "Removing project from favorites...",
      success: {
        title: "Success!",
        message: () => "Project removed from favorites.",
      },
      error: {
        title: "Error!",
        message: () => "Couldn't remove the project from favorites. Please try again.",
      },
    });
  };

  const projectLink = `${workspaceSlug}/projects/${project.id}/issues`;
  const handleProjectNavigation = (event: React.MouseEvent<HTMLAnchorElement>) => {
    if (!isMemberOfProject || isArchived) {
      event.preventDefault();
      event.stopPropagation();
      if (!isArchived) setJoinProjectModal(true);
    }
  };
  const handleCopyText = () => {
    void copyUrlToClipboard(projectLink)
      .then(() =>
        setToast({
          type: TOAST_TYPE.INFO,
          title: "Link Copied!",
          message: "Project link copied to clipboard.",
        })
      )
      .catch(() => {
        setToast({ type: TOAST_TYPE.ERROR, title: "Could not copy to clipboard", message: "Please try again." });
      });
  };
  const handleOpenInNewTab = () => window.open(`/${projectLink}`, "_blank");

  const menuItems = getProjectCardMenuItems({
    isArchived,
    isMemberOfProject,
    hasAdminRole,
    hasMemberRole,
    openSettings: () => void router.navigate({ href: `/${workspaceSlug}/settings/projects/${project.id}` }),
    joinProject: () => setJoinProjectModal(true),
    openInNewTab: handleOpenInNewTab,
    copyLink: handleCopyText,
    restoreProject: () => setRestoreProject(true),
    deleteProject: () => setDeleteProjectModal(true),
  });

  return (
    <>
      {/* Delete Project Modal */}
      <DeleteProjectModal
        project={project}
        isOpen={deleteProjectModalOpen}
        onClose={() => setDeleteProjectModal(false)}
      />
      {/* Join Project Modal */}
      {workspaceSlug && (
        <JoinProjectModal
          workspaceSlug={workspaceSlug}
          project={project}
          isOpen={joinProjectModalOpen}
          handleClose={() => setJoinProjectModal(false)}
        />
      )}
      {/* Restore project modal */}
      {workspaceSlug && project && (
        <ArchiveRestoreProjectModal
          workspaceSlug={workspaceSlug}
          projectId={project.id}
          isOpen={restoreProject}
          onClose={() => setRestoreProject(false)}
          archive={false}
        />
      )}
      <div
        ref={projectCardRef}
        className={cn(
          "group/project-card relative isolate flex w-full flex-col justify-between overflow-hidden rounded-lg border border-subtle bg-layer-2 transition-all duration-300 hover:border-strong hover:shadow-raised-200"
        )}
      >
        <Link
          aria-label={`Open project ${project.name}`}
          className="focus-visible:outline-focus-ring absolute inset-0 z-0 rounded-lg focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2"
          to="/$workspaceSlug/projects/$projectId/issues/"
          params={{ workspaceSlug, projectId: project.id }}
          onClick={handleProjectNavigation}
          data-prevent-progress={!isMemberOfProject || isArchived}
        />
        <ContextMenu parentRef={projectCardRef} items={menuItems} />
        <div className="pointer-events-none relative z-10 h-[118px] w-full rounded-t">
          <div className="absolute inset-0 z-[1] bg-gradient-to-t from-black/60 to-transparent" />

          <CoverImage
            src={project.cover_image_url}
            alt={project.name}
            className="absolute top-0 left-0 h-full w-full rounded-t"
          />

          <div className="absolute bottom-4 z-[1] flex h-10 w-full items-center justify-between gap-3 px-4">
            <div className="flex flex-grow items-center gap-2.5 truncate">
              <div className="grid h-9 w-9 flex-shrink-0 place-items-center rounded-sm bg-white/10">
                <Logo logo={project.logo_props} size={18} />
              </div>

              <div className="flex w-full flex-col justify-between gap-0.5 truncate">
                <h3 className="truncate font-semibold text-on-color">{project.name}</h3>
                <span className="flex items-center gap-1.5">
                  <p className="text-11 font-medium text-on-color">{project.identifier} </p>
                  {project.network === 0 && <LockIcon className="h-2.5 w-2.5 text-on-color" />}
                </span>
              </div>
            </div>

            {!isArchived && (
              <div
                data-prevent-progress
                className="pointer-events-auto relative z-20 flex h-full flex-shrink-0 items-center gap-2"
              >
                <button
                  className="flex h-6 w-6 items-center justify-center rounded-sm bg-white/10"
                  onClick={(e) => {
                    e.stopPropagation();
                    e.preventDefault();
                    handleCopyText();
                  }}
                >
                  <LinkIcon className="h-3 w-3 text-on-color" />
                </button>
                <AuthorizedFavoriteStar
                  canFavorite={canCurrentUserFavoriteProject && isMemberOfProject}
                  isReady={favorites !== undefined}
                  onAdd={handleAddToFavorites}
                  onRemove={handleRemoveFromFavorites}
                  buttonClassName="h-6 w-6 bg-white/10 rounded-sm"
                  iconClassName={cn("h-3 w-3", {
                    "text-on-color": !isProjectFavorite,
                  })}
                  selected={isProjectFavorite}
                  preventDefault
                  stopPropagation
                />
              </div>
            )}
          </div>
        </div>

        <ProjectCardFooter
          project={project}
          workspaceSlug={workspaceSlug}
          memberById={memberById}
          isMobile={isMobile}
          isArchived={isArchived}
          isMemberOfProject={isMemberOfProject}
          hasAdminRole={hasAdminRole}
          hasMemberRole={hasMemberRole}
          onProjectNavigation={handleProjectNavigation}
          onRestore={() => setRestoreProject(true)}
          onDelete={() => setDeleteProjectModal(true)}
          onJoin={() => setJoinProjectModal(true)}
        />
      </div>
    </>
  );
};
