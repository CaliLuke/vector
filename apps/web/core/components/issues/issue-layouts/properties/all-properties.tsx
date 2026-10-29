/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import type { SyntheticEvent } from "react";

// icons
import { Paperclip } from "lucide-react";
// i18n
import { useTranslation } from "@plane/i18n";
import { LinkIcon, StartDatePropertyIcon, ViewsIcon, DueDatePropertyIcon } from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { TIssue, IIssueDisplayProperties, TIssuePriorities } from "@plane/types";
// ui
import { cn } from "@plane/utils/common";
import { getDate, renderFormattedPayloadDate } from "@plane/utils/datetime";
import { generateWorkItemLink, shouldHighlightIssueDueDate } from "@plane/utils/work-item";
// components
import { CycleDropdown } from "@/components/dropdowns/cycle";
import { DateDropdown } from "@/components/dropdowns/date";
import { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { EstimateDropdown } from "@/components/dropdowns/estimate";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { ModuleDropdown } from "@/components/dropdowns/module/dropdown";
import { PriorityDropdown } from "@/components/dropdowns/priority";
import { StateDropdown } from "@/components/dropdowns/state/dropdown";
// hooks
import { useProjectEstimates } from "@/hooks/use-estimates";
import { useProjectLabels } from "@/hooks/use-labels";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useProjectStates } from "@/hooks/use-project-states";

import { useIssueListType } from "@/hooks/use-issue-list-type";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useIssuePropertyChange } from "@/hooks/use-issue-property-change";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local components
import { IssuePropertyLabels } from "./labels";
import { WithDisplayPropertiesHOC } from "./with-display-properties-HOC";

function handleEventPropagation(event: SyntheticEvent<HTMLDivElement>) {
  event.stopPropagation();
  event.preventDefault();
}

export interface IIssueProperties {
  issue: TIssue;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<TIssue>) => Promise<unknown>) | undefined;
  displayProperties: IIssueDisplayProperties | undefined;
  isReadOnly: boolean;
  className: string;
  activeLayout: string;
  isEpic?: boolean;
}

