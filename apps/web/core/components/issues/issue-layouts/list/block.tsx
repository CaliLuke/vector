/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import type { Dispatch, MouseEvent, SetStateAction } from "react";
import { useEffect, useRef } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { draggable } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";

import { ChevronRightIcon } from "@plane/propel/icons";
// types
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Tooltip } from "@plane/propel/tooltip";
import type { TIssue, IIssueDisplayProperties, TIssueMap } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// ui
import { Spinner, ControlLink, Row } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { generateWorkItemLink } from "@plane/utils/work-item";
// components
import { MultipleSelectEntityAction } from "@/components/core/multiple-select";
import { IssueProperties } from "@/components/issues/issue-layouts/properties";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// hooks
import { useSidebarCollapsed } from "@/providers/sidebar-state";
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useRouteProject, useWorkspaceProjectDetails } from "@/hooks/use-projects";
import type { TSelectionHelper } from "@/hooks/use-multiple-select";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { calculateIdentifierWidth } from "../utils";
import type { TRenderQuickActions } from "./list-view-types";

interface IssueBlockProps {
  issueId: string;
  issuesMap: TIssueMap;
  groupId: string;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<TIssue>) => Promise<unknown>) | undefined;
  quickActions: TRenderQuickActions;
  displayProperties: IIssueDisplayProperties | undefined;
  canEditProperties: (projectId: string | undefined) => boolean;
  nestingLevel: number;
  spacingLeft?: number;
  isExpanded: boolean;
  setExpanded: Dispatch<SetStateAction<boolean>>;
  selectionHelpers: TSelectionHelper;
  isCurrentBlockDragging: boolean;
  setIsCurrentBlockDragging: React.Dispatch<React.SetStateAction<boolean>>;
  canDrag: boolean;
  isEpic?: boolean;
}

