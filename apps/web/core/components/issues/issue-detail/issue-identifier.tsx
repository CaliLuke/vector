/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { TIssueIdentifierProps, TIssueIdentifierBaseProps } from "@plane/types";
// hooks
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { WorkItemTypeBadge } from "@/components/work-item-types/work-item-type-badge";
import { IdentifierText } from "@/components/issues/issue-detail/identifier-text";

export const IssueIdentifier = function IssueIdentifier(props: TIssueIdentifierProps) {
  const isUsingStoreData = "issueId" in props;
  const projectCatalogue = useWorkspaceProjectDetails(props.workspaceSlug, isUsingStoreData);
  const projectRows = projectCatalogue.projects;
  const detail = useWorkItemDetail(
    props.workspaceSlug,
    props.projectId,
    isUsingStoreData ? props.issueId : undefined,
    undefined,
    isUsingStoreData
  );
  const issue = isUsingStoreData ? detail.issue : undefined;
  const identity = isUsingStoreData
    ? {
        projectIdentifier: projectRows?.find((project) => project.id === props.projectId)?.identifier ?? "",
        issueSequenceId: issue?.sequence_id,
        issueTypeId: issue?.type_id,
        loaded: Boolean(issue),
      }
    : {
        projectIdentifier: props.projectIdentifier,
        issueSequenceId: props.issueSequenceId,
        issueTypeId: props.issueTypeId,
        loaded: true,
      };
  return <IdentifierContents {...props} {...identity} />;
};

function IdentifierContents({
  workspaceSlug,
  projectId,
  variant,
  size,
  displayProperties,
  enableClickToCopyIdentifier = false,
  projectIdentifier,
  issueSequenceId,
  issueTypeId,
  loaded,
}: TIssueIdentifierBaseProps & {
  projectIdentifier?: string;
  issueSequenceId?: string | number;
  issueTypeId?: string | null;
  loaded: boolean;
}) {
  const showKey = (displayProperties?.key ?? true) && hasCompleteIdentifier(loaded, projectIdentifier, issueSequenceId);
  const showType = Boolean(displayProperties?.issue_type && workspaceSlug && loaded);
  if (!showKey && !showType) return null;
  return (
    <div className="flex shrink-0 items-center space-x-2">
      {showType && workspaceSlug && (
        <WorkItemTypeBadge workspaceSlug={workspaceSlug} projectId={projectId} typeId={issueTypeId} />
      )}
      {showKey && (
        <IdentifierText
          identifier={`${projectIdentifier}-${issueSequenceId}`}
          enableClickToCopyIdentifier={enableClickToCopyIdentifier}
          variant={variant}
          size={size}
        />
      )}
    </div>
  );
}

function hasCompleteIdentifier(
  loaded: boolean,
  projectIdentifier: string | undefined,
  issueSequenceId: string | number | undefined
) {
  return loaded && Boolean(projectIdentifier) && issueSequenceId !== undefined && issueSequenceId !== null;
}
