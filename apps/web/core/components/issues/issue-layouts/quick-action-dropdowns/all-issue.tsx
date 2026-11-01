/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { ARCHIVABLE_STATE_GROUPS } from "@plane/constants";
import { EIssuesStoreType } from "@plane/types";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useRouteProjectStates } from "@/hooks/use-project-states";
// helper
import { IssueQuickActionControls } from "./issue-quick-action-modals";
import type { IQuickActionProps } from "../list/list-view-types";
import type { MenuItemFactoryProps } from "./helper";
import { useIssueMenuItems } from "./helper";
import { useIssueQuickActionModals } from "./use-issue-quick-action-modals";

export const AllIssueQuickActions = function AllIssueQuickActions(props: IQuickActionProps) {
  const { issue, handleDelete, handleUpdate, handleArchive, readOnly = false, placements = "bottom-start" } = props;
  // states
  const modals = useIssueQuickActionModals(issue);
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { getStateById } = useRouteProjectStates(issue.project_id);
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  // derived values
  const stateDetails = getStateById(issue.state_id);
  const isEditingAllowed = !readOnly;
  const projectIdentifier = projectRows?.find((project) => project.id === issue?.project_id)?.identifier ?? "";
  // auth
  const isArchivingAllowed = handleArchive && isEditingAllowed;
  const isInArchivableGroup = !!stateDetails && ARCHIVABLE_STATE_GROUPS.includes(stateDetails?.group);

  // Menu items and modals using helper
  const menuItemProps: MenuItemFactoryProps = {
    issue,
    workspaceSlug,
    projectIdentifier,
    isEditingAllowed,
    isArchivingAllowed,
    isDeletingAllowed: isEditingAllowed,
    isInArchivableGroup,
    setIssueToEdit: modals.setIssueToEdit,
    setCreateUpdateIssueModal: modals.setCreateUpdateIssueModal,
    setDeleteIssueModal: modals.setDeleteIssueModal,
    setArchiveIssueModal: modals.setArchiveIssueModal,
    handleDelete,
    handleUpdate,
    handleArchive,
    storeType: EIssuesStoreType.GLOBAL,
  };

  const MENU_ITEMS = useIssueMenuItems(menuItemProps);

  return (
    <IssueQuickActionControls
      {...props}
      placements={placements}
      items={MENU_ITEMS}
      modals={modals}
      storeType={EIssuesStoreType.GLOBAL}
    />
  );
};
