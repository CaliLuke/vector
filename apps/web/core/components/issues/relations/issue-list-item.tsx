/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

import { useTranslation } from "@plane/i18n";
import { LinkIcon, EditIcon, TrashIcon, CloseIcon } from "@plane/propel/icons";
// plane imports
import { Tooltip } from "@plane/propel/tooltip";
import type { TIssueRelationTypes } from "@plane/types";
import type { TIssue, TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
import { ControlLink, CustomMenu } from "@plane/ui";
import { generateWorkItemLink } from "@plane/utils/work-item";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useIssueRelations } from "@/hooks/use-issue-resources";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import useIssuePeekOverviewRedirection from "@/hooks/use-issue-peek-overview-redirection";
import { useWorkItemLinkCopy } from "@/hooks/use-work-item-link-copy";
import { usePlatformOS } from "@/hooks/use-platform-os";
// components
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// local imports
import { useRelationOperations } from "../issue-detail-widgets/relations/helper";
import { RelationIssueProperty } from "./properties";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  relationKey: TIssueRelationTypes;
  issue: TIssue;
  disabled: boolean;
  handleIssueCrudState: (
    key: "update" | "delete" | "removeRelation",
    issueId: string,
    issue?: TIssue | null,
    relationKey?: TIssueRelationTypes | null,
    relationIssueId?: string | null
  ) => void;
  issueServiceType?: TIssueServiceType;
};

export const RelationIssueListItem = function RelationIssueListItem(props: Props) {
  const {
    workspaceSlug,
    projectId: sourceProjectId,
    issueId,
    relationKey,
    issue,
    disabled = false,
    handleIssueCrudState,
    issueServiceType = EIssueServiceType.ISSUES,
  } = props;

  const { t } = useTranslation();

  // store hooks
  const { toggleCreateIssueModal, toggleDeleteIssueModal } = useIssueDetailUiState(issueServiceType);
  const { remove: removeRelation } = useIssueRelations({
    workspaceSlug,
    projectId: sourceProjectId,
    issueId,
    serviceType: issueServiceType,
  });
  const projectCatalogue = useWorkspaceProjectDetails(workspaceSlug);
  const { isMobile } = usePlatformOS();
  // derived values
  const { handleRedirection } = useIssuePeekOverviewRedirection(!!issue.is_epic);
  const issueOperations = useRelationOperations(
    workspaceSlug,
    issue.project_id ?? "",
    issue.id,
    issue.is_epic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES
  );
  const projectDetail = projectCatalogue.projects?.find((project) => project.id === issue.project_id);
  const relatedProjectId = issue.project_id;

  const workItemLink = generateWorkItemLink({
    workspaceSlug: workspaceSlug.toString(),
    projectId: issue.project_id,
    issueId: issue.id,
    projectIdentifier: projectDetail?.identifier,
    sequenceId: issue?.sequence_id,
    isEpic: issue?.is_epic,
  });

  const copyLink = useWorkItemLinkCopy({ workspaceSlug, projectId: relatedProjectId, issueId: issue.id }, workItemLink);
  if (!relatedProjectId) return <></>;

  // handlers
  const handleIssuePeekOverview = (relatedIssue: TIssue) => {
    if (relatedIssue.is_epic) {
      // open epics in new tab
      window.open(workItemLink, "_blank");
      return;
    }
    handleRedirection(workspaceSlug, relatedIssue, isMobile);
  };

  const handleEditIssue = (e: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
    e.stopPropagation();
    e.preventDefault();
    handleIssueCrudState("update", issue.id, { ...issue });
    toggleCreateIssueModal(true);
  };

  const handleDeleteIssue = (e: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
    e.stopPropagation();
    e.preventDefault();
    handleIssueCrudState("delete", issue.id, issue);
    toggleDeleteIssueModal(issue.id);
    handleIssueCrudState("removeRelation", issueId, issue, relationKey, issue.id);
  };

  const handleCopyIssueLink = (e: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
    e.stopPropagation();
    e.preventDefault();
    copyLink.copy();
  };

  const handleRemoveRelation = (e: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
    e.preventDefault();
    e.stopPropagation();
    void removeRelation(relationKey, issue.id);
  };

  return (
    <div
      key={issue.id}
      className="group relative flex h-full min-h-11 w-full items-center px-1.5 py-1 transition-all hover:bg-surface-2"
    >
      <span className="size-5 flex-shrink-0" />
      <ControlLink
        id={`issue-${issue.id}`}
        href={workItemLink}
        onClick={() => handleIssuePeekOverview(issue)}
        className="flex min-w-0 flex-1 cursor-pointer items-center gap-3"
      >
        <div className="flex-shrink-0">
          {projectDetail && (
            <IssueIdentifier
              projectId={projectDetail.id}
              issueTypeId={issue.type_id}
              projectIdentifier={projectDetail.identifier}
              issueSequenceId={issue.sequence_id}
              size="xs"
              variant="secondary"
            />
          )}
        </div>

        <Tooltip tooltipContent={issue.name} isMobile={isMobile}>
          <span className="w-0 flex-1 truncate text-13 text-primary">{issue.name}</span>
        </Tooltip>
      </ControlLink>
      <div className="flex-shrink-0 text-13">
        <RelationIssueProperty
          workspaceSlug={workspaceSlug}
          issueId={issue.id}
          issue={issue}
          disabled={disabled}
          issueOperations={issueOperations}
        />
      </div>
      <div className="flex-shrink-0 pl-2 text-13">
        <CustomMenu placement="bottom-end" ellipsis>
          {!disabled && (
            <CustomMenu.MenuItem onClick={handleEditIssue}>
              <div className="flex items-center gap-2">
                <EditIcon className="h-3.5 w-3.5" strokeWidth={2} />
                <span>{t("common.actions.edit")}</span>
              </div>
            </CustomMenu.MenuItem>
          )}

          <CustomMenu.MenuItem onClick={handleCopyIssueLink} disabled={copyLink.disabled}>
            <div className="flex items-center gap-2">
              <LinkIcon className="h-3.5 w-3.5" strokeWidth={2} />
              <span>{t("common.actions.copy_link")}</span>
            </div>
          </CustomMenu.MenuItem>

          {!disabled && (
            <CustomMenu.MenuItem onClick={handleRemoveRelation}>
              <div className="flex items-center gap-2">
                <CloseIcon className="h-3.5 w-3.5" strokeWidth={2} />
                <span>{t("common.actions.remove_relation")}</span>
              </div>
            </CustomMenu.MenuItem>
          )}

          {!disabled && (
            <CustomMenu.MenuItem onClick={handleDeleteIssue}>
              <div className="flex items-center gap-2">
                <TrashIcon className="h-3.5 w-3.5" strokeWidth={2} />
                <span>{t("common.actions.delete")}</span>
              </div>
            </CustomMenu.MenuItem>
          )}
        </CustomMenu>
      </div>
    </div>
  );
};
