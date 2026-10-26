/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// icons
import { DueDatePropertyIcon, StartDatePropertyIcon } from "@plane/propel/icons";
// types
import type { TIssuePriorities, TWorkspaceDraftIssue } from "@plane/types";
import { getDate, renderFormattedPayloadDate } from "@plane/utils/datetime";
import { shouldHighlightIssueDueDate } from "@plane/utils/work-item";
// components
import { CycleDropdown } from "@/components/dropdowns/cycle";
import { DateDropdown } from "@/components/dropdowns/date";
import { EstimateDropdown } from "@/components/dropdowns/estimate";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { ModuleDropdown } from "@/components/dropdowns/module/dropdown";
import { PriorityDropdown } from "@/components/dropdowns/priority";
import { StateDropdown } from "@/components/dropdowns/state/dropdown";
// helpers
// hooks
import { useProjectEstimates } from "@/hooks/use-estimates";
import { useProjectLabels } from "@/hooks/use-labels";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useProjectStates } from "@/hooks/use-project-states";
import { useWorkspaceDraftMutations } from "@/hooks/use-workspace-drafts";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { IssuePropertyLabels } from "../issue-layouts/properties";
// local components

export interface IIssueProperties {
  issue: TWorkspaceDraftIssue;
  updateIssue:
    | ((projectId: string | null, issueId: string, data: Partial<TWorkspaceDraftIssue>) => Promise<void>)
    | undefined;
  className: string;
}

export function DraftIssueProperties(props: IIssueProperties) {
  const { issue, updateIssue, className } = props;
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  // store hooks
  const { isMobile } = usePlatformOS();
  const { getStateById } = useProjectStates(workspaceSlug, issue.project_id);

  // router
  const { projectRole } = useProjectPermissions(workspaceSlug, issue.project_id ?? undefined);
  const projectId = projectRole ? (issue.project_id ?? undefined) : undefined;
  // derived values
  const stateDetails = getStateById(issue.state_id);

  const handleStartDate = (date: Date | null) => {
    if (!updateIssue) return;
    void updateIssue(issue.project_id ?? null, issue.id, {
      start_date: renderFormattedPayloadDate(date) ?? null,
    });
  };

  const handleTargetDate = (date: Date | null) => {
    if (!updateIssue) return;
    void updateIssue(issue.project_id ?? null, issue.id, {
      target_date: renderFormattedPayloadDate(date) ?? null,
    });
  };

  const minDate = getDate(issue.start_date);

  const maxDate = getDate(issue.target_date);

  return (
    <div className={className}>
      <DraftProjectProperties
        issue={issue}
        updateIssue={updateIssue}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        isMobile={isMobile}
      />
      {/* basic properties */}
      <div role="presentation" className="h-5" onClick={stopClickPropagation} onKeyDown={stopKeyPropagation}>
        <PriorityDropdown
          value={issue.priority}
          onChange={(value: TIssuePriorities) => {
            if (issue.project_id) void updateIssue?.(issue.project_id, issue.id, { priority: value });
          }}
          buttonVariant="border-without-text"
          buttonClassName="border"
          renderByDefault={isMobile}
          showTooltip
        />
      </div>

      {/* start date */}
      <div role="presentation" className="h-5" onClick={stopClickPropagation} onKeyDown={stopKeyPropagation}>
        <DateDropdown
          value={issue.start_date ?? null}
          onChange={handleStartDate}
          maxDate={maxDate}
          placeholder="Start date"
          icon={<StartDatePropertyIcon className="h-3 w-3 flex-shrink-0" />}
          buttonVariant={issue.start_date ? "border-with-text" : "border-without-text"}
          optionsClassName="z-10"
          renderByDefault={isMobile}
          showTooltip
        />
      </div>

      {/* target/due date */}
      <div role="presentation" className="h-5" onClick={stopClickPropagation} onKeyDown={stopKeyPropagation}>
        <DateDropdown
          value={issue.target_date ?? null}
          onChange={handleTargetDate}
          minDate={minDate}
          placeholder="Due date"
          icon={<DueDatePropertyIcon className="h-3 w-3 flex-shrink-0" />}
          buttonVariant={issue.target_date ? "border-with-text" : "border-without-text"}
          buttonClassName={
            shouldHighlightIssueDueDate(issue.target_date || null, stateDetails?.group) ? "text-danger-primary" : ""
          }
          clearIconClassName="!text-primary"
          optionsClassName="z-10"
          renderByDefault={isMobile}
          showTooltip
        />
      </div>
    </div>
  );
}

function DraftProjectProperties({
  issue,
  updateIssue,
  workspaceSlug,
  projectId,
  isMobile,
}: {
  issue: TWorkspaceDraftIssue;
  updateIssue: IIssueProperties["updateIssue"];
  workspaceSlug: string;
  projectId: string | undefined;
  isMobile: boolean;
}) {
  return (
    <>
      <DraftAssignmentProperties
        issue={issue}
        updateIssue={updateIssue}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        isMobile={isMobile}
      />
      <DraftPlanningProperties
        issue={issue}
        updateIssue={updateIssue}
        workspaceSlug={workspaceSlug}
        isMobile={isMobile}
      />
    </>
  );
}

