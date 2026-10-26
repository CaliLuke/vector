/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useCallback } from "react";

// types
import type { TIssue } from "@plane/types";
// components
import { CycleDropdown } from "@/components/dropdowns/cycle";
// hooks
import { useIssueListType } from "@/hooks/use-issue-list-type";
import { useIssuesActions } from "@/hooks/use-issues-actions";

type Props = {
  issue: TIssue;
  onClose: () => void;
  disabled: boolean;
};

export function SpreadsheetCycleColumn(props: Props) {
  const { issue, disabled, onClose } = props;
  // router
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // hooks
  const actions = useIssuesActions(useIssueListType());

  const handleCycle = useCallback(
    async (cycleId: string | null) => {
      if (!workspaceSlug || !issue || !issue.project_id || issue.cycle_id === cycleId) return;
      if (cycleId) await actions.addCycleToIssue(issue.project_id, cycleId, issue.id);
      else if (issue.cycle_id) await actions.removeCycleFromIssue(issue.project_id, issue.cycle_id, issue.id);
    },
    [workspaceSlug, issue, actions]
  );

  return (
    <div className="h-11 border-b-[0.5px] border-subtle">
      <CycleDropdown
        projectId={issue.project_id ?? undefined}
        value={issue.cycle_id}
        onChange={handleCycle}
        disabled={disabled}
        placeholder="Select cycle"
        buttonVariant="transparent-with-text"
        buttonContainerClassName="w-full relative flex items-center p-2 group-[.selected-issue-row]:bg-accent-primary/5 group-[.selected-issue-row]:hover:bg-accent-primary/10 px-page-x"
        buttonClassName="relative leading-4 h-4.5 bg-transparent hover:bg-transparent px-0"
        onClose={onClose}
      />
    </div>
  );
}
