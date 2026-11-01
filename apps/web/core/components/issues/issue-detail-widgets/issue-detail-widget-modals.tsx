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

function SubIssueModals({ workspaceSlug, projectId, issueId, issueServiceType }: Props) {
  const {
    isCreateIssueModalOpen,
    toggleCreateIssueModal,
    isSubIssuesModalOpen,
    toggleSubIssuesModal,
    setLastWidgetAction,
    issueCrudOperationState,
    setIssueCrudOperationState,
  } = useIssueDetailUiState(issueServiceType);
  const { create, existing } = issueCrudOperationState;
  const existingParentIssueId = existing.parentIssueId;
  const createProjectId = create.projectId ?? projectId;
  const existingProjectId = existing.projectId ?? projectId;
  const createOperations = useSubIssueOperations(
    workspaceSlug,
    createProjectId,
    create.parentIssueId ?? issueId,
    issueServiceType
  );
  const existingOperations = useSubIssueOperations(
    workspaceSlug,
    existingProjectId,
    existingParentIssueId ?? issueId,
    issueServiceType
  );
  const close = (kind: "create" | "existing") => {
    setIssueCrudOperationState({
      ...issueCrudOperationState,
      [kind]: { toggle: !issueCrudOperationState[kind].toggle, parentIssueId: null, projectId: null },
    });
    if (kind === "create") {
      toggleCreateIssueModal(false);
      setLastWidgetAction("sub-work-items");
    } else {
      setLastWidgetAction("sub-work-items");
      toggleSubIssuesModal(null);
    }
  };
  return (
    <>
      {create.toggle && create.parentIssueId && isCreateIssueModalOpen && (
        <CreateUpdateIssueModal
          isOpen={create.toggle}
          data={{ parent_id: create.parentIssueId, project_id: createProjectId }}
          onClose={() => close("create")}
          onSubmit={async (issue: TIssue) => {
            if (issue.parent_id) createOperations.refreshSubIssues(issue.parent_id);
          }}
          isProjectSelectionDisabled
        />
      )}
      {existing.toggle && existingParentIssueId && isSubIssuesModalOpen && (
        <ExistingIssuesListModal
          workspaceSlug={workspaceSlug}
          projectId={existingProjectId}
          isOpen={existing.toggle}
          handleClose={() => close("existing")}
          searchParams={{ sub_issue: true, issue_id: existingParentIssueId }}
          workspaceLevelToggle
          handleOnSubmit={(issues) =>
            existingOperations.addSubIssue(workspaceSlug, existingProjectId, existingParentIssueId, issues)
          }
        />
      )}
    </>
  );
}

function IssueLinkModal({ workspaceSlug, projectId, issueId, issueServiceType }: Props) {
  const { isIssueLinkModalOpen, toggleIssueLinkModal, setIssueLinkData, setLastWidgetAction } =
    useIssueDetailUiState(issueServiceType);
  const operations = useLinkOperations(workspaceSlug, projectId, issueId, issueServiceType);
  return (
    <IssueLinkCreateUpdateModal
      isModalOpen={isIssueLinkModalOpen}
      handleOnClose={() => {
        toggleIssueLinkModal(false);
        setLastWidgetAction("links");
        setIssueLinkData(null);
      }}
      linkOperations={operations}
      issueServiceType={issueServiceType}
    />
  );
}

function IssueRelationModal({ workspaceSlug, projectId, issueId, issueServiceType }: Props) {
  const { relationKey, isRelationModalOpen, setRelationKey, setLastWidgetAction, toggleRelationModal } =
    useIssueDetailUiState(issueServiceType);
  const { create } = useIssueRelations({ workspaceSlug, projectId, issueId, serviceType: issueServiceType });
  const submit = async (data: ISearchIssueResponse[]) => {
    if (relationKey && (await submitRelationSelection(data, relationKey, create))) toggleRelationModal(null, null);
  };
  return (
    <ExistingIssuesListModal
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      isOpen={isRelationModalOpen?.issueId === issueId && isRelationModalOpen?.relationType === relationKey}
      handleClose={() => {
        setRelationKey(null);
        toggleRelationModal(null, null);
        setLastWidgetAction("relations");
      }}
      searchParams={{ issue_relation: true, issue_id: issueId }}
      handleOnSubmit={submit}
      workspaceLevelToggle
    />
  );
}

export function IssueDetailWidgetModals(props: Props) {
  const hidden = props.hideWidgets ?? [];
  return (
    <>
      {!hidden.includes("links") && <IssueLinkModal {...props} />}
      {!hidden.includes("sub-work-items") && <SubIssueModals {...props} />}
      {!hidden.includes("relations") && <IssueRelationModal {...props} />}
    </>
  );
}
