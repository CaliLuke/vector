/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useCallback } from "react";
import type { FC } from "react";
import type { IssueActions } from "@/hooks/use-issues-actions";
import type { IQuickActionProps, TRenderQuickActions } from "./list/list-view-types";

type Props = Parameters<TRenderQuickActions>[0] & {
  QuickActions: FC<IQuickActionProps>;
  readOnly: boolean;
  actions: Pick<IssueActions, "removeIssue" | "updateIssue" | "removeIssueFromView" | "archiveIssue" | "restoreIssue">;
};

export function IssueLayoutQuickActions({
  QuickActions,
  issue,
  parentRef,
  customActionButton,
  placement,
  portalElement,
  readOnly,
  actions,
}: Props) {
  return (
    <QuickActions
      parentRef={parentRef}
      customActionButton={customActionButton}
      issue={issue}
      handleDelete={() => actions.removeIssue(issue.project_id, issue.id)}
      handleUpdate={async (data) => {
        await actions.updateIssue?.(issue.project_id, issue.id, data);
      }}
      handleRemoveFromView={async () => actions.removeIssueFromView?.(issue.project_id, issue.id)}
      handleArchive={async () => actions.archiveIssue?.(issue.project_id, issue.id)}
      handleRestore={async () => actions.restoreIssue?.(issue.project_id, issue.id)}
      readOnly={readOnly}
      placements={placement}
      portalElement={portalElement}
    />
  );
}

/** Shared renderer for card and row menus; each layout supplies its permission check. */
export function useIssueLayoutQuickActions({
  QuickActions,
  actions,
  canEditProperties,
  isCompletedCycle,
}: Pick<Props, "QuickActions" | "actions"> & {
  canEditProperties: (projectId: string | undefined) => boolean;
  isCompletedCycle: boolean;
}): TRenderQuickActions {
  return useCallback(
    ({ issue, parentRef, customActionButton }) => (
      <IssueLayoutQuickActions
        QuickActions={QuickActions}
        issue={issue}
        parentRef={parentRef}
        customActionButton={customActionButton}
        actions={actions}
        readOnly={!canEditProperties(issue.project_id ?? undefined) || isCompletedCycle}
      />
    ),
    [QuickActions, actions, canEditProperties, isCompletedCycle]
  );
}
