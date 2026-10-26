/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { Popover } from "@plane/propel/popover";
import { Tooltip } from "@plane/propel/tooltip";
import { ControlLink } from "@plane/ui";
import { findTotalDaysInRange } from "@plane/utils/datetime";
import { generateWorkItemLink } from "@plane/utils/work-item";
// components
import { SIDEBAR_WIDTH } from "@/components/gantt-chart/constants";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useIssueFilterPreferences } from "@/hooks/use-issue-filter-preferences";
import { useIssueListType } from "@/hooks/use-issue-list-type";
import { useRouteProjectStates } from "@/hooks/use-project-states";
import useIssuePeekOverviewRedirection from "@/hooks/use-issue-peek-overview-redirection";
import { usePlatformOS } from "@/hooks/use-platform-os";
import type { TIssue } from "@plane/types";
// local imports
import { IssuePreviewPopoverPanel } from "../preview-popover-panel";
import { getBlockViewDetails } from "../utils";
import type { GanttStoreType } from "./base-gantt-root";

type Props = {
  issue: TIssue;
  isEpic?: boolean;
};

export function IssueGanttBlock(props: Props) {
  const { issue, isEpic } = props;
  // router
  const { workspaceSlug: routerWorkspaceSlug } = useParams({ strict: false });
  const workspaceSlug = routerWorkspaceSlug?.toString();
  // hooks
  const { isMobile } = usePlatformOS();
  const { handleRedirection } = useIssuePeekOverviewRedirection(isEpic);

  // derived values
  const { states } = useRouteProjectStates(issue.project_id);
  const stateDetails = states?.find((state) => state.id === issue.state_id);

  const { blockStyle } = getBlockViewDetails(issue, stateDetails?.color ?? "");

  const handleIssuePeekOverview = () => handleRedirection(workspaceSlug, issue, isMobile);

  const duration = findTotalDaysInRange(issue.start_date, issue.target_date) || 0;

  return (
    <Popover>
      <Popover.Button
        delay={100}
        openOnHover
        className="w-full"
        render={
          // oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions
          <div
            id={`issue-${issue.id}`}
            className="space-between relative flex h-full w-full cursor-pointer items-center rounded-sm"
            style={blockStyle}
            onClick={handleIssuePeekOverview}
          >
            <div className="absolute top-0 left-0 h-full w-full bg-surface-1/50" />
            <div
              className="sticky w-auto flex-1 truncate overflow-hidden px-2.5 py-1 text-13 text-primary"
              style={{ left: `${SIDEBAR_WIDTH}px` }}
            >
              {issue.name}
            </div>
          </div>
        }
      />
      <IssuePreviewPopoverPanel issue={issue} />
    </Popover>
  );
}

// rendering issues on gantt sidebar
export function IssueGanttSidebarBlock(props: Props) {
  const { issue, isEpic = false } = props;
  // router
  const { workspaceSlug: routerWorkspaceSlug } = useParams({ strict: false });
  const workspaceSlug = routerWorkspaceSlug?.toString();
  const { isMobile } = usePlatformOS();
  const storeType = useIssueListType() as GanttStoreType;
  const { issueFilters } = useIssueFilterPreferences(storeType);
  const displayProperties = issueFilters?.displayProperties;
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;

  // handlers
  const { handleRedirection } = useIssuePeekOverviewRedirection(isEpic);

  // derived values
  const projectIdentifier = projectRows?.find((project) => project.id === issue.project_id)?.identifier ?? "";

  const handleIssuePeekOverview = (e: any) => {
    e.stopPropagation(true);
    e.preventDefault();
    handleRedirection(workspaceSlug, issue, isMobile);
  };

  const workItemLink = generateWorkItemLink({
    workspaceSlug,
    projectId: issue.project_id,
    issueId: issue.id,
    projectIdentifier,
    sequenceId: issue.sequence_id,
    isEpic,
  });

  return (
    <ControlLink
      id={`issue-${issue.id}`}
      href={workItemLink}
      onClick={handleIssuePeekOverview}
      className="line-clamp-1 w-full cursor-pointer text-13 text-primary"
      disabled={!!issue.tempId}
    >
      <div className="relative flex h-full w-full cursor-pointer items-center gap-2">
        {issue.project_id && (
          <IssueIdentifier
            workspaceSlug={workspaceSlug}
            issueId={issue.id}
            projectId={issue.project_id}
            size="xs"
            variant="tertiary"
            displayProperties={displayProperties}
          />
        )}
        <Tooltip tooltipContent={issue.name} isMobile={isMobile}>
          <span className="flex-grow truncate text-13 font-medium">{issue.name}</span>
        </Tooltip>
      </div>
    </ControlLink>
  );
}
