/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EIssuesStoreType } from "@plane/types";
import type { IQuickActionProps } from "../list/list-view-types";
import type { useIssueQuickActionModals } from "./use-issue-quick-action-modals";
import { ArchiveIssueModal } from "../../archive-issue-modal";
import { DeleteIssueModal } from "../../delete-issue-modal";
import { CreateUpdateIssueModal } from "../../issue-modal/modal";

type TIssueQuickActionModalsProps = Pick<
  IQuickActionProps,
  "issue" | "handleDelete" | "handleUpdate" | "handleArchive"
> & {
  modals: ReturnType<typeof useIssueQuickActionModals>;
  storeType: EIssuesStoreType;
  fetchIssueDetails?: boolean;
  onArchiveClose?: () => void;
  onDeleteClose?: () => void;
  onCreateUpdateClose?: (wasEditing: boolean) => void;
};

export function IssueQuickActionModals(props: TIssueQuickActionModalsProps) {
  const {
    issue,
    handleDelete,
    handleUpdate,
    handleArchive,
    modals,
    storeType,
    fetchIssueDetails,
    onArchiveClose,
    onDeleteClose,
    onCreateUpdateClose,
  } = props;

  const closeArchive = () => {
    modals.setArchiveIssueModal(false);
    onArchiveClose?.();
  };
  const closeDelete = () => {
    modals.setDeleteIssueModal(false);
    onDeleteClose?.();
  };
  const closeCreateUpdate = () => {
    const wasEditing = !!modals.issueToEdit;
    modals.closeCreateUpdateIssueModal();
    onCreateUpdateClose?.(wasEditing);
  };

  return (
    <>
      <ArchiveIssueModal
        data={issue}
        isOpen={modals.archiveIssueModal}
        handleClose={closeArchive}
        onSubmit={handleArchive}
      />
      <DeleteIssueModal
        data={issue}
        isOpen={modals.deleteIssueModal}
        handleClose={closeDelete}
        onSubmit={handleDelete}
      />
      <CreateUpdateIssueModal
        isOpen={modals.createUpdateIssueModal}
        onClose={closeCreateUpdate}
        data={modals.data}
        onSubmit={async (data) => {
          if (modals.issueToEdit && handleUpdate) await handleUpdate(data);
        }}
        storeType={storeType}
        fetchIssueDetails={fetchIssueDetails}
      />
    </>
  );
}