export const IssueBlock = function IssueBlock(props: IssueBlockProps) {
  const {
    issuesMap,
    issueId,
    groupId,
    updateIssue,
    quickActions,
    displayProperties,
    canEditProperties,
    nestingLevel,
    spacingLeft = 14,
    isExpanded,
    setExpanded,
    selectionHelpers,
    isCurrentBlockDragging,
    setIsCurrentBlockDragging,
    canDrag,
    isEpic = false,
  } = props;
  // ref
  const issueRef = useRef<HTMLDivElement | null>(null);
  // router
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // hooks
  const isSidebarCollapsed = useSidebarCollapsed();
  const routeProjectQuery = useRouteProject();
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const currentProjectNextSequenceId = routeProjectQuery.project?.next_work_item_sequence;
  const { getIsIssuePeeked, peekIssue, setPeekIssue } = useIssueDetailUiState(
    isEpic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES
  );

  const handleIssuePeekOverview = (issue: TIssue) =>
    workspaceSlug &&
    issue &&
    issue.project_id &&
    issue.id &&
    !getIsIssuePeeked(issue.id) &&
    setPeekIssue({
      workspaceSlug,
      projectId: issue.project_id,
      issueId: issue.id,
      nestingLevel: nestingLevel,
      isArchived: !!issue.archived_at,
    });

  // derived values
  const issue = issuesMap[issueId];
  const subIssuesCount = issue?.sub_issues_count ?? 0;
  const canEditIssueProperties = canEditProperties(issue?.project_id ?? undefined);
  const isDraggingAllowed = canDrag && canEditIssueProperties;

  const { isMobile } = usePlatformOS();

  useEffect(() => {
    const element = issueRef.current;

    if (!element) return undefined;

    return combine(
      draggable({
        element,
        canDrag: () => isDraggingAllowed,
        getInitialData: () => ({ id: issueId, type: "ISSUE", groupId }),
        onDragStart: () => {
          setIsCurrentBlockDragging(true);
        },
        onDrop: () => {
          setIsCurrentBlockDragging(false);
        },
      })
    );
  }, [isDraggingAllowed, issueId, groupId, setIsCurrentBlockDragging]);

  if (!issue) return null;

  const projectIdentifier = projectRows?.find((project) => project.id === issue.project_id)?.identifier ?? "";
  const isIssueSelected = selectionHelpers.getIsEntitySelected(issue.id);
  const isIssueActive = selectionHelpers.getIsEntityActive(issue.id);
  const isSubIssue = nestingLevel !== 0;
  const canSelectIssues = canEditIssueProperties && !selectionHelpers.isSelectionDisabled;

  const marginLeft = `${spacingLeft}px`;

  const handleToggleExpand = (e: MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    e.preventDefault();
    if (nestingLevel >= 3) {
      handleIssuePeekOverview(issue);
    } else {
      setExpanded((prevState) => !prevState);
    }
  };

  // Calculate width for: projectIdentifier + "-" + dynamic sequence number digits
  // Use next_work_item_sequence from backend (static value from project endpoint)
  const maxSequenceId = currentProjectNextSequenceId ?? 1;
  const keyMinWidth = displayProperties?.key
    ? calculateIdentifierWidth(projectIdentifier?.length ?? 0, maxSequenceId)
    : 0;

  const workItemLink = generateWorkItemLink({
    workspaceSlug,
    projectId: issue?.project_id,
    issueId,
    projectIdentifier,
    sequenceId: issue?.sequence_id,
    isEpic,
    isArchived: !!issue?.archived_at,
  });
  return (
    <ControlLink
      id={`issue-${issue.id}`}
      href={workItemLink}
      onClick={() => handleIssuePeekOverview(issue)}
      className="w-full cursor-pointer"
      disabled={!!issue?.tempId || issue?.is_draft}
    >
      <Row
        ref={issueRef}
        className={cn(
          "group/list-block relative flex min-h-11 flex-col gap-3 bg-layer-transparent py-3 text-13 transition-colors hover:bg-layer-transparent-hover",
          {
            "border-accent-strong": getIsIssuePeeked(issue.id) && peekIssue?.nestingLevel === nestingLevel,
            "border-strong-1": isIssueActive,
            "last:border-b-transparent": !getIsIssuePeeked(issue.id) && !isIssueActive,
            "bg-accent-primary/5 hover:bg-accent-primary/10": isIssueSelected,
            "bg-layer-1": isCurrentBlockDragging,
            "md:flex-row md:items-center": isSidebarCollapsed,
            "lg:flex-row lg:items-center": !isSidebarCollapsed,
          }
        )}
        onDragStart={() => {
          if (!isDraggingAllowed) {
            setToast({
              type: TOAST_TYPE.WARNING,
              title: "Cannot move work item",
              message: !canEditIssueProperties
                ? "You are not allowed to move this work item"
                : "Drag and drop is disabled for the current grouping",
            });
          }
        }}
      >
        <IssueBlockHeading
          issue={issue}
          issueId={issueId}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          displayProperties={displayProperties}
          isEpic={isEpic}
          isSubIssue={isSubIssue}
          marginLeft={marginLeft}
          keyMinWidth={keyMinWidth}
          subIssuesCount={subIssuesCount}
          isExpanded={isExpanded}
          onToggleExpand={handleToggleExpand}
          selectionHelpers={selectionHelpers}
          groupId={groupId}
          isIssueSelected={isIssueSelected}
          canSelectIssues={canSelectIssues}
          isMobile={isMobile}
          isCurrentBlockDragging={isCurrentBlockDragging}
          isSidebarCollapsed={Boolean(isSidebarCollapsed)}
          quickActions={quickActions}
          issueRef={issueRef}
        />
        <div className="flex flex-shrink-0 items-center gap-2">
          <IssueBlockProperties
            issue={issue}
            isEpic={isEpic}
            isSidebarCollapsed={Boolean(isSidebarCollapsed)}
            canEditIssueProperties={canEditIssueProperties}
            displayProperties={displayProperties}
            updateIssue={updateIssue}
            quickActions={quickActions}
            issueRef={issueRef}
          />
        </div>
      </Row>
    </ControlLink>
  );
};

type IssueBlockHeadingProps = {
  issue: TIssue;
  issueId: string;
  workspaceSlug: string | undefined;
  projectId: string | undefined;
  displayProperties: IIssueDisplayProperties | undefined;
  isEpic: boolean;
  isSubIssue: boolean;
  marginLeft: string;
  keyMinWidth: number;
  subIssuesCount: number;
  isExpanded: boolean;
  onToggleExpand: React.MouseEventHandler<HTMLButtonElement>;
  selectionHelpers: TSelectionHelper;
  groupId: string;
  isIssueSelected: boolean;
  canSelectIssues: boolean;
  isMobile: boolean;
  isCurrentBlockDragging: boolean;
  isSidebarCollapsed: boolean;
  quickActions: TRenderQuickActions;
  issueRef: React.RefObject<HTMLDivElement | null>;
};

