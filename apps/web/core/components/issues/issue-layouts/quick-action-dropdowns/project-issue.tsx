/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { EIssuesStoreType } from "@plane/types";
import { QuickActionMenu } from "@/components/common/quick-action-menu";
// hooks
// helper
import { IssueQuickActionModals } from "./issue-quick-action-modals";
import type { IQuickActionProps } from "../list/list-view-types";
import type { MenuItemFactoryProps } from "./helper";
import { useProjectIssueMenuItems, useIssueQuickActionContext } from "./helper";
import { useIssueQuickActionModals } from "./use-issue-quick-action-modals";

export const ProjectIssueQuickActions = function ProjectIssueQuickActions(props: IQuickActionProps) {
  const {
    issue,
    handleDelete,
    handleUpdate,
    handleArchive,
    customActionButton,
    portalElement,
    readOnly = false,
    placements = "bottom-end",
    parentRef,
  } = props;
  // states
  const modals = useIssueQuickActionModals(issue);
  const { workspaceSlug, projectIdentifier, isEditingAllowed, isInArchivableGroup } = useIssueQuickActionContext(
    issue,
    readOnly
  );
  const isArchivingAllowed = handleArchive && isEditingAllowed;
  const isDeletingAllowed = isEditingAllowed;

  // Menu items and modals using helper
  const menuItemProps: MenuItemFactoryProps = {
    issue,
    workspaceSlug: workspaceSlug?.toString(),
    projectIdentifier,
    isEditingAllowed,
    isArchivingAllowed,
    isDeletingAllowed,
    isInArchivableGroup,
    setIssueToEdit: modals.setIssueToEdit,
    setCreateUpdateIssueModal: modals.setCreateUpdateIssueModal,
    setDeleteIssueModal: modals.setDeleteIssueModal,
    setArchiveIssueModal: modals.setArchiveIssueModal,
    handleDelete,
    handleUpdate,
    handleArchive,
    storeType: EIssuesStoreType.PROJECT,
  };

  const MENU_ITEMS = useProjectIssueMenuItems(menuItemProps);

  return (
    <>
      <IssueQuickActionModals
        issue={issue}
        handleDelete={handleDelete}
        handleUpdate={handleUpdate}
        handleArchive={handleArchive}
        modals={modals}
        storeType={EIssuesStoreType.PROJECT}
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