export function IssueProperties(props: IIssueProperties) {
  const { issue, updateIssue, displayProperties, isReadOnly, className, isEpic = false } = props;
  const { workspaceSlug } = useParams({ strict: false });
  // i18n
  const { t } = useTranslation();
  // store hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const { labelsById } = useProjectLabels(workspaceSlug, issue.project_id);
  const storeType = useIssueListType();
  const actions = useIssuesActions(storeType);
  const { isEnabled: estimateEnabled } = useProjectEstimates(workspaceSlug, issue.project_id);
  const { getStateById } = useProjectStates(workspaceSlug, issue.project_id);
  const { isMobile } = usePlatformOS();
  const projectDetails = projectRows?.find((project) => project.id === issue.project_id);

  // router
  const router = useRouter();

  // derived values
  const stateDetails = getStateById(issue.state_id);
  const subIssueCount = issue.sub_issues_count ?? 0;

  const { run, disabled } = useIssuePropertyChange(
    { workspaceSlug, projectId: issue.project_id, issueId: issue.id, context: storeType },
    !isReadOnly && Boolean(workspaceSlug && issue.project_id && issue.id)
  );
  const isUpdateDisabled = disabled || !updateIssue;

  const handleUpdate = (data: Partial<TIssue>) => {
    if (updateIssue) run(() => updateIssue(issue.project_id, issue.id, data));
  };
  const handleState = (stateId: string) => handleUpdate({ state_id: stateId });
  const handlePriority = (priority: TIssuePriorities) => handleUpdate({ priority });
  const handleLabel = (label_ids: string[]) => handleUpdate({ label_ids });
  const handleAssignee = (assignee_ids: string[]) => handleUpdate({ assignee_ids });

  const handleModule = (moduleIds: string[] | null) => {
    if (!moduleIds) return;
    const selected = new Set(moduleIds);
    const current = new Set(issue.module_ids ?? []);
    const added = [...selected].filter((id) => !current.has(id));
    const removed = [...current].filter((id) => !selected.has(id));
    if (added.length || removed.length)
      run(() => actions.changeModulesInIssue(issue.project_id, issue.id, added, removed));
  };

  const handleCycle = (cycleId: string | null) => {
    if (issue.cycle_id === cycleId) return;
    if (cycleId) run(() => actions.addCycleToIssue(issue.project_id, cycleId, issue.id));
    else if (issue.cycle_id) {
      const previousCycleId = issue.cycle_id;
      run(() => actions.removeCycleFromIssue(issue.project_id, previousCycleId, issue.id));
    }
  };

  const handleStartDate = (date: Date | null) =>
    handleUpdate({ start_date: date ? renderFormattedPayloadDate(date) : null });
  const handleTargetDate = (date: Date | null) =>
    handleUpdate({ target_date: date ? renderFormattedPayloadDate(date) : null });
  const handleEstimate = (estimate_point: string | undefined) => handleUpdate({ estimate_point });

  const workItemLink = generateWorkItemLink({
    workspaceSlug,
    projectId: issue.project_id,
    issueId: issue.id,
    projectIdentifier: projectDetails?.identifier,
    sequenceId: issue.sequence_id,
    isArchived: !!issue.archived_at,
    isEpic,
  });

  const redirectToIssueDetail = () => void router.navigate({ href: `${workItemLink}#sub-issues` });

  if (!displayProperties || !issue.project_id) return null;

  // date range is enabled only when both dates are available and both dates are enabled
  const isDateRangeEnabled: boolean = Boolean(
    issue.start_date && issue.target_date && displayProperties.start_date && displayProperties.due_date
  );

  const defaultLabelOptions =
    issue.label_ids?.flatMap((id) => {
      const label = labelsById?.get(id);
      return label ? [label] : [];
    }) || [];

  const minDate = getDate(issue.start_date);
  const maxDate = getDate(issue.target_date);

  return (
    <div className={className}>
      {/* basic properties */}
      {/* state */}
      <WithDisplayPropertiesHOC displayProperties={displayProperties} displayPropertyKey="state">
        {/* oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
        <div className="h-5" onFocus={handleEventPropagation} onClick={handleEventPropagation}>
          <StateDropdown
            buttonContainerClassName="truncate max-w-40"
            value={issue.state_id}
            onChange={handleState}
            projectId={issue.project_id}
            disabled={isUpdateDisabled}
            buttonVariant="border-with-text"
            renderByDefault={isMobile}
            showTooltip
          />
        </div>
      </WithDisplayPropertiesHOC>

      {/* priority */}
      <WithDisplayPropertiesHOC displayProperties={displayProperties} displayPropertyKey="priority">
        {/* oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
        <div className="h-5" onFocus={handleEventPropagation} onClick={handleEventPropagation}>
          <PriorityDropdown
            value={issue.priority}
            onChange={handlePriority}
            disabled={isUpdateDisabled}
            buttonVariant="border-without-text"
            renderByDefault={isMobile}
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
        {/* oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
        <div className="h-5" onFocus={handleEventPropagation} onClick={handleEventPropagation}>
          <DateRangeDropdown
            value={{
              from: getDate(issue.start_date) || undefined,
              to: getDate(issue.target_date) || undefined,
            }}
            onSelect={(range) =>
              handleUpdate({
                start_date: range?.from ? renderFormattedPayloadDate(range.from) : null,
                target_date: range?.to ? renderFormattedPayloadDate(range.to) : null,
              })
            }
            hideIcon={{
              from: false,
            }}
            isClearable
            mergeDates
            buttonVariant={issue.start_date || issue.target_date ? "border-with-text" : "border-without-text"}
            buttonClassName={
              shouldHighlightIssueDueDate(issue.target_date, stateDetails?.group) ? "text-danger-primary" : ""
            }
            clearIconClassName="text-primary!"
            disabled={isUpdateDisabled}
            renderByDefault={isMobile}
            showTooltip
            renderPlaceholder={false}
            customTooltipHeading="Date Range"
          />
        </div>
      </WithDisplayPropertiesHOC>

      {/* start date */}
      <WithDisplayPropertiesHOC
        displayProperties={displayProperties}
        displayPropertyKey="start_date"
        shouldRenderProperty={() => !isDateRangeEnabled}
      >
        {/* oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
        <div className="h-5" onFocus={handleEventPropagation} onClick={handleEventPropagation}>
          <DateDropdown
            value={issue.start_date ?? null}
            onChange={handleStartDate}
            maxDate={maxDate}
            placeholder={t("common.order_by.start_date")}
            icon={<StartDatePropertyIcon className="h-3 w-3 flex-shrink-0" />}
            buttonVariant={issue.start_date ? "border-with-text" : "border-without-text"}
            optionsClassName="z-10"
            disabled={isUpdateDisabled}
            renderByDefault={isMobile}
            showTooltip
            labelClassName="text-caption-sm-regular"
          />
        </div>
      </WithDisplayPropertiesHOC>

      {/* target/due date */}
      <WithDisplayPropertiesHOC
        displayProperties={displayProperties}
        displayPropertyKey="due_date"
        shouldRenderProperty={() => !isDateRangeEnabled}
      >
        {/* oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
        <div className="h-5" onFocus={handleEventPropagation} onClick={handleEventPropagation}>
          <DateDropdown
            value={issue.target_date ?? null}
            onChange={handleTargetDate}
            minDate={minDate}
            placeholder={t("common.order_by.due_date")}
            icon={<DueDatePropertyIcon className="h-3 w-3 shrink-0" />}
            buttonVariant={issue.target_date ? "border-with-text" : "border-without-text"}
            buttonClassName={
              shouldHighlightIssueDueDate(issue.target_date, stateDetails?.group) ? "text-danger-primary" : ""
            }
            clearIconClassName="text-primary!"
            optionsClassName="z-10"
            disabled={isUpdateDisabled}
            renderByDefault={isMobile}
            showTooltip
            labelClassName="text-caption-sm-regular"
          />
        </div>
      </WithDisplayPropertiesHOC>

      {/* assignee */}
      <WithDisplayPropertiesHOC displayProperties={displayProperties} displayPropertyKey="assignee">
        {/* oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
        <div className="h-5" onFocus={handleEventPropagation} onClick={handleEventPropagation}>
          <MemberDropdown
            workspaceSlug={workspaceSlug}
            projectId={issue.project_id}
            value={issue.assignee_ids}
            onChange={handleAssignee}
            disabled={isUpdateDisabled}
            multiple
            buttonVariant={issue.assignee_ids?.length > 0 ? "transparent-without-text" : "border-without-text"}
            buttonClassName={issue.assignee_ids?.length > 0 ? "hover:bg-transparent px-0" : ""}
            showTooltip={issue.assignee_ids?.length === 0}
            placeholder={t("common.assignees")}
            optionsClassName="z-10"
            tooltipContent=""
            renderByDefault={isMobile}
          />
        </div>
      </WithDisplayPropertiesHOC>

      <>
        {!isEpic && (
          <>
            {/* modules */}
            {projectDetails?.module_view && (
              <WithDisplayPropertiesHOC displayProperties={displayProperties} displayPropertyKey="modules">
                {/* oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
                <div className="h-5" onFocus={handleEventPropagation} onClick={handleEventPropagation}>
                  <ModuleDropdown
                    buttonContainerClassName="truncate max-w-40"
                    projectId={issue.project_id}
                    value={issue.module_ids ?? []}
                    onChange={handleModule}
                    disabled={disabled}
                    renderByDefault={isMobile}
                    multiple
                    buttonVariant="border-with-text"
                    showCount
                    showTooltip
                  />
                </div>
              </WithDisplayPropertiesHOC>
            )}

            {/* cycles */}
            {projectDetails?.cycle_view && (
              <WithDisplayPropertiesHOC displayProperties={displayProperties} displayPropertyKey="cycle">
                {/* oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
                <div className="h-5" onFocus={handleEventPropagation} onClick={handleEventPropagation}>
                  <CycleDropdown
                    buttonContainerClassName="truncate max-w-40"
                    projectId={issue.project_id}
                    value={issue.cycle_id}
                    onChange={handleCycle}
                    disabled={disabled}
                    buttonVariant="border-with-text"
                    renderByDefault={isMobile}
                    showTooltip
                  />
                </div>
              </WithDisplayPropertiesHOC>
            )}
          </>
        )}
      </>

      {/* estimates */}
      {estimateEnabled && (
        <WithDisplayPropertiesHOC displayProperties={displayProperties} displayPropertyKey="estimate">
          {/* oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
          <div className="h-5" onFocus={handleEventPropagation} onClick={handleEventPropagation}>
            <EstimateDropdown
              value={issue.estimate_point ?? undefined}
              onChange={handleEstimate}
              projectId={issue.project_id}
              disabled={isUpdateDisabled}
              buttonVariant="border-with-text"
              renderByDefault={isMobile}
              showTooltip
            />
          </div>
        </WithDisplayPropertiesHOC>
      )}

      <IssueSummaryProperties
        issue={issue}
        displayProperties={displayProperties}
        isEpic={isEpic}
        isMobile={isMobile}
        subIssueCount={subIssueCount}
        onPropertyEvent={handleEventPropagation}
        onOpenSubIssues={redirectToIssueDetail}
      />

      {/* label */}
      <WithDisplayPropertiesHOC displayProperties={displayProperties} displayPropertyKey="labels">
        <IssuePropertyLabels
          projectId={issue.project_id || null}
          value={issue.label_ids || []}
          defaultOptions={defaultLabelOptions}
          onChange={handleLabel}
          disabled={isUpdateDisabled}
          renderByDefault={isMobile}
          hideDropdownArrow
          maxRender={3}
        />
      </WithDisplayPropertiesHOC>
    </div>
  );
}

type IssueSummaryPropertiesProps = {
  issue: TIssue;
  displayProperties: IIssueDisplayProperties;
  isEpic: boolean;
  isMobile: boolean;
  subIssueCount: number;
  onPropertyEvent: (event: SyntheticEvent<HTMLDivElement>) => void;
  onOpenSubIssues: () => void;
};

function IssueSummaryProperties(props: IssueSummaryPropertiesProps) {
  const { issue, displayProperties, isEpic, isMobile, subIssueCount, onPropertyEvent, onOpenSubIssues } = props;
  const { t } = useTranslation();

  return (
    <>
      {!isEpic && (
        <WithDisplayPropertiesHOC
          displayProperties={displayProperties}
          displayPropertyKey="sub_issue_count"
          shouldRenderProperty={(properties) => !!properties.sub_issue_count && !!subIssueCount}
        >
          <Tooltip
            tooltipHeading={t("common.sub_work_items")}
            tooltipContent={`${subIssueCount}`}
            isMobile={isMobile}
            renderByDefault={false}
          >
            {/* oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
            <div
              onFocus={onPropertyEvent}
              onClick={(event) => {
                event.stopPropagation();
                event.preventDefault();
                if (subIssueCount) onOpenSubIssues();
              }}
              className={cn(
                "flex h-5 flex-shrink-0 items-center justify-center gap-2 overflow-hidden rounded-sm border-[0.5px] border-strong px-2.5 py-1",
                { "cursor-pointer hover:bg-layer-1": subIssueCount }
              )}
            >
              <ViewsIcon className="h-3 w-3 flex-shrink-0" strokeWidth={2} />
              <div className="text-caption-sm-regular">{subIssueCount}</div>
            </div>
          </Tooltip>
        </WithDisplayPropertiesHOC>
      )}

      <WithDisplayPropertiesHOC
        displayProperties={displayProperties}
        displayPropertyKey="attachment_count"
        shouldRenderProperty={(properties) => !!properties.attachment_count && !!issue.attachment_count}
      >
        <Tooltip
          tooltipHeading={t("common.attachments")}
          tooltipContent={`${issue.attachment_count}`}
          isMobile={isMobile}
          renderByDefault={false}
        >
          {/* oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
          <div
            className="flex h-5 flex-shrink-0 items-center justify-center gap-2 overflow-hidden rounded-sm border-[0.5px] border-strong px-2.5 py-1"
            onFocus={onPropertyEvent}
            onClick={onPropertyEvent}
          >
            <Paperclip className="h-3 w-3 flex-shrink-0" strokeWidth={2} />
            <div className="text-caption-sm-regular">{issue.attachment_count}</div>
          </div>
        </Tooltip>
      </WithDisplayPropertiesHOC>

      <WithDisplayPropertiesHOC
        displayProperties={displayProperties}
        displayPropertyKey="link"
        shouldRenderProperty={(properties) => !!properties.link && !!issue.link_count}
      >
        <Tooltip
          tooltipHeading={t("common.links")}
          tooltipContent={`${issue.link_count}`}
          isMobile={isMobile}
          renderByDefault={false}
        >
          {/* oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
          <div
            className="flex h-5 flex-shrink-0 items-center justify-center gap-2 overflow-hidden rounded-sm border-[0.5px] border-strong px-2.5 py-1"
            onFocus={onPropertyEvent}
            onClick={onPropertyEvent}
          >
            <LinkIcon className="h-3 w-3 flex-shrink-0" strokeWidth={2} />
            <div className="text-caption-sm-regular">{issue.link_count}</div>
          </div>
        </Tooltip>
      </WithDisplayPropertiesHOC>
    </>
  );
}
