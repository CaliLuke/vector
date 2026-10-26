/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// plane imports
import { EUserPermissions } from "@plane/constants";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IWorkspaceView } from "@plane/types";
import { CustomMenu } from "@plane/ui";
import { copyUrlToClipboard } from "@plane/utils/string";
// helpers
import { useViewMenuItems } from "@/components/common/quick-actions-helper";
import { renderQuickActionMenuItems } from "@/components/common/quick-action-menu";
// hooks
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";
// local imports
import { DeleteGlobalViewModal } from "./delete-view-modal";
import { CreateUpdateWorkspaceViewModal } from "./modal";

type Props = {
  workspaceSlug: string;
  view: IWorkspaceView;
};

export function WorkspaceViewQuickActions(props: Props) {
  const { workspaceSlug, view } = props;
  // states
  const [updateViewModal, setUpdateViewModal] = useState(false);
  const [deleteViewModal, setDeleteViewModal] = useState(false);
  // store hooks
  const { data } = useCurrentUser();
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);
  // auth
  const isOwner = view?.owned_by === data?.id;
  const isAdmin = canWorkspace([EUserPermissions.ADMIN]);

  const viewLink = `${workspaceSlug}/workspace-views/${view.id}`;
  const handleCopyText = async () => {
    await copyUrlToClipboard(viewLink);
    setToast({
      type: TOAST_TYPE.SUCCESS,
      title: "Link Copied!",
      message: "View link copied to clipboard.",
    });
  };

  const handleOpenInNewTab = () => window.open(`/${viewLink}`, "_blank");

  const MENU_ITEMS = useViewMenuItems({
    isOwner,
    isAdmin,
    handleDelete: () => setDeleteViewModal(true),
    handleEdit: () => setUpdateViewModal(true),
    handleOpenInNewTab,
    handleCopyLink: handleCopyText,
    workspaceSlug,
    view,
  });

  return (
    <>
      <CreateUpdateWorkspaceViewModal data={view} isOpen={updateViewModal} onClose={() => setUpdateViewModal(false)} />
      <DeleteGlobalViewModal data={view} isOpen={deleteViewModal} onClose={() => setDeleteViewModal(false)} />
      <CustomMenu
        ellipsis
        placement="bottom-end"
        closeOnSelect
        buttonClassName="flex-shrink-0 flex items-center justify-center size-[26px] bg-layer-1/70 rounded-sm"
      >
        {renderQuickActionMenuItems(MENU_ITEMS.items)}
      </CustomMenu>
    </>
  );
}
