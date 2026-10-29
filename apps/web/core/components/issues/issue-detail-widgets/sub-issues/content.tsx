/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
import type { TIssueServiceType } from "@plane/types";
import { EIssueServiceType, EIssuesStoreType } from "@plane/types";
import { DeleteIssueModal } from "@/components/issues/delete-issue-modal";
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useWorkItemSubItems } from "@/hooks/use-work-item-detail";
import { CreateUpdateIssueModal } from "../../issue-modal/modal";
import { useSubIssueOperations } from "./helper";
import { SubIssuesListRoot } from "./issues-list/root";
import type { SubIssueAction } from "./types";

type Props = {
  workspaceSlug: string;
  projectId: string;
  parentIssueId: string;
  disabled: boolean;
  issueServiceType?: TIssueServiceType;
};

export function SubIssuesCollapsibleContent(props: Props) {
  const { workspaceSlug, projectId, parentIssueId, disabled, issueServiceType = EIssueServiceType.ISSUES } = props;
  const [action, setAction] = useState<SubIssueAction>();
  const subItems = useWorkItemSubItems(workspaceSlug, projectId, parentIssueId, issueServiceType);
  return (
    <>
      {subItems.isSuccess && (
        <SubIssuesListRoot
          storeType={EIssuesStoreType.PROJECT}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          parentIssueId={parentIssueId}
          rootIssueId={parentIssueId}
          spacingLeft={6}
          canEdit={!disabled}
          onAction={setAction}
          issueServiceType={issueServiceType}
        />
      )}
      {action && (
        <SubIssueActionModal
          key={JSON.stringify([action.kind, action.parent, action.issue.project_id, action.issue.id])}
          workspaceSlug={workspaceSlug}
          action={action}
          onClose={() => setAction(undefined)}
        />
      )}
    </>
  );
}

function SubIssueActionModal({
  workspaceSlug,
  action,
  onClose,
}: {
  workspaceSlug: string;
  action: SubIssueAction;
  onClose: () => void;
}) {
  const { issue, parent } = action;
  const operations = useSubIssueOperations(workspaceSlug, parent.projectId, parent.issueId, parent.serviceType);
  const { toggleCreateIssueModal, toggleDeleteIssueModal } = useIssueDetailUiState(parent.serviceType);
  if (action.kind === "delete") {
    const projectId = issue.project_id;
    if (!projectId || !issue.id) return null;
    return (
      <DeleteIssueModal
        isOpen
        handleClose={() => {
          onClose();
          toggleDeleteIssueModal(null);
        }}
        data={issue}
        onSubmit={() => operations.deleteSubIssue(workspaceSlug, projectId, parent.issueId, issue.id)}
        isSubIssue
      />
    );
  }
  return (
    <CreateUpdateIssueModal
      isOpen
      onClose={() => {
        onClose();
        toggleCreateIssueModal(false);
      }}
      data={issue}
      onSubmit={async () => {
        operations.refreshSubIssues(parent.issueId);
      }}
    />
  );
}
