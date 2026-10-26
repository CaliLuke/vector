/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import React from "react";

// types
import type { TIssue } from "@plane/types";
// helpers
import { Row } from "@plane/ui";
import { cn } from "@plane/utils/common";
// hooks

type Props = {
  issue: TIssue;
};

export const SpreadsheetSubIssueColumn = function SpreadsheetSubIssueColumn(props: Props) {
  const { issue } = props;
  // router
  const router = useRouter();
  // hooks
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // derived values
  const isEpic = issue?.is_epic;
  const subIssueCount = issue?.sub_issues_count ?? 0;

  const redirectToIssueDetail = () => {
    void router.navigate({
      href: `/${workspaceSlug?.toString()}/projects/${issue.project_id}/${issue.archived_at ? "archives/" : ""}${isEpic ? "epics" : "issues"}/${issue.id}#sub-issues`,
    });
  };

  const label = `${subIssueCount} sub-work item${subIssueCount !== 1 ? "s" : ""}`;

  return (
    <Row
      onClick={subIssueCount ? redirectToIssueDetail : () => {}}
      className={cn(
        "flex h-11 w-full items-center border-b-[0.5px] border-subtle py-1 text-11 group-[.selected-issue-row]:bg-accent-primary/5 hover:bg-surface-2 group-[.selected-issue-row]:hover:bg-accent-primary",
        {
          "cursor-pointer": subIssueCount,
        }
      )}
    >
      {isEpic ? null : label}
    </Row>
  );
};
