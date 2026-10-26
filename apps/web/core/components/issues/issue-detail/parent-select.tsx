/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "@tanstack/react-router";
import React from "react";

import { EditIcon, CloseIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import { cn } from "@plane/utils/common";
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
import type { TIssue } from "@plane/types";
import { useTranslation } from "@plane/i18n";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { WorkItemSelectionModal } from "../work-item-selection-modal";

type TIssueParentSelect = {
  className?: string;
  disabled?: boolean;
  issueId: string;
  projectId: string;
  workspaceSlug: string;
  parentIssue?: Partial<TIssue>;
  hasParentRelation: boolean;
  handleParentIssue: (issueId: string, parentProjectId: string) => Promise<void>;
  handleRemoveSubIssue: () => Promise<void>;
  workItemLink?: string;
};

type TParentIssueWithIdentity = Partial<TIssue> & { id: string; project_id: string };

function hasParentIdentity(issue: Partial<TIssue> | undefined): issue is TParentIssueWithIdentity {
  return Boolean(issue?.id && issue.project_id);
}

export const IssueParentSelect = function IssueParentSelect(props: TIssueParentSelect) {
  const {
    className = "",
    disabled = false,
    issueId,
    projectId,
    workspaceSlug,
    parentIssue,
    hasParentRelation,
    handleParentIssue,
    handleRemoveSubIssue,
    workItemLink,
  } = props;
  const { isParentIssueModalOpen, toggleParentIssueModal } = useIssueDetailUiState();

  return (
    <>
      <WorkItemSelectionModal
        workspaceSlug={workspaceSlug}
        filters={{ parent: true, issue_id: issueId }}
        projectId={projectId}
        issueId={issueId}
        isOpen={isParentIssueModalOpen === issueId}
        onClose={() => toggleParentIssueModal(null)}
        onSelect={(selected) => handleParentIssue(selected.id, selected.project_id)}
      />
      <div className={cn("group flex items-center gap-2", className)}>
        <ParentRelationDisplay
          parentIssue={parentIssue}
          hasParentRelation={hasParentRelation}
          workItemLink={workItemLink}
          disabled={disabled}
          openParentPicker={() => toggleParentIssueModal(issueId)}
        />
        <ParentActions
          disabled={disabled}
          hasParentRelation={hasParentRelation}
          parentIssue={parentIssue}
          handleRemoveSubIssue={handleRemoveSubIssue}
          openParentPicker={() => toggleParentIssueModal(issueId)}
        />
      </div>
    </>
  );
};

function ParentRelationDisplay({
  parentIssue,
  hasParentRelation,
  workItemLink,
  disabled,
  openParentPicker,
}: {
  parentIssue: Partial<TIssue> | undefined;
  hasParentRelation: boolean;
  workItemLink: string | undefined;
  disabled: boolean;
  openParentPicker: () => void;
}) {
  const { t } = useTranslation();
  const { isMobile } = usePlatformOS();
  const parent = hasParentIdentity(parentIssue) ? parentIssue : undefined;
  const projectRows = useWorkspaceProjectDetails().projects;
  const project = parent ? projectRows?.find((row) => row.id === parent.project_id) : undefined;
  if (!hasParentRelation)
    return (
      <button
        type="button"
        className="text-body-xs-medium text-placeholder"
        onClick={openParentPicker}
        disabled={disabled}
      >
        {t("issue.add.parent")}
      </button>
    );

  const label = parent?.name ?? t("common.parent");
  const identity =
    parent && parent.sequence_id !== undefined && project?.identifier ? (
      <IssueIdentifier
        projectId={parent.project_id}
        issueTypeId={parent.type_id}
        projectIdentifier={project.identifier}
        issueSequenceId={parent.sequence_id}
        size="xs"
        variant="secondary"
      />
    ) : (
      <span className="text-body-xs-medium text-secondary">{label}</span>
    );
  if (!workItemLink || !parent) return identity;

  return (
    <Tooltip tooltipHeading="Title" tooltipContent={label} isMobile={isMobile}>
      <Link
        to={workItemLink}
        target="_blank"
        rel="noopener noreferrer"
        className="flex items-center"
        onClick={(event) => event.stopPropagation()}
      >
        {identity}
      </Link>
    </Tooltip>
  );
}

function ParentActions({
  disabled,
  hasParentRelation,
  parentIssue,
  handleRemoveSubIssue,
  openParentPicker,
}: {
  disabled: boolean;
  hasParentRelation: boolean;
  parentIssue: Partial<TIssue> | undefined;
  handleRemoveSubIssue: () => Promise<void>;
  openParentPicker: () => void;
}) {
  const { t } = useTranslation();
  const { isMobile } = usePlatformOS();
  if (disabled) return null;
  return (
    <>
      {hasParentRelation && hasParentIdentity(parentIssue) && (
        <Tooltip tooltipContent={t("common.remove")} position="bottom" isMobile={isMobile}>
          <button
            type="button"
            aria-label={t("common.remove")}
            className="flex items-center"
            onClick={() => void handleRemoveSubIssue()}
          >
            <CloseIcon className="h-2.5 w-2.5 text-tertiary hover:text-danger-primary" />
          </button>
        </Tooltip>
      )}
      <button
        type="button"
        aria-label={t(hasParentRelation ? "common.actions.edit" : "issue.add.parent")}
        className="flex-shrink-0 p-1 text-placeholder opacity-0 group-hover:opacity-100"
        onClick={openParentPicker}
      >
        <EditIcon className="h-2.5 w-2.5 flex-shrink-0" />
      </button>
    </>
  );
}
