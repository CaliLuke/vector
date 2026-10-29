/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@plane/i18n";
import { cn } from "@plane/utils/common";
import { CycleDropdown } from "@/components/dropdowns/cycle";
import { useIssueAction } from "@/hooks/use-issue-action";
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import type { TIssueOperations } from "./root";

type TIssueCycleSelect = {
  className?: string;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueOperations: TIssueOperations;
  disabled?: boolean;
};

export const IssueCycleSelect = function IssueCycleSelect(props: TIssueCycleSelect) {
  const { className = "", workspaceSlug, projectId, issueId, issueOperations, disabled = false } = props;
  const { t } = useTranslation();
  const { issue } = useWorkItemDetail(workspaceSlug, projectId, issueId);
  const change = useIssueAction(
    { workspaceSlug, projectId, issueId, context: "detail" },
    !disabled && !!issue && !!issueOperations.addCycleToIssue && !!issueOperations.removeIssueFromCycle
  );
  const handleIssueCycleChange = (cycleId: string | null) => {
    const action = cycleId ? issueOperations.addCycleToIssue : issueOperations.removeIssueFromCycle;
    if (!issue || issue.cycle_id === cycleId || !action) return;
    change.run(() => action(workspaceSlug, projectId, cycleId ?? issue.cycle_id ?? "", issueId), {
      error: t(cycleId ? "issue.add.cycle.failed" : "issue.remove.cycle.failed"),
      success: cycleId ? undefined : t("issue.remove.cycle.success"),
    });
  };

  return (
    <div className={cn("flex h-full items-center gap-1", className)}>
      <CycleDropdown
        value={issue?.cycle_id ?? null}
        onChange={handleIssueCycleChange}
        projectId={projectId}
        disabled={change.disabled}
        buttonVariant="transparent-with-text"
        className="group w-full"
        buttonContainerClassName="w-full text-left h-7.5 rounded-sm"
        buttonClassName={`text-body-xs-medium justify-between ${issue?.cycle_id ? "" : "text-placeholder"}`}
        placeholder={t("cycle.no_cycle")}
        hideIcon
        dropdownArrow
        dropdownArrowClassName="h-3.5 w-3.5 hidden group-hover:inline"
      />
    </div>
  );
};