function IssueBlockHeading(props: IssueBlockHeadingProps) {
  const {
    issue,
    issueId,
    workspaceSlug,
    projectId,
    displayProperties,
    isEpic,
    isSubIssue,
    marginLeft,
    keyMinWidth,
    subIssuesCount,
    isExpanded,
    onToggleExpand,
    selectionHelpers,
    groupId,
    isIssueSelected,
    canSelectIssues,
    isMobile,
    isCurrentBlockDragging,
    isSidebarCollapsed,
    quickActions,
    issueRef,
  } = props;

  return (
    <div className="flex w-full gap-2 truncate">
      <div className="flex flex-grow items-center gap-0.5 truncate">
        <div className="flex items-center gap-1" style={isSubIssue ? { marginLeft } : {}}>
          {projectId && canSelectIssues && !isEpic && (
            <Tooltip
              tooltipContent={
                <>
                  Only work items within the current
                  <br />
                  project can be selected.
                </>
              }
              disabled={issue.project_id === projectId}
            >
              <div className="absolute left-1 grid w-3.5 flex-shrink-0 place-items-center">
                <MultipleSelectEntityAction
                  className={cn(
                    "pointer-events-none opacity-0 transition-opacity group-hover/list-block:pointer-events-auto group-hover/list-block:opacity-100",
                    { "pointer-events-auto opacity-100": isIssueSelected }
                  )}
                  groupId={groupId}
                  id={issue.id}
                  selectionHelpers={selectionHelpers}
                  disabled={issue.project_id !== projectId}
                />
              </div>
            </Tooltip>
          )}
          {displayProperties && (displayProperties.key || displayProperties.issue_type) && (
            <div className="flex-shrink-0" style={{ minWidth: `${keyMinWidth}px` }}>
              {issue.project_id && (
                <IssueIdentifier
                  workspaceSlug={workspaceSlug}
                  issueId={issueId}
                  projectId={issue.project_id}
                  size="xs"
                  variant="tertiary"
                  displayProperties={displayProperties}
                />
              )}
            </div>
          )}
          <div className="grid size-4 flex-shrink-0 place-items-center">
            {subIssuesCount > 0 && !isEpic && (
              <button
                type="button"
                className="grid size-4 place-items-center rounded-xs text-placeholder hover:text-tertiary"
                onClick={onToggleExpand}
              >
                <ChevronRightIcon className={cn("size-4", { "rotate-90": isExpanded })} strokeWidth={2.5} />
              </button>
            )}
          </div>
          {issue.tempId !== undefined && (
            <div className="absolute top-0 left-0 z-[99999] h-full w-full animate-pulse bg-surface-1/20" />
          )}
        </div>
        <Tooltip
          tooltipContent={issue.name}
          isMobile={isMobile}
          position="top-start"
          disabled={isCurrentBlockDragging}
          renderByDefault={false}
        >
          <p className="cursor-pointer truncate text-body-xs-medium text-primary">{issue.name}</p>
        </Tooltip>
      </div>
      {!issue.tempId && (
        <div
          className={cn("block rounded-sm border border-strong", {
            "md:hidden": isSidebarCollapsed,
            "lg:hidden": !isSidebarCollapsed,
          })}
        >
          {quickActions({ issue, parentRef: issueRef })}
        </div>
      )}
    </div>
  );
}

function IssueBlockProperties({
  issue,
  isEpic,
  isSidebarCollapsed,
  canEditIssueProperties,
  displayProperties,
  updateIssue,
  quickActions,
  issueRef,
}: {
  issue: TIssue;
  isEpic: boolean;
  isSidebarCollapsed: boolean;
  canEditIssueProperties: boolean;
  displayProperties: IIssueDisplayProperties | undefined;
  updateIssue: IssueBlockProps["updateIssue"];
  quickActions: TRenderQuickActions;
  issueRef: React.RefObject<HTMLDivElement | null>;
}) {
  if (issue.tempId) return <Spinner className="h-4 w-4" />;

  return (
    <>
      <IssueProperties
        className={`relative flex flex-wrap ${isSidebarCollapsed ? "md:flex-shrink-0 md:flex-grow" : "lg:flex-shrink-0 lg:flex-grow"} items-center gap-2 whitespace-nowrap`}
        issue={issue}
        isReadOnly={!canEditIssueProperties}
        updateIssue={updateIssue}
        displayProperties={displayProperties}
        activeLayout="List"
        isEpic={isEpic}
      />
      {/* oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
      <div
        className={cn("hidden", { "md:flex": isSidebarCollapsed, "lg:flex": !isSidebarCollapsed })}
        onClick={(event) => {
          event.preventDefault();
          event.stopPropagation();
        }}
      >
        {quickActions({ issue, parentRef: issueRef })}
      </div>
    </>
  );
}
