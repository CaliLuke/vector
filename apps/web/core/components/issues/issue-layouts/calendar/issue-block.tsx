/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useRef, forwardRef } from "react";

import { MoreHorizontal } from "lucide-react";
// plane imports
import { Popover } from "@plane/propel/popover";
import type { TIssue } from "@plane/types";
import { ControlLink } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { generateWorkItemLink } from "@plane/utils/work-item";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useRouteProjectStates } from "@/hooks/use-project-states";
import { useIssueListType } from "@/hooks/use-issue-list-type";
import { useIssueFilterPreferences } from "@/hooks/use-issue-filter-preferences";
import useIssuePeekOverviewRedirection from "@/hooks/use-issue-peek-overview-redirection";
import { usePlatformOS } from "@/hooks/use-platform-os";
// components
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// local components
import { IssuePreviewPopoverPanel } from "../preview-popover-panel";
import type { TRenderQuickActions } from "../list/list-view-types";

type Props = {
  issue: TIssue;
  quickActions: TRenderQuickActions;
  isDragging?: boolean;
  isEpic?: boolean;
};

export const CalendarIssueBlock = forwardRef(function CalendarIssueBlock(
  props: Props,
  ref: React.ForwardedRef<HTMLElement>
) {
  const { issue, quickActions, isDragging = false, isEpic = false } = props;
  // states
  // refs
  const blockRef = useRef(null);
  // hooks
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });

  const { getIsIssuePeeked } = useIssueDetailUiState();
  const { handleRedirection } = useIssuePeekOverviewRedirection(isEpic);
  const { isMobile } = usePlatformOS();
  const storeType = useIssueListType();
  const { issueFilters: displayPreferences } = useIssueFilterPreferences(storeType);
  const displayProperties = displayPreferences?.displayProperties;
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;

  const { states } = useRouteProjectStates(issue.project_id);
  const stateColor = states?.find((state) => state.id === issue.state_id)?.color || "";
  const projectIdentifier = projectRows?.find((project) => project.id === issue.project_id)?.identifier ?? "";

  // handlers
  const handleIssuePeekOverview = (peekIssue: TIssue) => handleRedirection(workspaceSlug, peekIssue, isMobile);

  const customActionButton = (
    <span className="block w-full rounded-sm p-1 text-secondary hover:bg-layer-1 [button[aria-expanded=true]_&]:bg-layer-1-active [button[aria-expanded=true]_&]:text-primary">
      <MoreHorizontal className="h-3.5 w-3.5" />
    </span>
  );

  const workItemLink = generateWorkItemLink({
    workspaceSlug,
    projectId: issue.project_id,
    issueId: issue.id,
    projectIdentifier,
    sequenceId: issue.sequence_id,
    isEpic,
    isArchived: !!issue.archived_at,
  });

  return (
    <Popover>
      <Popover.Button
        delay={100}
        openOnHover
        className="w-full"
        render={
          <ControlLink
            id={`issue-${issue.id}`}
            href={workItemLink}
            onClick={() => handleIssuePeekOverview(issue)}
            className="block w-full rounded-sm border-b border-subtle text-13 text-primary hover:border-subtle-1 md:border-[1px]"
            disabled={!!issue.tempId || isMobile}
            ref={ref}
          >
            <>
              {issue.tempId !== undefined && (
                <div className="absolute top-0 left-0 z-[99999] h-full w-full animate-pulse bg-surface-1/20" />
              )}

              <div
                ref={blockRef}
                className={cn(
                  "group/calendar-block flex h-10 w-full items-center justify-between gap-1.5 rounded-sm px-4 py-1.5 md:h-8 md:px-1",
                  {
                    "border-accent-strong bg-surface-2 shadow-raised-200": isDragging,
                    "bg-surface-1 hover:bg-surface-2": !isDragging,
                    "border border-accent-strong hover:border-accent-strong": getIsIssuePeeked(issue.id),
                  }
                )}
              >
                <div className="flex h-full items-center gap-1.5 truncate">
                  <span
                    className="h-full w-0.5 flex-shrink-0 rounded-sm"
                    style={{
                      backgroundColor: stateColor,
                    }}
                  />
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
                  <div className="truncate text-13 font-medium md:text-11 md:font-regular">{issue.name}</div>
                </div>
                {/* Wrapper exists only to stop clicks reaching the ControlLink; the
                      quick-action menu inside carries its own interactive semantics. */}
                <div
                  role="presentation"
                  className={cn("size-5 flex-shrink-0", {
                    "opacity-0 group-hover/calendar-block:opacity-100 focus-within:opacity-100 has-[button[aria-expanded=true]]:opacity-100":
                      !isMobile,
                  })}
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                  }}
                >
                  {quickActions({
                    issue,
                    parentRef: blockRef,
                    customActionButton,
                    placement: "bottom-end",
                  })}
                </div>
              </div>
            </>
          </ControlLink>
        }
      />
      <IssuePreviewPopoverPanel issue={issue} />
    </Popover>
  );
});

CalendarIssueBlock.displayName = "CalendarIssueBlock";
