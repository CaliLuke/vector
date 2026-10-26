/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

// plane imports
import type { TIssue, TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
import type { TIssueRelationTypes } from "@plane/types";
// local imports
import { RelationIssueListItem } from "./issue-list-item";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issues: TIssue[];
  relationKey: TIssueRelationTypes;
  handleIssueCrudState: (
    key: "update" | "delete" | "removeRelation",
    issueId: string,
    issue?: TIssue | null,
    relationKey?: TIssueRelationTypes | null,
    relationIssueId?: string | null
  ) => void;
  disabled?: boolean;
  issueServiceType?: TIssueServiceType;
};

export const RelationIssueList = function RelationIssueList(props: Props) {
  const {
    workspaceSlug,
    projectId,
    issueId,
    issues,
    relationKey,
    disabled = false,
    handleIssueCrudState,
    issueServiceType = EIssueServiceType.ISSUES,
  } = props;

  return (
    <div className="relative">
      {issues.length > 0 &&
        issues.map((issue) => (
          <RelationIssueListItem
            key={issue.id}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            issueId={issueId}
            relationKey={relationKey}
            issue={issue}
            disabled={disabled}
            handleIssueCrudState={handleIssueCrudState}
            issueServiceType={issueServiceType}
          />
        ))}
    </div>
  );
};
