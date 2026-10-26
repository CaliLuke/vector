/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import type { MutableRefObject } from "react";
import { useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { draggable, dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";

// plane helpers
import { MoreHorizontal } from "lucide-react";
import { useOutsideClickDetector } from "@plane/hooks";
// types
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Tooltip } from "@plane/propel/tooltip";
import type { TIssue, IIssueDisplayProperties, IIssueMap } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// ui
import { ControlLink, DropIndicator } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { generateWorkItemLink } from "@plane/utils/work-item";
// components
import RenderIfVisible from "@/components/core/render-if-visible-HOC";
import { HIGHLIGHT_CLASS, getIssueBlockId } from "@/components/issues/issue-layouts/utils";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useKanbanDragOwner } from "@/providers/issue-layout-ui-state";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import useIssuePeekOverviewRedirection from "@/hooks/use-issue-peek-overview-redirection";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local components
import type { TRenderQuickActions } from "../list/list-view-types";
import { IssueProperties } from "../properties/all-properties";

interface IssueBlockProps {
  issueId: string;
  groupId: string;
  subGroupId: string;
  issuesMap: IIssueMap;
  displayProperties: IIssueDisplayProperties | undefined;
  draggableId: string;
  canDropOverIssue: boolean;
  canDragIssuesInCurrentGrouping: boolean;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<TIssue>) => Promise<unknown>) | undefined;
  quickActions: TRenderQuickActions;
  canEditProperties: (projectId: string | undefined) => boolean;
  scrollableContainerRef?: MutableRefObject<HTMLDivElement | null>;
  shouldRenderByDefault?: boolean;
  isEpic?: boolean;
}

interface IssueDetailsBlockProps {
  cardRef: React.RefObject<HTMLElement | null>;
  issue: TIssue;
  displayProperties: IIssueDisplayProperties | undefined;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<TIssue>) => Promise<unknown>) | undefined;
  quickActions: TRenderQuickActions;
  isReadOnly: boolean;
  isEpic?: boolean;
}

