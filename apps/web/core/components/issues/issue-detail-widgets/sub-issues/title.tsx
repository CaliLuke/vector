/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { useTranslation } from "@plane/i18n";
import type { TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
import { CircularProgressIndicator, CollapsibleButton } from "@plane/ui";
// hooks
import { useWorkItemSubItems } from "@/hooks/use-work-item-detail";

type Props = {
  isOpen: boolean;
  parentIssueId: string;
  workspaceSlug: string;
  projectId: string;
  issueServiceType?: TIssueServiceType;
};

export const SubIssuesCollapsibleTitle = function SubIssuesCollapsibleTitle(props: Props) {
  const { isOpen, parentIssueId, workspaceSlug, projectId, issueServiceType = EIssueServiceType.ISSUES } = props;
  // translation
  const { t } = useTranslation();
  // store hooks
  const { subItems, stateDistribution } = useWorkItemSubItems(
    workspaceSlug,
    projectId,
    parentIssueId,
    issueServiceType
  );
  const completedCount = stateDistribution?.completed?.length ?? 0;
  const totalCount = subItems?.length ?? 0;
  // if there are no sub-issues, return null
  if (!subItems) return null;

  // calculate percentage of completed sub-issues
  const percentage = completedCount && totalCount ? (completedCount / totalCount) * 100 : 0;

  return (
    <CollapsibleButton
      isOpen={isOpen}
      title={`${issueServiceType === EIssueServiceType.EPICS ? t("issue.label", { count: 1 }) : t("common.sub_work_items")}`}
      indicatorElement={
        <div className="flex items-center gap-1.5 text-13 text-tertiary">
          <CircularProgressIndicator size={18} percentage={percentage} strokeWidth={3} />
          <span>
            {completedCount}/{totalCount} {t("common.done")}
          </span>
        </div>
      }
    />
  );
};
