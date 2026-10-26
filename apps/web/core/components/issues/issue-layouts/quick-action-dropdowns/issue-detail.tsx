/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Ellipsis } from "lucide-react";
// plane imports
import { EIssuesStoreType } from "@plane/types";
import { QuickActionMenu } from "@/components/common/quick-action-menu";
// hooks
// helper
import { IssueQuickActionModals } from "./issue-quick-action-modals";
import type { IQuickActionProps } from "../list/list-view-types";
import type { MenuItemFactoryProps } from "./helper";
import { useIssueQuickActionContext, useWorkItemDetailMenuItems } from "./helper";
import { IconButton } from "@plane/propel/icon-button";
import { useIssueQuickActionModals } from "./use-issue-quick-action-modals";

type TWorkItemDetailQuickActionProps = IQuickActionProps & {
  toggleEditIssueModal?: (value: boolean) => void;
  toggleDeleteIssueModal?: (value: boolean) => void;
  toggleDuplicateIssueModal?: (value: boolean) => void;
  toggleArchiveIssueModal?: (value: boolean) => void;
  isPeekMode?: boolean;
};

export const WorkItemDetailQuickActions = function WorkItemDetailQuickActions(props: TWorkItemDetailQuickActionProps) {
  const {
    issue,
    handleDelete,
    handleUpdate,
    handleArchive,
    handleRestore,
    portalElement,
    readOnly = false,
    placements = "bottom-end",
    parentRef,
    toggleDeleteIssueModal,
    toggleDuplicateIssueModal,
    toggleArchiveIssueModal,
    isPeekMode = false,
  } = props;
  // states
  const modals = useIssueQuickActionModals(issue);
  const { workspaceSlug, projectIdentifier, isEditingAllowed, isInArchivableGroup } = useIssueQuickActionContext(
    issue,
    readOnly
  );

  const isArchivingAllowed = !issue.archived_at && isEditingAllowed;
  const isRestoringAllowed = !!issue.archived_at && isEditingAllowed;

  const isDeletingAllowed = isEditingAllowed;

  const customDeleteAction = async () => {
    modals.setDeleteIssueModal(true);
    if (toggleDeleteIssueModal) toggleDeleteIssueModal(true);
  };

  const customDuplicateAction = () => {
    modals.setIssueToEdit(undefined);
    modals.setCreateUpdateIssueModal(true);
    if (toggleDuplicateIssueModal) {
      toggleDuplicateIssueModal(true);
    }
  };

  const customArchiveAction = async () => {
    modals.setArchiveIssueModal(true);
    if (toggleArchiveIssueModal) toggleArchiveIssueModal(true);
  };

  const customRestoreAction = async () => {
    if (handleRestore) await handleRestore();
  };

  // Menu items and modals using helper
  const menuItemProps: MenuItemFactoryProps = {
    issue,
    workspaceSlug: workspaceSlug?.toString(),
    projectIdentifier,
    isEditingAllowed,
    isArchivingAllowed,
    isRestoringAllowed,
    isDeletingAllowed,
    isInArchivableGroup,
    setIssueToEdit: modals.setIssueToEdit,
    setCreateUpdateIssueModal: modals.setCreateUpdateIssueModal,
    setDeleteIssueModal: customDeleteAction,
    setArchiveIssueModal: customArchiveAction,
    onDuplicate: toggleDuplicateIssueModal ? customDuplicateAction : undefined,
    handleDelete: customDeleteAction,
    handleUpdate,
    handleArchive: customArchiveAction,
    handleRestore: customRestoreAction,
    storeType: EIssuesStoreType.PROJECT,
  };

  //   const MENU_ITEMS = useWorkItemDetailMenuItems(menuItemProps);
  const baseMenuItems = useWorkItemDetailMenuItems(menuItemProps);

  const MENU_ITEMS = baseMenuItems
    // oxlint-disable-next-line oxc/no-map-spread
    .map((item) => {
      // Customize edit action for work item
      if (item.key === "edit") {
        return {
          ...item,
          shouldRender: isEditingAllowed && !isPeekMode,
        };
      }
      // Customize delete action for work item
      if (item.key === "delete") {
        return {
          ...item,
        };
      }
      // Hide copy link in peek mode
      if (item.key === "copy-link") {
        return {
          ...item,
          shouldRender: !isPeekMode,
        };
      }
      return item;
    })
    .filter(function MENU_ITEMS(item) {
      return item.shouldRender !== false;
    });

  return (
    <>
      <IssueQuickActionModals
        issue={issue}
        handleDelete={handleDelete}
        handleUpdate={handleUpdate}
        handleArchive={handleArchive}
        modals={modals}
        storeType={EIssuesStoreType.PROJECT}
        fetchIssueDetails={false}
        onArchiveClose={() => toggleArchiveIssueModal?.(false)}
        onDeleteClose={() => toggleDeleteIssueModal?.(false)}
        onCreateUpdateClose={(wasEditing) => {
          if (!wasEditing) toggleDuplicateIssueModal?.(false);
        }}
      />

      <QuickActionMenu
        items={MENU_ITEMS}
        parentRef={parentRef}
        customMenuProps={{
          customButtonAsChild: true,
          placement: placements,
          customButton: <IconButton size="lg" variant="secondary" icon={Ellipsis} />,
          portalElement,
          menuItemsClassName: "z-[14]",
          maxHeight: "lg",
          closeOnSelect: true,
        }}
      />
    </>
  );
};
