/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { EIssuesStoreType } from "@plane/types";
import { QuickActionMenu } from "@/components/common/quick-action-menu";
// hooks
// helper
import { IssueQuickActionModals } from "./issue-quick-action-modals";
import type { IQuickActionProps } from "../list/list-view-types";
import type { MenuItemFactoryProps } from "./helper";
import { useCycleIssueMenuItems, useIssueQuickActionContext } from "./helper";
import { useIssueQuickActionModals } from "./use-issue-quick-action-modals";

export const CycleIssueQuickActions = function CycleIssueQuickActions(props: IQuickActionProps) {
  const {
    issue,
    handleDelete,
    handleUpdate,
    handleRemoveFromView,
    handleArchive,
    customActionButton,
    portalElement,
    readOnly = false,
    placements = "bottom-start",
    parentRef,
  } = props;
  // states
  const modals = useIssueQuickActionModals(issue);
  // router
  const { cycleId } = useParams({ strict: false });
  const {
    workspaceSlug: contextWorkspaceSlug,
    projectIdentifier,
    isEditingAllowed,
    isInArchivableGroup,
  } = useIssueQuickActionContext(issue, readOnly);
  // derived values
  const isArchivingAllowed = handleArchive && isEditingAllowed;
  const isDeletingAllowed = isEditingAllowed;

  // Menu items and modals using helper
  const menuItemProps: MenuItemFactoryProps = {
    issue,
    workspaceSlug: contextWorkspaceSlug?.toString(),
    projectIdentifier,
    isEditingAllowed,
    isArchivingAllowed,
    isDeletingAllowed,
    isInArchivableGroup,
    setIssueToEdit: modals.setIssueToEdit,
    setCreateUpdateIssueModal: modals.setCreateUpdateIssueModal,
    setDeleteIssueModal: modals.setDeleteIssueModal,
    setArchiveIssueModal: modals.setArchiveIssueModal,
    handleRemoveFromView,
    cycleId: cycleId?.toString(),
    handleDelete,
    handleUpdate,
    handleArchive,
    storeType: EIssuesStoreType.CYCLE,
  };

  const MENU_ITEMS = useCycleIssueMenuItems(menuItemProps);

  return (
    <>
      <IssueQuickActionModals
        issue={issue}
        handleDelete={handleDelete}
        handleUpdate={handleUpdate}
        handleArchive={handleArchive}
        modals={modals}
        storeType={EIssuesStoreType.CYCLE}
      />

      <QuickActionMenu
        items={MENU_ITEMS}
        parentRef={parentRef}
        customMenuProps={{
          ellipsis: true,
          customButton: customActionButton,
          portalElement,
          placement: placements,
          menuItemsClassName: "z-[14]",
          maxHeight: "lg",
          useCaptureForOutsideClick: true,
          closeOnSelect: true,
        }}
      />
    </>
  );
};
