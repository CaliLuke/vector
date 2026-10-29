/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { useTranslation } from "@plane/i18n";
import { useIssueAction } from "@/hooks/use-issue-action";
import type { useIssueRelations } from "@/hooks/use-issue-resources";

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
  captureRelations: ReturnType<typeof useIssueRelations>["captureReconciliation"];
};

export const RelationIssueProperty = function RelationIssueProperty(props: Props) {
  const { workspaceSlug, issueId, issue, disabled: readOnly, issueOperations, captureRelations } = props;
  const { t } = useTranslation();
  const action = useIssueAction(
    { workspaceSlug, projectId: issue.project_id, issueId, context: "relation-property" },
    !readOnly && Boolean(issue.project_id),
    { success: t("entity.update.success", { entity: t("common.work_item") }) }
  );
  const disabled = action.disabled;
  const update = (data: Partial<TIssue>) =>
    action.run(async () => {
      const owner = captureRelations();
      if (!owner.isCurrent() || !issue.project_id) throw new Error("The work item relation has changed.");
      await issueOperations.update(data);
      await owner.refresh();
    });
  const handleStateChange = (state_id: string) => update({ state_id });
  const handlePriorityChange = (priority: TIssuePriorities) => update({ priority });
  const handleAssigneeChange = (assignee_ids: string[]) => update({ assignee_ids });

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
