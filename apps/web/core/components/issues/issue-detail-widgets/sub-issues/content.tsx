/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState, useCallback } from "react";

import type { TIssue, TIssueServiceType } from "@plane/types";
import { EIssueServiceType, EIssuesStoreType } from "@plane/types";
// components
import { DeleteIssueModal } from "@/components/issues/delete-issue-modal";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useWorkItemSubItems } from "@/hooks/use-work-item-detail";
// local imports
import { CreateUpdateIssueModal } from "../../issue-modal/modal";
import { useSubIssueOperations } from "./helper";
import { SubIssuesListRoot } from "./issues-list/root";

type Props = {
  workspaceSlug: string;
  projectId: string;
  parentIssueId: string;
  disabled: boolean;
  issueServiceType?: TIssueServiceType;
};

type TIssueCrudState = { toggle: boolean; parentIssueId: string | undefined; issue: TIssue | undefined };

export const SubIssuesCollapsibleContent = function SubIssuesCollapsibleContent(props: Props) {
  const { workspaceSlug, projectId, parentIssueId, disabled, issueServiceType = EIssueServiceType.ISSUES } = props;
  // state
  const [issueCrudState, setIssueCrudState] = useState<{
    create: TIssueCrudState;
    existing: TIssueCrudState;
    update: TIssueCrudState;
    delete: TIssueCrudState;
  }>({
    create: {
      toggle: false,
      parentIssueId: undefined,
      issue: undefined,
    },
    existing: {
      toggle: false,
      parentIssueId: undefined,
      issue: undefined,
    },
    update: {
      toggle: false,
      parentIssueId: undefined,
      issue: undefined,
    },
    delete: {
      toggle: false,
      parentIssueId: undefined,
      issue: undefined,
    },
  });
  // store hooks
  const { toggleCreateIssueModal, toggleDeleteIssueModal } = useIssueDetailUiState(issueServiceType);

  // helpers
  const subIssueOperations = useSubIssueOperations(workspaceSlug, projectId, parentIssueId, issueServiceType);
  const subItems = useWorkItemSubItems(workspaceSlug, projectId, parentIssueId, issueServiceType);

  // handler
  const handleIssueCrudState = useCallback(
    (key: "create" | "existing" | "update" | "delete", _parentIssueId: string | null, issue: TIssue | null = null) => {
      setIssueCrudState({
        ...issueCrudState,
        [key]: {
          toggle: !issueCrudState[key].toggle,
          parentIssueId: _parentIssueId,
          issue,
        },
      });
    },
    [issueCrudState]
  );

  // render conditions
  const { issue: issueToDelete, parentIssueId: deleteParentId, toggle: isDeleteOpen } = issueCrudState.delete;
  const deleteProjectId = issueToDelete?.project_id;

  const shouldRenderUpdateIssueModal = issueCrudState?.update?.toggle && issueCrudState?.update?.issue;

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
          handleIssueCrudState={handleIssueCrudState}
          subIssueOperations={subIssueOperations}
          issueServiceType={issueServiceType}
        />
      )}

      {isDeleteOpen && issueToDelete?.id && deleteParentId && deleteProjectId && (
        <DeleteIssueModal
          isOpen={isDeleteOpen}
          handleClose={() => {
            handleIssueCrudState("delete", null, null);
            toggleDeleteIssueModal(null);
          }}
          data={issueToDelete}
          onSubmit={() =>
            subIssueOperations.deleteSubIssue(workspaceSlug, deleteProjectId, deleteParentId, issueToDelete.id)
          }
          isSubIssue
        />
      )}

      {shouldRenderUpdateIssueModal && (
        <CreateUpdateIssueModal
          isOpen={issueCrudState?.update?.toggle}
          onClose={() => {
            handleIssueCrudState("update", null, null);
            toggleCreateIssueModal(false);
          }}
          data={issueCrudState?.update?.issue ?? undefined}
          onSubmit={async () => {
            subIssueOperations.refreshSubIssues(issueCrudState.update.parentIssueId ?? parentIssueId);
          }}
        />
      )}
    </>
  );
};
