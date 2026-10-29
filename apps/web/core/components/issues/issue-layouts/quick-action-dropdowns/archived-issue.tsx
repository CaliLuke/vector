/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useState } from "react";

// ui
import { EUserPermissions } from "@plane/constants";
import { ContextMenu, CustomMenu } from "@plane/ui";
import { renderQuickActionMenuItems } from "@/components/common/quick-action-menu";
// hooks
import { useProjectPermissions } from "@/hooks/use-permissions";
// local imports
import { DeleteIssueModal } from "../../delete-issue-modal";
import type { IQuickActionProps } from "../list/list-view-types";
import type { MenuItemFactoryProps } from "./helper";
import { useArchivedIssueMenuItems } from "./helper";

export const ArchivedIssueQuickActions = function ArchivedIssueQuickActions(props: IQuickActionProps) {
  const {
    issue,
    handleDelete,
    handleRestore,
    customActionButton,
    portalElement,
    readOnly = false,
    placements = "bottom-end",
    parentRef,
  } = props;
  // states
  const [deleteIssueModal, setDeleteIssueModal] = useState(false);
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  const { can: canProject } = useProjectPermissions(workspaceSlug ?? undefined, issue.project_id ?? undefined);

  // auth
  const isEditingAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]) && !readOnly;
  const isRestoringAllowed = !!handleRestore && isEditingAllowed;

  // Menu items and modals using helper
  const menuItemProps: MenuItemFactoryProps = {
    issue,
    workspaceSlug: workspaceSlug?.toString(),
    isEditingAllowed,
    isDeletingAllowed: isEditingAllowed,
    isRestoringAllowed,
    setIssueToEdit: () => {},
    setCreateUpdateIssueModal: () => {},
    setDeleteIssueModal,
    handleRestore,
    handleDelete,
  };

  const MENU_ITEMS = useArchivedIssueMenuItems(menuItemProps);

  const CONTEXT_MENU_ITEMS = MENU_ITEMS.map(function CONTEXT_MENU_ITEMS(item) {
    return {
      ...item,

      onClick: () => {
        item.action();
      },
    };
  });
  return (
    <>
      {/* Modals */}
      <DeleteIssueModal
        data={issue}
        isOpen={deleteIssueModal}
        handleClose={() => setDeleteIssueModal(false)}
        onSubmit={handleDelete}
      />

      <ContextMenu parentRef={parentRef} items={CONTEXT_MENU_ITEMS} />
      <CustomMenu
        ellipsis
        customButton={customActionButton}
        portalElement={portalElement}
        placement={placements}
        menuItemsClassName="z-[14]"
        maxHeight="lg"
        useCaptureForOutsideClick
        closeOnSelect
      >
        {renderQuickActionMenuItems(MENU_ITEMS)}
      </CustomMenu>
    </>
  );
};
