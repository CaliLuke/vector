/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

// components
import type { TIssuePriorities } from "@plane/types";
import type { TIssue } from "@plane/types";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { PriorityDropdown } from "@/components/dropdowns/priority";
import { StateDropdown } from "@/components/dropdowns/state/dropdown";
// hooks
// types
import type { TRelationIssueOperations } from "../issue-detail-widgets/relations/helper";

type Props = {
  workspaceSlug: string;
  issueId: string;
  issue: TIssue;
  disabled: boolean;
  issueOperations: TRelationIssueOperations;
};

export const RelationIssueProperty = function RelationIssueProperty(props: Props) {
  const { workspaceSlug, issueId, issue, disabled, issueOperations } = props;

  // handlers
  const handleStateChange = (val: string) =>
    issue.project_id &&
    issueOperations.update(workspaceSlug, issue.project_id, issueId, {
      state_id: val,
    });

  const handlePriorityChange = (val: TIssuePriorities) =>
    issue.project_id &&
    issueOperations.update(workspaceSlug, issue.project_id, issueId, {
      priority: val,
    });

  const handleAssigneeChange = (val: string[]) =>
    issue.project_id &&
    issueOperations.update(workspaceSlug, issue.project_id, issueId, {
      assignee_ids: val,
    });

  return (
    <div className="relative flex items-center gap-2">
      <div className="h-5 flex-shrink-0">
        <StateDropdown
          value={issue.state_id}
          projectId={issue.project_id ?? undefined}
          onChange={handleStateChange}
          disabled={disabled}
          buttonVariant="border-with-text"
        />
      </div>

      <div className="h-5 flex-shrink-0">
        <PriorityDropdown
          value={issue.priority}
          onChange={handlePriorityChange}
          disabled={disabled}
          buttonVariant="border-without-text"
          buttonClassName="border"
        />
      </div>

      <div className="h-5 flex-shrink-0">
        <MemberDropdown
          workspaceSlug={workspaceSlug}
          value={issue.assignee_ids}
          projectId={issue.project_id ?? undefined}
          onChange={handleAssigneeChange}
          disabled={disabled}
          multiple
          buttonVariant={(issue?.assignee_ids || []).length > 0 ? "transparent-without-text" : "border-without-text"}
          buttonClassName={(issue?.assignee_ids || []).length > 0 ? "hover:bg-transparent px-0" : ""}
        />
      </div>
    </div>
  );
};
