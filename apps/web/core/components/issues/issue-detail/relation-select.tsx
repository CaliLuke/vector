/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "@tanstack/react-router";
import React from "react";

import { EditIcon, CloseIcon } from "@plane/propel/icons";
// Vector
import { submitRelationSelection } from "@/components/issues/issue-detail/submit-relation-selection";
import { IssueResourceRetryButton } from "@/components/issues/issue-detail/resource-retry-button";
import { Tooltip } from "@plane/propel/tooltip";
import type { ISearchIssueResponse } from "@plane/types";
import { cn } from "@plane/utils/common";
import { generateWorkItemLink } from "@plane/utils/work-item";
// components
import { ExistingIssuesListModal } from "@/components/core/modals/existing-issues-list-modal";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { useIssueRelations } from "@/hooks/use-issue-resources";
// Vector web imports
import { useTimeLineRelationOptions } from "@/components/relations";
import type { TIssueRelationTypes } from "@plane/types";
import type { TRelationObject } from "../issue-detail-widgets/relations";

type TIssueRelationSelect = {
  className?: string;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  relationKey: TIssueRelationTypes;
  disabled?: boolean;
};

export const IssueRelationSelect = function IssueRelationSelect(props: TIssueRelationSelect) {
  const { className = "", workspaceSlug, projectId, issueId, relationKey, disabled = false } = props;
  // hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const { isRelationModalOpen, toggleRelationModal } = useIssueDetailUiState();
  const {
    relations,
    query,
    isError,
    retry,
    create: createRelation,
    remove: removeRelation,
  } = useIssueRelations({
    workspaceSlug,
    projectId,
    issueId,
  });
  const { isMobile } = usePlatformOS();
  const relationIssues = relations?.[relationKey];
  const relationIssueIds = relationIssues?.map((issue) => issue.id);
  const ISSUE_RELATION_OPTIONS = useTimeLineRelationOptions();

  const onSubmit = async (data: ISearchIssueResponse[]) => {
    if (await submitRelationSelection(data, relationKey, createRelation)) {
      toggleRelationModal(null, null);
    }
  };

  if (query.isError || isError)
    return <IssueResourceRetryButton onRetry={() => void (isError ? retry() : query.refetch())} />;
  if (!relationIssueIds) return null;

  const isRelationKeyModalActive =
    isRelationModalOpen?.relationType === relationKey && isRelationModalOpen?.issueId === issueId;

  const currRelationOption: TRelationObject | undefined = ISSUE_RELATION_OPTIONS[relationKey];

  return (
    <>
      <ExistingIssuesListModal
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        isOpen={isRelationKeyModalActive}
        handleClose={() => toggleRelationModal(null, null)}
        searchParams={{ issue_relation: true, issue_id: issueId }}
        handleOnSubmit={onSubmit}
        workspaceLevelToggle
      />

      <button
        type="button"
        className={cn(
          "group flex items-center gap-2 rounded-sm px-2 py-0.5 outline-none",
          {
            "cursor-not-allowed": disabled,
            "hover:bg-layer-1": !disabled,
            "bg-layer-1": isRelationKeyModalActive,
          },
          className
        )}
        onClick={() => toggleRelationModal(issueId, relationKey)}
        disabled={disabled}
      >
        <div className="flex w-full items-start justify-between">
          {relationIssueIds.length > 0 ? (
            <div className="flex flex-wrap items-center gap-2 py-0.5">
              {relationIssueIds.map((relationIssueId) => {
                const currentIssue = relationIssues?.find((relatedIssue) => relatedIssue.id === relationIssueId);
                if (!currentIssue) return;

                const projectDetails = projectRows?.find((project) => project.id === currentIssue.project_id);

                return (
                  <div
                    key={relationIssueId}
                    className={`group flex items-center gap-1 rounded-sm px-1.5 pt-1 pb-1 leading-3 hover:bg-surface-2 ${currRelationOption?.className}`}
                  >
                    <Tooltip tooltipHeading="Title" tooltipContent={currentIssue.name} isMobile={isMobile}>
                      <Link
                        to={generateWorkItemLink({
                          workspaceSlug,
                          projectId: projectDetails?.id,
                          issueId: currentIssue.id,
                          projectIdentifier: projectDetails?.identifier,
                          sequenceId: currentIssue?.sequence_id,
                        })}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-caption-sm-medium"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {`${projectDetails?.identifier}-${currentIssue?.sequence_id}`}
                      </Link>
                    </Tooltip>
                    {!disabled && (
                      <Tooltip tooltipContent="Remove" position="bottom" isMobile={isMobile}>
                        {/* eslint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
                        <span
                          onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            void removeRelation(relationKey, relationIssueId);
                          }}
                        >
                          <CloseIcon className="h-2.5 w-2.5 text-tertiary hover:text-danger-primary" />
                        </span>
                      </Tooltip>
                    )}
                  </div>
                );
              })}
            </div>
          ) : (
            <span className="text-body-xs-regular text-placeholder">{currRelationOption?.placeholder}</span>
          )}
          {!disabled && (
            <span
              className={cn("flex-shrink-0 p-1 opacity-0 group-hover:opacity-100", {
                "text-placeholder": relationIssueIds.length === 0,
              })}
            >
              <EditIcon className="h-2.5 w-2.5 flex-shrink-0" />
            </span>
          )}
        </div>
      </button>
    </>
  );
};
