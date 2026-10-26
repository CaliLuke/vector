/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useEffect } from "react";
import { useParams } from "@tanstack/react-router";
import { EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
import { useItemEditLease } from "../use-item-edit-coordination";

// plane imports
import type { EIssuesStoreType, TIssue } from "@plane/types";
// local imports
import { CreateUpdateIssueModalBase } from "./base";
import { IssueModalProvider } from "./provider";

export interface IssuesModalProps {
  data?: Partial<TIssue>;
  isOpen: boolean;
  onClose: () => void;
  beforeFormSubmit?: () => Promise<void>;
  onSubmit?: (res: TIssue) => Promise<void>;
  withDraftIssueWrapper?: boolean;
  storeType?: EIssuesStoreType;
  isDraft?: boolean;
  fetchIssueDetails?: boolean;
  moveToIssue?: boolean;
  modalTitle?: string;
  primaryButtonText?: {
    default: string;
    loading: string;
  };
  isProjectSelectionDisabled?: boolean;
  allowedProjectIds?: string[];
  showActionItemsOnUpdate?: boolean;
}

export const CreateUpdateIssueModal = function CreateUpdateIssueModal(props: IssuesModalProps) {
  if (!props.isOpen) return null;
  return (
    <IssueModalProvider allowedProjectIds={props.allowedProjectIds}>
      {props.data?.id && !props.isDraft ? (
        <PublishedIssueEditor {...props} />
      ) : (
        <CreateUpdateIssueModalBase {...props} />
      )}
    </IssueModalProvider>
  );
};

function PublishedIssueEditor(props: IssuesModalProps) {
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const project = props.data?.project_id ?? projectId;
  if (!workspaceSlug || !project)
    return <PendingTextSave onClose={props.onClose} message="The work item's project is unavailable." />;
  return (
    <PublishedIssueEditorLease
      key={`${workspaceSlug}/${project}/${props.data!.id}`}
      scope={`${workspaceSlug}/${project}/${props.data!.id}`}
      {...props}
    />
  );
}
function PublishedIssueEditorLease({ scope, ...props }: IssuesModalProps & { scope: string }) {
  const { held, blocked, acquire } = useItemEditLease(scope);
  useEffect(() => {
    if (!blocked) acquire();
  }, [blocked, acquire]);
  if (!held)
    return (
      <PendingTextSave
        onClose={props.onClose}
        message="Waiting for title and description edits to finish saving. Close this editor to retry any failed saves."
      />
    );
  return <CreateUpdateIssueModalBase {...props} />;
}
function PendingTextSave({ onClose, message }: { onClose: () => void; message: string }) {
  return (
    <ModalCore isOpen position={EModalPosition.TOP} width={EModalWidth.XXXXL}>
      <div className="space-y-3 rounded-lg bg-surface-1 p-5">
        <output>{message}</output>
        <button type="button" className="text-accent-primary" onClick={onClose}>
          Close
        </button>
      </div>
    </ModalCore>
  );
}
