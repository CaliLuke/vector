/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useState } from "react";
import { omit } from "lodash-es";
import { observer } from "mobx-react";

// plane imports
import { ARCHIVABLE_STATE_GROUPS, EUserPermissions, EUserPermissionsLevel } from "@plane/constants";
import type { TIssue } from "@plane/types";
import { EIssuesStoreType } from "@plane/types";
import { QuickActionMenu } from "./quick-action-menu";
// hooks
import { useIssues } from "@/hooks/store/use-issues";
import { useProject } from "@/hooks/store/use-project";
import { useProjectState } from "@/hooks/store/use-project-state";
import { useUserPermissions } from "@/hooks/store/user";
// helper
import { ArchiveIssueModal } from "../../archive-issue-modal";
import { DeleteIssueModal } from "../../delete-issue-modal";
import { CreateUpdateIssueModal } from "../../issue-modal/modal";
import type { IQuickActionProps } from "../list/list-view-types";
import type { MenuItemFactoryProps } from "./helper";
import { useCycleIssueMenuItems } from "./helper";

export const CycleIssueQuickActions = observer(function CycleIssueQuickActions(props: IQuickActionProps) {
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
  const [createUpdateIssueModal, setCreateUpdateIssueModal] = useState(false);
  const [issueToEdit, setIssueToEdit] = useState<TIssue | undefined>(undefined);
  const [deleteIssueModal, setDeleteIssueModal] = useState(false);
  const [archiveIssueModal, setArchiveIssueModal] = useState(false);
  const [_, setDuplicateWorkItemModal] = useState(false);
  // router
  const { workspaceSlug, cycleId } = useParams({ strict: false });
  const { issuesFilter } = useIssues(EIssuesStoreType.CYCLE);
  const { allowPermissions } = useUserPermissions();
  const { getStateById } = useProjectState();
  const { getProjectIdentifierById } = useProject();
  // derived values
  const stateDetails = getStateById(issue.state_id);
  const projectIdentifier = getProjectIdentifierById(issue?.project_id);
  // auth
  const isEditingAllowed =
    allowPermissions([EUserPermissions.ADMIN, EUserPermissions.MEMBER], EUserPermissionsLevel.PROJECT) && !readOnly;
  const isArchivingAllowed = handleArchive && isEditingAllowed;
  const isInArchivableGroup = !!stateDetails && ARCHIVABLE_STATE_GROUPS.includes(stateDetails?.group);
  const isDeletingAllowed = isEditingAllowed;

  const activeLayout = `${issuesFilter.issueFilters?.displayFilters?.layout} layout`;

  const duplicateIssuePayload = omit(
    {
      ...issue,
      name: `${issue.name} (copy)`,
      sourceIssueId: issue.id,
    },
    ["id"]
  );

  // Menu items and modals using helper
  const menuItemProps: MenuItemFactoryProps = {
    issue,
    workspaceSlug: workspaceSlug?.toString(),
    projectIdentifier,
    activeLayout,
    isEditingAllowed,
    isArchivingAllowed,
    isDeletingAllowed,
    isInArchivableGroup,
    setIssueToEdit,
    setCreateUpdateIssueModal,
    setDeleteIssueModal,
    setArchiveIssueModal,
    setDuplicateWorkItemModal,
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
      {/* Modals */}
      <ArchiveIssueModal
        data={issue}
        isOpen={archiveIssueModal}
        handleClose={() => setArchiveIssueModal(false)}
        onSubmit={handleArchive}
      />
      <DeleteIssueModal
        data={issue}
        isOpen={deleteIssueModal}
        handleClose={() => setDeleteIssueModal(false)}
        onSubmit={handleDelete}
      />
      <CreateUpdateIssueModal
        isOpen={createUpdateIssueModal}
        onClose={() => {
          setCreateUpdateIssueModal(false);
          setIssueToEdit(undefined);
        }}
        data={issueToEdit ?? duplicateIssuePayload}
        onSubmit={async (data) => {
          if (issueToEdit && handleUpdate) await handleUpdate(data);
        }}
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
});
