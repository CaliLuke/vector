/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { MutableRefObject } from "react";
import React, { useEffect, useRef, useState } from "react";
import { useParams } from "@tanstack/react-router";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { attachInstruction, extractInstruction } from "@atlaskit/pragmatic-drag-and-drop-hitbox/tree-item";

// plane helpers
import { useOutsideClickDetector } from "@plane/hooks";
// types
import type { IIssueDisplayProperties, TIssue, TIssueMap } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// components
import { DropIndicator } from "@plane/ui";
import RenderIfVisible from "@/components/core/render-if-visible-HOC";
import { ListLoaderItemRow } from "@/components/ui/loader/layouts/list-layout-loader";
// hooks
import type { TSelectionHelper } from "@/hooks/use-multiple-select";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { useWorkItemSubItems } from "@/hooks/use-work-item-detail";
// types
import { HIGHLIGHT_CLASS, getIssueBlockId, isIssueNew } from "../utils";
import { IssueBlock } from "./block";
import type { TRenderQuickActions } from "./list-view-types";

type Props = {
  issueId: string;
  issuesMap: TIssueMap;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<TIssue>) => Promise<unknown>) | undefined;
  quickActions: TRenderQuickActions;
  canEditProperties: (projectId: string | undefined) => boolean;
  displayProperties: IIssueDisplayProperties | undefined;
  nestingLevel: number;
  spacingLeft?: number;
  containerRef: MutableRefObject<HTMLDivElement | null>;
  selectionHelpers: TSelectionHelper;
  groupId: string;
  isDragAllowed: boolean;
  canDropOverIssue: boolean;
  isParentIssueBeingDragged?: boolean;
  isLastChild?: boolean;
  shouldRenderByDefault?: boolean;
  isEpic?: boolean;
};

export const IssueBlockRoot = function IssueBlockRoot(props: Props) {
  const { issueId, issuesMap, groupId, nestingLevel, canDropOverIssue, isLastChild = false, isEpic = false } = props;
  const { workspaceSlug } = useParams({ strict: false });
  // states
  const [isExpanded, setExpanded] = useState<boolean>(false);
  const [instruction, setInstruction] = useState<"DRAG_OVER" | "DRAG_BELOW" | undefined>(undefined);
  const [isCurrentBlockDragging, setIsCurrentBlockDragging] = useState(false);
  // ref
  const issueBlockRef = useRef<HTMLDivElement | null>(null);
  // hooks
  const { isMobile } = usePlatformOS();
  // store hooks
  const isSubIssue = nestingLevel !== 0;
  const issue = issuesMap[issueId];
  const childQuery = useWorkItemSubItems(
    workspaceSlug ?? undefined,
    issue?.project_id ?? undefined,
    issueId,
    isEpic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES,
    isExpanded && Boolean(issue?.created_at)
  );

  useIssueBlockDropTarget(issueBlockRef, {
    issueId,
    isLastChild,
    isSubIssue,
    canDropOverIssue,
    setInstruction,
  });

  useOutsideClickDetector(issueBlockRef, () => {
    issueBlockRef?.current?.classList?.remove(HIGHLIGHT_CLASS);
  });

  if (!issueId || !issue?.created_at) return null;

  const subIssues = childQuery.subItems;
  const visibleIssueMap = { ...issuesMap, ...Object.fromEntries((subIssues ?? []).map((item) => [item.id, item])) };
  return (
    <div className="relative" ref={issueBlockRef} id={getIssueBlockId(issueId, groupId)}>
      <DropIndicator classNames={"absolute top-0 z-[2]"} isVisible={instruction === "DRAG_OVER"} />
      <IssueBlockVirtualRow
        props={props}
        isMobile={isMobile}
        isSubIssue={isSubIssue}
        isExpanded={isExpanded}
        setExpanded={setExpanded}
        isCurrentBlockDragging={isCurrentBlockDragging}
        setIsCurrentBlockDragging={setIsCurrentBlockDragging}
        visibleIssueMap={visibleIssueMap}
      />
      <IssueBlockSubItems
        props={props}
        subItems={subIssues}
        isLoaded={childQuery.isSuccess}
        isExpanded={isExpanded}
        isCurrentBlockDragging={isCurrentBlockDragging}
        visibleIssueMap={visibleIssueMap}
      />
      {isLastChild && <DropIndicator classNames={"absolute z-[2]"} isVisible={instruction === "DRAG_BELOW"} />}
    </div>
  );
};

