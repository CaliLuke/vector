/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useWorkItemSubItems } from "@/hooks/use-work-item-detail";
// components
// hooks
// types
import { IssueParentSiblingItem } from "./sibling-item";

export type TIssueParentSiblings = {
  workspaceSlug: string;
  currentIssueId: string;
  parentIssueId: string;
  parentProjectId: string;
};

export const IssueParentSiblings = function IssueParentSiblings(props: TIssueParentSiblings) {
  const { workspaceSlug, currentIssueId, parentIssueId, parentProjectId } = props;
  // hooks
  const children = useWorkItemSubItems(workspaceSlug, parentProjectId, parentIssueId);
  const isLoading = children.isPending;
  const subIssues = children.subItems;

  return (
    <div className="my-1">
      {isLoading ? (
        <div className="flex items-center gap-2 px-1 py-1 text-left text-11 whitespace-nowrap text-secondary">
          Loading
        </div>
      ) : subIssues && subIssues.length > 0 ? (
        subIssues.map(
          (issue) =>
            currentIssueId !== issue.id && (
              <IssueParentSiblingItem key={issue.id} workspaceSlug={workspaceSlug} issue={issue} />
            )
        )
      ) : (
        <div className="flex items-center gap-2 px-1 py-1 text-left text-11 whitespace-nowrap text-secondary">
          No sibling work items
        </div>
      )}
    </div>
  );
};
