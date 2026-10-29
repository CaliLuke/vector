/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { SyntheticEvent } from "react";

import { useTranslation } from "@plane/i18n";
import { StartDatePropertyIcon, DueDatePropertyIcon } from "@plane/propel/icons";
import type { IIssueDisplayProperties, TIssue, TSubIssueOperations } from "@plane/types";
import { getDate, renderFormattedPayloadDate } from "@plane/utils/datetime";
import { shouldHighlightIssueDueDate } from "@plane/utils/work-item";
// components
import { DateDropdown } from "@/components/dropdowns/date";
import { IssuePropertyDateRange } from "@/components/issues/issue-layouts/properties/date-range";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { PriorityDropdown } from "@/components/dropdowns/priority";
import { StateDropdown } from "@/components/dropdowns/state/dropdown";
// hooks
import { WithDisplayPropertiesHOC } from "@/components/issues/issue-layouts/properties/with-display-properties-HOC";
import { useIssueAction } from "@/hooks/use-issue-action";
import { useProjectStates } from "@/hooks/use-project-states";

type Props = {
  workspaceSlug: string;
  parentIssueId: string;
  issueId: string;
  canEdit: boolean;
  updateSubIssue: TSubIssueOperations["updateSubIssue"];
  displayProperties?: IIssueDisplayProperties;
  issue: TIssue;
};

export const SubIssuesListItemProperties = function SubIssuesListItemProperties(props: Props) {
  const { workspaceSlug, parentIssueId, issueId, canEdit, updateSubIssue, displayProperties, issue } = props;
  const { t } = useTranslation();
  const { getStateById } = useProjectStates(workspaceSlug, issue.project_id);

  const change = useIssueAction(
    { workspaceSlug, projectId: issue.project_id, issueId, context: `sub-item:${parentIssueId}` },
    canEdit && Boolean(issue.project_id),
    { success: t("sub_work_item.update.success"), error: t("sub_work_item.update.error") }
  );
  const update = (data: Partial<TIssue>) => {
    const projectId = issue.project_id;
    if (projectId) change.run(() => updateSubIssue(workspaceSlug, projectId, parentIssueId, issueId, data));
  };
  const handleStartDate = (date: Date | null) => update({ start_date: date ? renderFormattedPayloadDate(date) : null });
  const handleTargetDate = (date: Date | null) =>
    update({ target_date: date ? renderFormattedPayloadDate(date) : null });

  //derived values
  const stateDetails = getStateById(issue.state_id);
  const shouldHighlight = shouldHighlightIssueDueDate(issue.target_date, stateDetails?.group);
  // date range is enabled only when both dates are available and both dates are enabled
  const isDateRangeEnabled: boolean = Boolean(
    issue.start_date && issue.target_date && displayProperties?.start_date && displayProperties?.due_date
  );

  if (!displayProperties) return <></>;

  const maxDate = getDate(issue.target_date);
  const minDate = getDate(issue.start_date);

  return (
    <div className="relative flex items-center gap-2">
      <WithDisplayPropertiesHOC displayProperties={displayProperties} displayPropertyKey="state">
        <div className="h-5 flex-shrink-0">
          <StateDropdown
            value={issue.state_id}
            projectId={issue.project_id ?? undefined}
            onChange={(val) => update({ state_id: val })}
            disabled={change.disabled}
            buttonVariant="transparent-without-text"
            buttonClassName="hover:bg-transparent px-0"
            iconSize="size-5"
            showTooltip
          />
        </div>
      </WithDisplayPropertiesHOC>

      <WithDisplayPropertiesHOC displayProperties={displayProperties} displayPropertyKey="priority">
        <div className="h-5 flex-shrink-0">
          <PriorityDropdown
            value={issue.priority}
            onChange={(val) => update({ priority: val })}
            disabled={change.disabled}
            buttonVariant="border-without-text"
            showTooltip
          />
        </div>
      </WithDisplayPropertiesHOC>

      {/* merged dates */}
      <WithDisplayPropertiesHOC
        displayProperties={displayProperties}
        displayPropertyKey={["start_date", "due_date"]}
        shouldRenderProperty={() => isDateRangeEnabled}
      >
        <div className="h-5" onFocus={handleEventPropagation} onClick={handleEventPropagation}>
          <IssuePropertyDateRange
            dates={issue}
            onChange={update}
            highlight={shouldHighlight}
            disabled={change.disabled}
            placement="top-end"
            renderInPortal
          />
        </div>
      </WithDisplayPropertiesHOC>

      {/* start date */}
      <WithDisplayPropertiesHOC
        displayProperties={displayProperties}
        displayPropertyKey="start_date"
        shouldRenderProperty={() => !isDateRangeEnabled}
      >
        <div className="h-5">
          <DateDropdown
            value={issue.start_date ?? null}
            onChange={handleStartDate}
            maxDate={maxDate}
            placeholder={t("common.order_by.start_date")}
            icon={<StartDatePropertyIcon className="h-3 w-3 flex-shrink-0" />}
            buttonVariant={issue.start_date ? "border-with-text" : "border-without-text"}
            optionsClassName="z-30"
            disabled={change.disabled}
            showTooltip
          />
        </div>
      </WithDisplayPropertiesHOC>

      {/* target/due date */}
      <WithDisplayPropertiesHOC
        displayProperties={displayProperties}
        displayPropertyKey="due_date"
        shouldRenderProperty={() => !isDateRangeEnabled}
      >
        <div className="h-5">
          <DateDropdown
            value={issue.target_date ?? null}
            onChange={handleTargetDate}
            minDate={minDate}
            placeholder={t("common.order_by.due_date")}
            icon={<DueDatePropertyIcon className="h-3 w-3 flex-shrink-0" />}
            buttonVariant={issue.target_date ? "border-with-text" : "border-without-text"}
            buttonClassName={shouldHighlight ? "text-danger-primary" : ""}
            clearIconClassName="text-primary"
            optionsClassName="z-30"
            disabled={change.disabled}
            showTooltip
          />
        </div>
      </WithDisplayPropertiesHOC>

      <WithDisplayPropertiesHOC displayProperties={displayProperties} displayPropertyKey="assignee">
        <div className="h-5 flex-shrink-0">
          <MemberDropdown
            workspaceSlug={workspaceSlug}
            value={issue.assignee_ids}
            projectId={issue.project_id ?? undefined}
            onChange={(val) => update({ assignee_ids: val })}
            disabled={change.disabled}
            multiple
            buttonVariant={issue.assignee_ids.length > 0 ? "transparent-without-text" : "border-without-text"}
            buttonClassName={issue.assignee_ids.length > 0 ? "hover:bg-transparent px-0" : ""}
          />
        </div>
      </WithDisplayPropertiesHOC>
    </div>
  );
};

function handleEventPropagation(e: SyntheticEvent<HTMLDivElement>) {
  e.stopPropagation();
  e.preventDefault();
}
