/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { ARCHIVABLE_STATE_GROUPS, EUserPermissions } from "@plane/constants";
import { EIssuesStoreType } from "@plane/types";
import { QuickActionMenu } from "@/components/common/quick-action-menu";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useProjectStates } from "@/hooks/use-project-states";
import { useProjectPermissions } from "@/hooks/use-permissions";
// helper
import { IssueQuickActionModals } from "./issue-quick-action-modals";
import type { IQuickActionProps } from "../list/list-view-types";
import type { MenuItemFactoryProps } from "./helper";
import { useModuleIssueMenuItems } from "./helper";
import { useIssueQuickActionModals } from "./use-issue-quick-action-modals";

export const ModuleIssueQuickActions = function ModuleIssueQuickActions(props: IQuickActionProps) {
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
  const { workspaceSlug, moduleId } = useParams({ strict: false });
  // store hooks
  const { can: canProject } = useProjectPermissions(workspaceSlug, issue.project_id ?? undefined);
  const { getStateById } = useProjectStates(workspaceSlug, issue.project_id);
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  // derived values
  const stateDetails = getStateById(issue.state_id);
  const projectIdentifier = projectRows?.find((project) => project.id === issue?.project_id)?.identifier ?? "";
  // auth
  const isEditingAllowed = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]) && !readOnly;
  const isArchivingAllowed = handleArchive && isEditingAllowed;
  const isInArchivableGroup = !!stateDetails && ARCHIVABLE_STATE_GROUPS.includes(stateDetails?.group);
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
    handleRemoveFromView,
    moduleId: moduleId?.toString(),
    handleDelete,
    handleUpdate,
    handleArchive,
    storeType: EIssuesStoreType.MODULE,
  };

  const MENU_ITEMS = useModuleIssueMenuItems(menuItemProps);

  return (
    <>
      <IssueQuickActionModals
        issue={issue}
        handleDelete={handleDelete}
        handleUpdate={handleUpdate}
        handleArchive={handleArchive}
        modals={modals}
        storeType={EIssuesStoreType.MODULE}
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