const KanbanIssueDetailsBlock = function KanbanIssueDetailsBlock(props: IssueDetailsBlockProps) {
  const { cardRef, issue, updateIssue, quickActions, isReadOnly, displayProperties, isEpic = false } = props;
  const { workspaceSlug } = useParams({ strict: false });
  // refs
  const menuActionRef = useRef<HTMLDivElement | null>(null);
  // states
  const [isMenuActive, setIsMenuActive] = useState(false);
  // hooks
  const { isMobile } = usePlatformOS();

  const customActionButton = (
    // oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions
    <div
      ref={menuActionRef}
      className={`flex h-full w-full cursor-pointer items-center rounded-sm p-1 text-placeholder hover:bg-layer-1 ${
        isMenuActive ? "bg-layer-1 text-primary" : "text-secondary"
      }`}
      onClick={() => setIsMenuActive(!isMenuActive)}
    >
      <MoreHorizontal className="h-3.5 w-3.5" />
    </div>
  );

  // oxlint-disable-next-line unicorn/consistent-function-scoping
  const handleEventPropagation = (e: React.MouseEvent) => {
    e.stopPropagation();
    e.preventDefault();
  };

  useOutsideClickDetector(menuActionRef, () => setIsMenuActive(false));

  return (
    <>
      <div className="relative">
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
        {/* oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
        <div
          className={cn("absolute -top-1 right-0", {
            "hidden group-hover/kanban-block:block": !isMobile,
            "!block": isMenuActive,
          })}
          onClick={handleEventPropagation}
        >
          {quickActions({
            issue,
            parentRef: cardRef,
            customActionButton,
          })}
        </div>
      </div>

      <Tooltip tooltipContent={issue.name} isMobile={isMobile} renderByDefault={false}>
        <div className="line-clamp-1 w-full text-body-sm-medium text-primary">
          <span>{issue.name}</span>
        </div>
      </Tooltip>

      <IssueProperties
        className="flex flex-wrap items-center gap-2 pt-1.5 whitespace-nowrap text-tertiary"
        issue={issue}
        displayProperties={displayProperties}
        activeLayout="Kanban"
        updateIssue={updateIssue}
        isReadOnly={isReadOnly}
        isEpic={isEpic}
      />
    </>
  );
};

export const KanbanIssueBlock = function KanbanIssueBlock(props: IssueBlockProps) {
  const {
    issueId,
    issuesMap,
    canDropOverIssue,
    canDragIssuesInCurrentGrouping,
    canEditProperties,
    isEpic = false,
  } = props;

  const cardRef = useRef<HTMLAnchorElement | null>(null);
  // router
  const { workspaceSlug } = useParams({ strict: false });
  // hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const { getIsIssuePeeked } = useIssueDetailUiState(isEpic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES);
  const { handleRedirection } = useIssuePeekOverviewRedirection(isEpic);
  const { isMobile } = usePlatformOS();

  // handlers
  const handleIssuePeekOverview = (issue: TIssue) => handleRedirection(workspaceSlug, issue, isMobile);

  const issue = issuesMap[issueId];

  const { beginDrag, endDrag } = useKanbanDragOwner();

  const [isDraggingOverBlock, setIsDraggingOverBlock] = useState(false);
  const [isCurrentBlockDragging, setIsCurrentBlockDragging] = useState(false);

  const canEditIssueProperties = canEditProperties(issue?.project_id ?? undefined);

  const isDragAllowed = canDragIssuesInCurrentGrouping && !issue?.tempId && canEditIssueProperties;
  const workItemLink = getKanbanWorkItemLink({ workspaceSlug, issueId, issue, projects: projectRows, isEpic });

  useOutsideClickDetector(cardRef, () => {
    cardRef?.current?.classList?.remove(HIGHLIGHT_CLASS);
  });

  useKanbanBlockDrag({
    cardRef,
    issueId: issue?.id,
    isDragAllowed,
    canDropOverIssue,
    beginDrag,
    endDrag,
    setIsCurrentBlockDragging,
    setIsDraggingOverBlock,
  });

  if (!issue) return null;

  return (
    <KanbanIssueCard
      {...props}
      issue={issue}
      cardRef={cardRef}
      workItemLink={workItemLink}
      isDragAllowed={isDragAllowed}
      canEditIssueProperties={canEditIssueProperties}
      isDraggingOverBlock={isDraggingOverBlock}
      isCurrentBlockDragging={isCurrentBlockDragging}
      isIssuePeeked={getIsIssuePeeked(issue.id)}
      onPeek={handleIssuePeekOverview}
      setIsCurrentBlockDragging={setIsCurrentBlockDragging}
    />
  );
};

type KanbanIssueCardProps = IssueBlockProps & {
  issue: TIssue;
  cardRef: React.RefObject<HTMLAnchorElement | null>;
  workItemLink: string;
  isDragAllowed: boolean;
  canEditIssueProperties: boolean;
  isDraggingOverBlock: boolean;
  isCurrentBlockDragging: boolean;
  isIssuePeeked: boolean;
  onPeek: (issue: TIssue) => void;
  setIsCurrentBlockDragging: React.Dispatch<React.SetStateAction<boolean>>;
};

function KanbanIssueCard(props: KanbanIssueCardProps) {
  const {
    issue,
    issueId,
    groupId,
    subGroupId,
    cardRef,
    workItemLink,
    isDragAllowed,
    canEditIssueProperties,
    isDraggingOverBlock,
    isCurrentBlockDragging,
    isIssuePeeked,
    onPeek,
    setIsCurrentBlockDragging,
    scrollableContainerRef,
    shouldRenderByDefault,
    displayProperties,
    updateIssue,
    quickActions,
    isEpic = false,
  } = props;

  return (
    <>
      <DropIndicator isVisible={!isCurrentBlockDragging && isDraggingOverBlock} />
      <div
        id={`issue-${issueId}`}
        // make Z-index higher at the beginning of drag, to have a issue drag image of issue block without any overlaps
        className={cn("group/kanban-block relative mb-2", { "z-[1]": isCurrentBlockDragging })}
        onDragStart={() => handleKanbanDragStart(isDragAllowed, canEditIssueProperties, setIsCurrentBlockDragging)}
      >
        <ControlLink
          id={getIssueBlockId(issueId, groupId, subGroupId)}
          href={workItemLink}
          ref={cardRef}
          className={cn(
            "block w-full rounded-lg border border-subtle bg-layer-2 p-3 text-13 shadow-raised-100 outline-[0.5px] outline-transparent transition-all hover:border-strong hover:shadow-raised-200",
            { "hover:cursor-pointer": isDragAllowed },
            { "border border-accent-strong hover:border-accent-strong": isIssuePeeked },
            { "z-[100] bg-layer-1": isCurrentBlockDragging }
          )}
          onClick={() => onPeek(issue)}
          disabled={!!issue?.tempId}
        >
          <RenderIfVisible
            classNames="space-y-2"
            root={scrollableContainerRef}
            defaultHeight="100px"
            horizontalOffset={100}
            verticalOffset={200}
            defaultValue={shouldRenderByDefault}
          >
            <KanbanIssueDetailsBlock
              cardRef={cardRef}
              issue={issue}
              displayProperties={displayProperties}
              updateIssue={updateIssue}
              quickActions={quickActions}
              isReadOnly={!canEditIssueProperties}
              isEpic={isEpic}
            />
          </RenderIfVisible>
        </ControlLink>
      </div>
    </>
  );
}

function getKanbanWorkItemLink({
  workspaceSlug,
  issueId,
  issue,
  projects,
  isEpic,
}: {
  workspaceSlug: string | undefined;
  issueId: string;
  issue: TIssue | undefined;
  projects: ReturnType<typeof useWorkspaceProjectDetails>["projects"];
  isEpic: boolean;
}) {
  const projectIdentifier = projects?.find((project) => project.id === issue?.project_id)?.identifier ?? "";
  return generateWorkItemLink({
    workspaceSlug,
    projectId: issue?.project_id,
    issueId,
    projectIdentifier,
    sequenceId: issue?.sequence_id,
    isEpic,
    isArchived: !!issue?.archived_at,
  });
}

function handleKanbanDragStart(
  isDragAllowed: boolean,
  canEditIssueProperties: boolean,
  setIsDragging: React.Dispatch<React.SetStateAction<boolean>>
) {
  if (isDragAllowed) {
    setIsDragging(true);
    return;
  }

  setToast({
    type: TOAST_TYPE.WARNING,
    title: "Cannot move work item",
    message: canEditIssueProperties
      ? "Drag and drop is disabled for the current grouping"
      : "You are not allowed to move this work item",
  });
}

function useKanbanBlockDrag({
  cardRef,
  issueId,
  isDragAllowed,
  canDropOverIssue,
  beginDrag,
  endDrag,
  setIsCurrentBlockDragging,
  setIsDraggingOverBlock,
}: {
  cardRef: React.RefObject<HTMLAnchorElement | null>;
  issueId: string | undefined;
  isDragAllowed: boolean;
  canDropOverIssue: boolean;
  beginDrag: () => void;
  endDrag: () => void;
  setIsCurrentBlockDragging: React.Dispatch<React.SetStateAction<boolean>>;
  setIsDraggingOverBlock: React.Dispatch<React.SetStateAction<boolean>>;
}) {
  useEffect(() => {
    const element = cardRef.current;
    if (!element) return endDrag;

    const cleanup = combine(
      draggable({
        element,
        dragHandle: element,
        canDrag: () => isDragAllowed,
        getInitialData: () => ({ id: issueId, type: "ISSUE" }),
        onDragStart: () => {
          setIsCurrentBlockDragging(true);
          beginDrag();
        },
        onDrop: () => {
          endDrag();
          setIsCurrentBlockDragging(false);
        },
      }),
      dropTargetForElements({
        element,
        canDrop: ({ source }) => source?.data?.id !== issueId && canDropOverIssue,
        getData: () => ({ id: issueId, type: "ISSUE" }),
        onDragEnter: () => setIsDraggingOverBlock(true),
        onDragLeave: () => setIsDraggingOverBlock(false),
        onDrop: () => setIsDraggingOverBlock(false),
      })
    );
    return () => {
      cleanup();
      endDrag();
    };
  }, [
    beginDrag,
    canDropOverIssue,
    cardRef,
    endDrag,
    isDragAllowed,
    issueId,
    setIsCurrentBlockDragging,
    setIsDraggingOverBlock,
  ]);
}
