/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane types
import { PriorityIcon, StateGroupIcon, WorkItemsIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { TActivityEntityData, TIssueEntityData } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// plane ui
import { calculateTimeAgo } from "@plane/utils/datetime";
import { generateWorkItemLink } from "@plane/utils/work-item";
// components
import { ListItem } from "@/components/core/list";
import { ButtonAvatars } from "@/components/dropdowns/member/avatar";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkspaceStates } from "@/hooks/use-project-states";

type BlockProps = {
  activity: TActivityEntityData;
  parentRef: React.RefObject<HTMLDivElement | null>;
  workspaceSlug: string;
};
export const RecentIssue = function RecentIssue(props: BlockProps) {
  const { activity, parentRef, workspaceSlug } = props;
  // hooks
  const { getStateById } = useWorkspaceStates(workspaceSlug);
  const { setPeekIssue } = useIssueDetailUiState();
  const { setPeekIssue: setPeekEpic } = useIssueDetailUiState(EIssueServiceType.EPICS);
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  // derived values
  const issueDetails: TIssueEntityData = activity.entity_data as TIssueEntityData;
  const projectIdentifier = projectRows?.find((project) => project.id === issueDetails?.project_id)?.identifier ?? "";

  if (!issueDetails) return <></>;

  const state = getStateById(issueDetails?.state);

  const workItemLink = generateWorkItemLink({
    workspaceSlug: workspaceSlug?.toString(),
    projectId: issueDetails?.project_id,
    issueId: issueDetails?.id,
    projectIdentifier,
    sequenceId: issueDetails?.sequence_id,
    isEpic: issueDetails?.is_epic,
  });

  const handlePeekOverview = (e: React.MouseEvent<HTMLAnchorElement>) => {
    e.preventDefault();
    e.stopPropagation();
    const peekDetails = {
      workspaceSlug,
      projectId: issueDetails?.project_id,
      issueId: activity.entity_data.id,
    };
    if (issueDetails?.is_epic) setPeekEpic(peekDetails);
    else setPeekIssue(peekDetails);
  };

  return (
    <ListItem
      key={activity.id}
      id={`issue-${issueDetails?.id}`}
      itemLink={workItemLink}
      title={issueDetails?.name}
      prependTitleElement={<RecentIssueTitle issueDetails={issueDetails} />}
      appendTitleElement={
        <div className="flex-shrink-0 text-11 font-medium text-placeholder">
          {calculateTimeAgo(activity.visited_at)}
        </div>
      }
      quickActionElement={
        <RecentIssueActions issueDetails={issueDetails} state={state} workspaceSlug={workspaceSlug} />
      }
      parentRef={parentRef}
      disableLink={false}
      className="my-auto border-none !px-2 py-3"
      itemClassName="my-auto"
      onItemClick={handlePeekOverview}
      preventDefaultProgress
    />
  );
};

function RecentIssueTitle({ issueDetails }: { issueDetails: TIssueEntityData }) {
  return (
    <div className="flex flex-shrink-0 items-center gap-2">
      {issueDetails.type ? (
        <IssueIdentifier
          size="lg"
          issueTypeId={issueDetails.type}
          projectId={issueDetails.project_id || ""}
          projectIdentifier={issueDetails.project_identifier || ""}
          issueSequenceId={issueDetails.sequence_id || ""}
          variant="tertiary"
        />
      ) : (
        <div className="flex items-center justify-center gap-2">
          <div className="grid size-8 flex-shrink-0 place-items-center rounded-sm bg-layer-2">
            <WorkItemsIcon className="size-4 text-tertiary" />
          </div>
          <div className="text-13 font-medium whitespace-nowrap text-placeholder">
            {issueDetails.project_identifier}-{issueDetails.sequence_id}
          </div>
        </div>
      )}
    </div>
  );
}

function RecentIssueActions({
  issueDetails,
  state,
  workspaceSlug,
}: {
  issueDetails: TIssueEntityData;
  state: ReturnType<ReturnType<typeof useWorkspaceStates>["getStateById"]>;
  workspaceSlug: string;
}) {
  const hasAssignees = Boolean(issueDetails.assignees?.length);
  return (
    <div className="flex gap-4">
      <Tooltip tooltipHeading="State" tooltipContent={state?.name ?? "State"}>
        <div>
          <StateGroupIcon
            stateGroup={state?.group ?? "backlog"}
            color={state?.color}
            className="my-auto h-4 w-4"
            percentage={state?.order}
          />
        </div>
      </Tooltip>
      <Tooltip tooltipHeading="Priority" tooltipContent={issueDetails.priority ?? "Priority"}>
        <div>
          <PriorityIcon priority={issueDetails.priority} withContainer size={12} />
        </div>
      </Tooltip>
      {hasAssignees && (
        <div className="h-5">
          <MemberDropdown
            workspaceSlug={workspaceSlug}
            projectId={issueDetails.project_id}
            value={issueDetails.assignees}
            onChange={() => {}}
            disabled
            multiple
            button={
              <ButtonAvatars showTooltip={false} userIds={issueDetails.assignees} workspaceSlug={workspaceSlug} />
            }
            buttonVariant="transparent-without-text"
            buttonClassName="hover:bg-transparent px-0"
            showTooltip={false}
            placeholder="Assignees"
            optionsClassName="z-10"
            tooltipContent=""
          />
        </div>
      )}
    </div>
  );
}