function DraftAssignmentProperties({
  issue,
  updateIssue,
  workspaceSlug,
  projectId,
  isMobile,
}: {
  issue: TWorkspaceDraftIssue;
  updateIssue: IIssueProperties["updateIssue"];
  workspaceSlug: string;
  projectId: string | undefined;
  isMobile: boolean;
}) {
  const { labelsById } = useProjectLabels(workspaceSlug, issue.project_id);
  const defaultLabelOptions = issue.label_ids?.flatMap((id) => (labelsById?.[id] ? [labelsById[id]] : [])) ?? [];

  return (
    <>
      {projectId && (
        <div role="presentation" className="h-5" onClick={stopClickPropagation} onKeyDown={stopKeyPropagation}>
          <StateDropdown
            buttonContainerClassName="truncate max-w-40"
            value={issue.state_id}
            onChange={(stateId) => {
              if (issue.project_id) void updateIssue?.(issue.project_id, issue.id, { state_id: stateId });
            }}
            projectId={projectId}
            buttonVariant="border-with-text"
            renderByDefault={isMobile}
            showTooltip
          />
        </div>
      )}
      {projectId && (
        <IssuePropertyLabels
          projectId={issue.project_id || null}
          value={issue.label_ids || null}
          defaultOptions={defaultLabelOptions}
          onChange={(ids) => {
            if (issue.project_id) void updateIssue?.(issue.project_id, issue.id, { label_ids: ids });
          }}
          renderByDefault={isMobile}
          hideDropdownArrow
        />
      )}
      {projectId && (
        <div role="presentation" className="h-5" onClick={stopClickPropagation} onKeyDown={stopKeyPropagation}>
          <MemberDropdown
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            value={issue.assignee_ids}
            onChange={(ids) => {
              if (issue.project_id) void updateIssue?.(issue.project_id, issue.id, { assignee_ids: ids });
            }}
            multiple
            buttonVariant={issue.assignee_ids?.length ? "transparent-without-text" : "border-without-text"}
            buttonClassName={issue.assignee_ids?.length ? "hover:bg-transparent px-0" : ""}
            showTooltip={!issue.assignee_ids?.length}
            placeholder="Assignees"
            optionsClassName="z-10"
            tooltipContent=""
            renderByDefault={isMobile}
          />
        </div>
      )}
    </>
  );
}

function DraftPlanningProperties({
  issue,
  updateIssue,
  workspaceSlug,
  isMobile,
}: {
  issue: TWorkspaceDraftIssue;
  updateIssue: IIssueProperties["updateIssue"];
  workspaceSlug: string;
  isMobile: boolean;
}) {
  const projectCatalogue = useWorkspaceProjectDetails();
  const project = projectCatalogue.projects?.find((row) => row.id === issue.project_id);
  const { updateDraft } = useWorkspaceDraftMutations(workspaceSlug);
  const { isEnabled: estimateEnabled } = useProjectEstimates(workspaceSlug, issue.project_id);

  return (
    <>
      {project?.module_view && (
        <div role="presentation" className="h-5" onClick={stopClickPropagation} onKeyDown={stopKeyPropagation}>
          <ModuleDropdown
            buttonContainerClassName="truncate max-w-40"
            projectId={issue.project_id ?? undefined}
            value={issue.module_ids ?? []}
            onChange={(moduleIds) => {
              if (issue.module_ids && moduleIds) void updateDraft(issue.id, { module_ids: moduleIds });
            }}
            renderByDefault={isMobile}
            multiple
            buttonVariant="border-with-text"
            showCount
            showTooltip
          />
        </div>
      )}
      {project?.cycle_view && (
        <div role="presentation" className="h-5" onClick={stopClickPropagation} onKeyDown={stopKeyPropagation}>
          <CycleDropdown
            buttonContainerClassName="truncate max-w-40"
            projectId={issue.project_id ?? undefined}
            value={issue.cycle_id || null}
            onChange={(cycleId) => {
              if (issue.cycle_id === cycleId) return;
              void updateDraft(issue.id, { cycle_id: cycleId ?? "" });
            }}
            buttonVariant="border-with-text"
            renderByDefault={isMobile}
            showTooltip
          />
        </div>
      )}
      {estimateEnabled && issue.project_id && (
        <div role="presentation" className="h-5" onClick={stopClickPropagation} onKeyDown={stopKeyPropagation}>
          <EstimateDropdown
            value={issue.estimate_point ?? undefined}
            onChange={(value) => {
              void updateIssue?.(issue.project_id ?? null, issue.id, { estimate_point: value ?? null });
            }}
            projectId={issue.project_id}
            buttonVariant="border-with-text"
            renderByDefault={isMobile}
            showTooltip
          />
        </div>
      )}
    </>
  );
}

function stopClickPropagation(event: React.MouseEvent) {
  event.stopPropagation();
  event.preventDefault();
}

function stopKeyPropagation(event: React.KeyboardEvent) {
  event.stopPropagation();
}
