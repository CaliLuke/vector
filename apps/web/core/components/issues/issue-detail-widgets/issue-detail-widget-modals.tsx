/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

import { submitRelationSelection } from "@/components/issues/issue-detail/submit-relation-selection";
import type { ISearchIssueResponse, TIssue, TIssueServiceType, TWorkItemWidgets } from "@plane/types";
// components
import { ExistingIssuesListModal } from "@/components/core/modals/existing-issues-list-modal";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useIssueRelations } from "@/hooks/use-issue-resources";
// local imports
import { IssueLinkCreateUpdateModal } from "../issue-detail/links/create-update-link-modal";
// helpers
import { CreateUpdateIssueModal } from "../issue-modal/modal";
import { useLinkOperations } from "@/hooks/use-issue-link-operations";
import { useSubIssueOperations } from "./sub-issues/helper";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueServiceType: TIssueServiceType;
  hideWidgets?: TWorkItemWidgets[];
};

export const IssueDetailWidgetModals = function IssueDetailWidgetModals(props: Props) {
  const { workspaceSlug, projectId, issueId, issueServiceType, hideWidgets } = props;
  // store hooks
  const {
    isIssueLinkModalOpen,
    toggleIssueLinkModal: toggleIssueLinkModalStore,
    setIssueLinkData,
    isCreateIssueModalOpen,
    toggleCreateIssueModal,
    isSubIssuesModalOpen,
    toggleSubIssuesModal,
    relationKey,
    isRelationModalOpen,
    setRelationKey,
    setLastWidgetAction,
    toggleRelationModal,
    issueCrudOperationState,
    setIssueCrudOperationState,
  } = useIssueDetailUiState(issueServiceType);

  // helper hooks
  const subIssueOperations = useSubIssueOperations(workspaceSlug, projectId, issueId, issueServiceType);
  const handleLinkOperations = useLinkOperations(workspaceSlug, projectId, issueId, issueServiceType);
  const { create: createRelation } = useIssueRelations({
    workspaceSlug,
    projectId,
    issueId,
    serviceType: issueServiceType,
  });

  // handlers
  const handleIssueCrudState = (
    key: "create" | "existing",
    _parentIssueId: string | null,
    issue: TIssue | null = null
  ) => {
    setIssueCrudOperationState({
      ...issueCrudOperationState,
      [key]: {
        toggle: !issueCrudOperationState[key].toggle,
        parentIssueId: _parentIssueId,
        issue: issue,
      },
    });
  };

  const handleExistingIssuesModalClose = () => {
    handleIssueCrudState("existing", null, null);
    setLastWidgetAction("sub-work-items");
    toggleSubIssuesModal(null);
  };

  const handleExistingIssuesModalOnSubmit = async (_issue: ISearchIssueResponse[]) =>
    subIssueOperations.addSubIssue(
      workspaceSlug,
      projectId,
      issueId,
      _issue.map((issue) => issue.id)
    );

  const handleCreateUpdateModalClose = () => {
    handleIssueCrudState("create", null, null);
    toggleCreateIssueModal(false);
    setLastWidgetAction("sub-work-items");
  };

  const handleCreateUpdateModalOnSubmit = async (_issue: TIssue) => {
    if (_issue.parent_id) {
      subIssueOperations.refreshSubIssues(_issue.parent_id);
    }
  };

  const handleIssueLinkModalOnClose = () => {
    toggleIssueLinkModalStore(false);
    setLastWidgetAction("links");
    setIssueLinkData(null);
  };

  const handleRelationOnClose = () => {
    setRelationKey(null);
    toggleRelationModal(null, null);
    setLastWidgetAction("relations");
  };

  const handleExistingIssueModalOnSubmit = async (data: ISearchIssueResponse[]) => {
    if (!relationKey) return;
    if (await submitRelationSelection(data, relationKey, createRelation)) {
      toggleRelationModal(null, null);
    }
  };

  // helpers
  const createUpdateModalData: Partial<TIssue> = {
    parent_id: issueCrudOperationState?.create?.parentIssueId,
    project_id: projectId,
  };

  const existingIssuesModalSearchParams = {
    sub_issue: true,
    issue_id: issueCrudOperationState?.existing?.parentIssueId,
  };

  // render conditions
  const shouldRenderExistingIssuesModal =
    !hideWidgets?.includes("sub-work-items") &&
    issueCrudOperationState?.existing?.toggle &&
    issueCrudOperationState?.existing?.parentIssueId &&
    isSubIssuesModalOpen;

  const shouldRenderCreateUpdateModal =
    !hideWidgets?.includes("sub-work-items") &&
    issueCrudOperationState?.create?.toggle &&
    issueCrudOperationState?.create?.parentIssueId &&
    isCreateIssueModalOpen;

  return (
    <>
      {!hideWidgets?.includes("links") && (
        <IssueLinkCreateUpdateModal
          isModalOpen={isIssueLinkModalOpen}
          handleOnClose={handleIssueLinkModalOnClose}
          linkOperations={handleLinkOperations}
          issueServiceType={issueServiceType}
        />
      )}

      {shouldRenderCreateUpdateModal && (
        <CreateUpdateIssueModal
          isOpen={issueCrudOperationState?.create?.toggle}
          data={createUpdateModalData}
          onClose={handleCreateUpdateModalClose}
          onSubmit={handleCreateUpdateModalOnSubmit}
          isProjectSelectionDisabled
        />
      )}

      {shouldRenderExistingIssuesModal && (
        <ExistingIssuesListModal
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          isOpen={issueCrudOperationState?.existing?.toggle}
          handleClose={handleExistingIssuesModalClose}
          searchParams={existingIssuesModalSearchParams}
          workspaceLevelToggle
          handleOnSubmit={handleExistingIssuesModalOnSubmit}
        />
      )}

      {!hideWidgets?.includes("relations") && (
        <ExistingIssuesListModal
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          isOpen={isRelationModalOpen?.issueId === issueId && isRelationModalOpen?.relationType === relationKey}
          handleClose={handleRelationOnClose}
          searchParams={{ issue_relation: true, issue_id: issueId }}
          handleOnSubmit={handleExistingIssueModalOnSubmit}
          workspaceLevelToggle
        />
      )}
    </>
  );
};
