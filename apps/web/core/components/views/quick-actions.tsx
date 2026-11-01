/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";

// types
import { EUserPermissions } from "@plane/constants";
import { useViewLinkCopy } from "@/components/views/use-view-link-copy";
import type { IProjectView } from "@plane/types";
// ui
import { QuickActionsMenu } from "@/components/common/quick-actions-menu";
import { normalizeQuickActions } from "@/components/common/quick-actions-helper";
// helpers
import { useViewMenuItems } from "@/components/common/quick-actions-helper";
// hooks
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useViewPublish } from "@/components/views/publish";
// local imports
import { DeleteProjectViewModal } from "./delete-view-modal";
import { CreateUpdateProjectViewModal } from "./modal";

type Props = {
  parentRef: React.RefObject<HTMLElement | null>;
  projectId: string;
  view: IProjectView;
  workspaceSlug: string;
  customClassName?: string;
};

export const ViewQuickActions = function ViewQuickActions(props: Props) {
  const { parentRef, projectId, view, workspaceSlug, customClassName } = props;
  // states
  const [createUpdateViewModal, setCreateUpdateViewModal] = useState(false);
  const [deleteViewModal, setDeleteViewModal] = useState(false);
  // store hooks
  const { data } = useCurrentUser();
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  // auth
  const isOwner = view?.owned_by === data?.id;
  const isAdmin = canProject([EUserPermissions.ADMIN]);

  const { publishContextMenu } = useViewPublish(!!view.anchor, isAdmin || isOwner);

  const viewLink = `${workspaceSlug}/projects/${projectId}/views/${view.id}`;
  const handleCopyText = useViewLinkCopy(viewLink);
  const handleOpenInNewTab = () => window.open(`/${viewLink}`, "_blank");

  const menuResult = useViewMenuItems({
    isOwner,
    isAdmin,
    workspaceSlug,
    projectId,
    view,
    handleEdit: () => setCreateUpdateViewModal(true),
    handleDelete: () => setDeleteViewModal(true),
    handleCopyLink: () => void handleCopyText(),
    handleOpenInNewTab,
  });

  const { items: MENU_ITEMS, modals: additionalModals } = normalizeQuickActions(menuResult);

  if (publishContextMenu) MENU_ITEMS.splice(2, 0, publishContextMenu);

  return (
    <>
      <CreateUpdateProjectViewModal
        isOpen={createUpdateViewModal}
        onClose={() => setCreateUpdateViewModal(false)}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        data={view}
      />
      <DeleteProjectViewModal data={view} isOpen={deleteViewModal} onClose={() => setDeleteViewModal(false)} />
      {additionalModals}
      <QuickActionsMenu parentRef={parentRef} items={MENU_ITEMS} className={customClassName} />
    </>
  );
};