function IssueBlockVirtualRow({
  props,
  isMobile,
  isSubIssue,
  isExpanded,
  setExpanded,
  isCurrentBlockDragging,
  setIsCurrentBlockDragging,
  visibleIssueMap,
}: {
  props: Props;
  isMobile: boolean;
  isSubIssue: boolean;
  isExpanded: boolean;
  setExpanded: React.Dispatch<React.SetStateAction<boolean>>;
  isCurrentBlockDragging: boolean;
  setIsCurrentBlockDragging: React.Dispatch<React.SetStateAction<boolean>>;
  visibleIssueMap: TIssueMap;
}) {
  const {
    issueId,
    groupId,
    updateIssue,
    quickActions,
    canEditProperties,
    displayProperties,
    nestingLevel,
    spacingLeft = 14,
    containerRef,
    isDragAllowed,
    isParentIssueBeingDragged = false,
    isLastChild = false,
    selectionHelpers,
    shouldRenderByDefault,
    isEpic = false,
    issuesMap,
  } = props;
  const issue = issuesMap[issueId];
  if (!issue) return null;

  return (
    <RenderIfVisible
      key={issueId}
      root={containerRef}
      classNames={`relative ${isLastChild && !isExpanded ? "" : "border-b border-b-subtle"}`}
      verticalOffset={100}
      defaultValue={shouldRenderByDefault || isIssueNew(issue)}
      placeholderChildren={<ListLoaderItemRow shouldAnimate={false} renderForPlaceHolder defaultPropertyCount={4} />}
      shouldRecordHeights={isMobile}
    >
      <IssueBlock
        issueId={issueId}
        issuesMap={visibleIssueMap}
        groupId={groupId}
        updateIssue={updateIssue}
        quickActions={quickActions}
        canEditProperties={canEditProperties}
        displayProperties={displayProperties}
        isExpanded={isExpanded}
        setExpanded={setExpanded}
        nestingLevel={nestingLevel}
        spacingLeft={spacingLeft}
        selectionHelpers={selectionHelpers}
        canDrag={!isSubIssue && isDragAllowed}
        isCurrentBlockDragging={isParentIssueBeingDragged || isCurrentBlockDragging}
        setIsCurrentBlockDragging={setIsCurrentBlockDragging}
        isEpic={isEpic}
      />
    </RenderIfVisible>
  );
}

function IssueBlockSubItems({
  props,
  subItems,
  isLoaded,
  isExpanded,
  isCurrentBlockDragging,
  visibleIssueMap,
}: {
  props: Props;
  subItems: TIssue[] | undefined;
  isLoaded: boolean;
  isExpanded: boolean;
  isCurrentBlockDragging: boolean;
  visibleIssueMap: TIssueMap;
}) {
  const { isEpic = false } = props;
  if (!isExpanded || isEpic || !isLoaded || !subItems) return null;

  return subItems.map((subItem) => (
    <IssueBlockRoot
      key={subItem.id}
      {...props}
      issueId={subItem.id}
      issuesMap={visibleIssueMap}
      nestingLevel={props.nestingLevel + 1}
      spacingLeft={(props.spacingLeft ?? 14) + 12}
      isParentIssueBeingDragged={props.isParentIssueBeingDragged || isCurrentBlockDragging}
      shouldRenderByDefault={isExpanded}
      isLastChild={false}
    />
  ));
}

function useIssueBlockDropTarget(
  blockRef: React.RefObject<HTMLDivElement | null>,
  options: {
    issueId: string;
    isLastChild: boolean;
    isSubIssue: boolean;
    canDropOverIssue: boolean;
    setInstruction: React.Dispatch<React.SetStateAction<"DRAG_OVER" | "DRAG_BELOW" | undefined>>;
  }
) {
  const { issueId, isLastChild, isSubIssue, canDropOverIssue, setInstruction } = options;

  useEffect(() => {
    const element = blockRef.current;
    if (!element) return undefined;

    return combine(
      dropTargetForElements({
        element,
        canDrop: ({ source }) => source?.data?.id !== issueId && !isSubIssue && canDropOverIssue,
        getData: ({ input, element: target }) =>
          attachInstruction(
            { id: issueId, type: "ISSUE" },
            {
              input,
              element: target,
              currentLevel: 0,
              indentPerLevel: 0,
              mode: isLastChild ? "last-in-group" : "standard",
            }
          ),
        onDrag: ({ self }) => setInstruction(getIssueDropInstruction(self?.data, isLastChild)),
        onDragLeave: () => setInstruction(undefined),
        onDrop: () => setInstruction(undefined),
      })
    );
  }, [blockRef, canDropOverIssue, isLastChild, isSubIssue, issueId, setInstruction]);
}

function getIssueDropInstruction(data: Record<string | symbol, unknown> | undefined, isLastChild: boolean) {
  const instruction = extractInstruction(data ?? {})?.type;
  if (!instruction) return undefined;
  return instruction === "reorder-below" && isLastChild ? "DRAG_BELOW" : "DRAG_OVER";
}
