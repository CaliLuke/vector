/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EPageAccess } from "@plane/types";
import type { TPage } from "@plane/types";

import { Earth, Info, Minus } from "lucide-react";
// plane imports
import { LockIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import { Avatar, FavoriteStar } from "@plane/ui";
import { renderFormattedDate } from "@plane/utils/datetime";
import { getFileURL } from "@plane/utils/file";
// hooks
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { usePageOperations } from "@/hooks/use-page-operations";
import { usePagePermissions } from "@/hooks/use-page-permissions";
// plane web hooks
// store
// local imports
import { PageActions } from "../dropdowns";

type Props = {
  page: TPage;
  parentRef: React.RefObject<HTMLElement | null>;
  projectId: string;
  workspaceSlug: string;
};

export function BlockItemAction(props: Props) {
  const { page, parentRef, projectId, workspaceSlug } = props;
  // store hooks
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  const { canCurrentUserFavoritePage } = usePagePermissions(page, workspaceSlug);
  // page operations
  const { pageOperations, isFavorite, isFavoriteReady } = usePageOperations({
    page,
    projectId,
    workspaceSlug,
  });
  // derived values
  const { access, created_at, owned_by } = page;
  const ownerDetails = owned_by ? memberById(owned_by)?.member : undefined;

  return (
    <>
      {/* page details */}
      <div className="cursor-default">
        <Tooltip tooltipHeading="Owned by" tooltipContent={ownerDetails?.display_name}>
          <Avatar src={getFileURL(ownerDetails?.avatar_url ?? "")} name={ownerDetails?.display_name} />
        </Tooltip>
      </div>
      <div className="cursor-default text-tertiary">
        <Tooltip tooltipContent={access === EPageAccess.PUBLIC ? "Public" : "Private"}>
          {access === EPageAccess.PUBLIC ? <Earth className="h-4 w-4" /> : <LockIcon className="h-4 w-4" />}
        </Tooltip>
      </div>
      {/* vertical divider */}
      <Minus className="-mx-3 h-5 w-5 rotate-90 text-placeholder" strokeWidth={1} />

      {/* page info */}
      <Tooltip tooltipContent={`Created on ${renderFormattedDate(created_at)}`}>
        <span className="grid h-4 w-4 cursor-default place-items-center">
          <Info className="h-4 w-4 text-tertiary" />
        </span>
      </Tooltip>

      {/* favorite/unfavorite */}
      {canCurrentUserFavoritePage && isFavoriteReady && (
        <FavoriteStar
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            void pageOperations.toggleFavorite();
          }}
          selected={isFavorite}
        />
      )}

      {/* quick actions dropdown */}
      <PageActions
        workspaceSlug={workspaceSlug}
        optionsOrder={[
          "open-in-new-tab",
          "copy-link",
          "make-a-copy",
          "toggle-lock",
          "toggle-access",
          "archive-restore",
          "delete",
        ]}
        page={page}
        projectId={projectId}
        parentRef={parentRef}
      />
    </>
  );
}
