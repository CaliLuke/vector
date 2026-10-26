/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React, { useState } from "react";

import { Earth } from "lucide-react";
// plane imports
import { EUserPermissions, IS_FAVORITE_MENU_OPEN } from "@plane/constants";
import { useLocalStorage } from "@plane/hooks";
import { LockIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { IProjectView } from "@plane/types";
import { EViewAccess } from "@plane/types";
import { getPublishViewLink } from "@plane/utils/project-views";
// hooks
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useFavorites } from "@/hooks/use-favorites";
// local imports
import { ButtonAvatars } from "../dropdowns/member/avatar";
import { AuthorizedFavoriteStar } from "@/components/common/authorized-favorite-star";
import { DeleteProjectViewModal } from "./delete-view-modal";
import { CreateUpdateProjectViewModal } from "./modal";
import { ViewQuickActions } from "./quick-actions";

type Props = {
  parentRef: React.RefObject<HTMLElement | null>;
  view: IProjectView;
};

function ViewPublishStatus({
  anchor,
  publishLink,
  onOpen,
}: {
  anchor: string | null | undefined;
  publishLink: string | null | undefined;
  onOpen: () => void;
}) {
  if (!anchor || !publishLink) return null;

  return (
    // oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions
    <div
      className="flex cursor-pointer items-center gap-1.5 rounded-sm bg-success-subtle px-3 py-1.5 text-11 font-medium text-success-primary"
      onClick={onOpen}
    >
      <span className="size-1.5 flex-shrink-0 rounded-full bg-success-primary" />
      Live
    </div>
  );
}

export const ViewListItemAction = function ViewListItemAction(props: Props) {
  const { parentRef, view } = props;
  // states
  const [createUpdateViewModal, setCreateUpdateViewModal] = useState(false);
  const [deleteViewModal, setDeleteViewModal] = useState(false);
  const [_, setPublishModalOpen] = useState<boolean>(false);
  // router
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // store
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);

  const { favorites, favoritesByEntity, toggleFavorite } = useFavorites(workspaceSlug);
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const getUserDetails = (userId: string) => membersById?.get(userId)?.member;

  // local storage
  const { setValue: toggleFavoriteMenu, storedValue: isFavoriteOpen } = useLocalStorage<boolean>(
    IS_FAVORITE_MENU_OPEN,
    false
  );

  // derived values
  const isEditingAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);
  const isFavorite = favoritesByEntity?.has(`view:${view.id}`) ?? false;

  const access = view.access;

  const publishLink = getPublishViewLink(view?.anchor);

  // handlers
  const handleAddToFavorites = async () => {
    if (!workspaceSlug || !projectId) return;

    await toggleFavorite("view", view.id, projectId.toString());
    if (!isFavoriteOpen) toggleFavoriteMenu(true);
  };

  const handleRemoveFromFavorites = () => {
    if (!workspaceSlug || !projectId) return;

    void toggleFavorite("view", view.id, projectId.toString());
  };

  const ownedByDetails = view.owned_by ? getUserDetails(view.owned_by) : undefined;

  return (
    <>
      {workspaceSlug && projectId && view && (
        <CreateUpdateProjectViewModal
          isOpen={createUpdateViewModal}
          onClose={() => setCreateUpdateViewModal(false)}
          workspaceSlug={workspaceSlug.toString()}
          projectId={projectId.toString()}
          data={view}
        />
      )}
      <DeleteProjectViewModal data={view} isOpen={deleteViewModal} onClose={() => setDeleteViewModal(false)} />
      <div className="cursor-default text-tertiary">
        <Tooltip tooltipContent={access === EViewAccess.PUBLIC ? "Public" : "Private"}>
          {access === EViewAccess.PUBLIC ? <Earth className="h-4 w-4" /> : <LockIcon className="h-4 w-4" />}
        </Tooltip>
      </div>

      <ViewPublishStatus anchor={view.anchor} publishLink={publishLink} onOpen={() => setPublishModalOpen(true)} />

      {/* created by */}
      {<ButtonAvatars showTooltip={false} userIds={ownedByDetails?.id ?? []} workspaceSlug={workspaceSlug} />}

      <AuthorizedFavoriteStar
        canFavorite={isEditingAllowed}
        isReady={favorites !== undefined}
        onAdd={handleAddToFavorites}
        onRemove={handleRemoveFromFavorites}
        selected={isFavorite}
        preventDefault
        stopPropagation
      />
      {projectId && workspaceSlug && (
        <div className="hidden md:block">
          <ViewQuickActions
            parentRef={parentRef}
            projectId={projectId.toString()}
            view={view}
            workspaceSlug={workspaceSlug.toString()}
          />
        </div>
      )}
    </>
  );
};
