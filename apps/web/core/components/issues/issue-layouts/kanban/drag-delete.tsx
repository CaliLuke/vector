/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useEffect, useRef, useState } from "react";
import { dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import type { TIssueMap } from "@plane/types";
import { useKanbanDragState } from "@/providers/issue-layout-ui-state";
import { DeleteIssueModal } from "../../delete-issue-modal";
import { getSourceFromDropPayload } from "../utils";

type Props = {
  issuesById: TIssueMap;
  removeIssue: (projectId: string | null, issueId: string) => Promise<unknown>;
  isEpic: boolean;
};

export function KanbanDragDelete({ issuesById, removeIssue, isEpic }: Props) {
  const targetRef = useRef<HTMLDivElement>(null);
  const [isDragOver, setIsDragOver] = useState(false);
  const [selectedId, setSelectedId] = useState<string>();
  const { isDragging } = useKanbanDragState();
  useEffect(() => {
    const element = targetRef.current;
    if (!element) return undefined;
    return dropTargetForElements({
      element,
      getData: () => ({ columnId: "issue-trash-box", groupId: "issue-trash-box", type: "DELETE" }),
      onDragEnter: () => setIsDragOver(true),
      onDragLeave: () => setIsDragOver(false),
      onDrop: (payload) => {
        setIsDragOver(false);
        const source = getSourceFromDropPayload(payload);
        if (source) setSelectedId(source.id);
      },
    });
  }, []);
  const selectedIssue = issuesById[selectedId ?? ""];
  return (
    <>
      <DeleteIssueModal
        data={selectedIssue}
        isOpen={Boolean(selectedId)}
        handleClose={() => setSelectedId(undefined)}
        isEpic={isEpic}
        onSubmit={
          selectedIssue
            ? async () => {
                await removeIssue(selectedIssue.project_id, selectedIssue.id);
              }
            : undefined
        }
      />
      <div
        ref={targetRef}
        className={`fixed left-1/2 -translate-x-1/2 ${isDragging ? "z-40" : ""} top-3 mx-3 flex w-72 items-center justify-center`}
      >
        <div
          className={`${isDragging ? "opacity-100" : "opacity-0"} flex w-full items-center justify-center rounded-sm border-2 border-danger-strong/20 bg-surface-1 px-3 py-5 text-11 font-medium text-danger-primary italic ${isDragOver ? "bg-danger-primary blur-2xl" : ""} transition duration-300`}
        >
          Drop here to delete the work item.
        </div>
      </div>
    </>
  );
}
