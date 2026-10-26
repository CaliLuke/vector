/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
import {
  StatePropertyIcon,
  MembersPropertyIcon,
  PriorityPropertyIcon,
  DueDatePropertyIcon,
  LabelPropertyIcon,
  DuplicatePropertyIcon,
} from "@plane/propel/icons";
import { Tooltip } from "@plane/propel/tooltip";
import type { TInboxDuplicateIssueDetails, TIssue } from "@plane/types";
import { ControlLink } from "@plane/ui";
import { getDate, renderFormattedPayloadDate } from "@plane/utils/datetime";
import { generateWorkItemLink } from "@plane/utils/work-item";
// components
import { DateDropdown } from "@/components/dropdowns/date";
import { IntakeStateDropdown } from "@/components/dropdowns/intake-state/dropdown";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { PriorityDropdown } from "@/components/dropdowns/priority";
import { StateDropdown } from "@/components/dropdowns/state/dropdown";
import type { TIssueOperations } from "@/components/issues/issue-detail";
import { IssueLabel } from "@/components/issues/issue-detail/label";
// hooks
import { useProjectDetail } from "@/hooks/use-projects";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issue: Partial<TIssue>;
  issueOperations: TIssueOperations;
  isEditable: boolean;
  duplicateIssueDetails: TInboxDuplicateIssueDetails | undefined;
  isIntakeAccepted: boolean;
};

export const InboxIssueContentProperties = function InboxIssueContentProperties(props: Props) {
  const { workspaceSlug, projectId, issue, issueOperations, isEditable, duplicateIssueDetails, isIntakeAccepted } =
    props;

  const minDate = issue.start_date ? getDate(issue.start_date) : null;
  if (!issue.id) return null;
  const issueId = issue.id;
  const assigneeIds = issue.assignee_ids ?? [];

  const DropdownComponent = isIntakeAccepted ? StateDropdown : IntakeStateDropdown;
  const updateProperty = (patch: Partial<TIssue>) => {
    // The operation reports failures; picker callbacks are synchronous event boundaries.
    void issueOperations.update(workspaceSlug, projectId, issueId, patch).catch(() => {});
  };

  return (
    <div className="flex w-full flex-col divide-y-2 divide-subtle-1">
      <div className="w-full overflow-y-auto">
        <h5 className="mb-2 text-body-sm-medium">Properties</h5>
        <div className={`divide-y-2 divide-subtle-1 ${!isEditable ? "opacity-60" : ""}`}>
          <div className="flex flex-col gap-3">
            {/* Intake State */}
            <div className="flex h-8 items-center gap-2">
              <div className="flex w-2/5 flex-shrink-0 items-center gap-1 text-13 text-tertiary">
                <StatePropertyIcon className="h-4 w-4 flex-shrink-0" />
                <span>State</span>
              </div>
              {issue.state_id && (
                <DropdownComponent
                  value={issue.state_id}
                  onChange={() => {}}
                  projectId={projectId}
                  disabled
                  buttonVariant="transparent-with-text"
                  className="group w-3/5 flex-grow"
                  buttonContainerClassName="w-full text-left"
                  buttonClassName="text-13"
                  dropdownArrow
                  dropdownArrowClassName="h-3.5 w-3.5 hidden group-hover:inline"
                />
              )}
            </div>
            {/* Assignee */}
            <div className="flex h-8 items-center gap-2">
              <div className="flex w-2/5 flex-shrink-0 items-center gap-1 text-13 text-tertiary">
                <MembersPropertyIcon className="h-4 w-4 flex-shrink-0" />
                <span>Assignees</span>
              </div>
              <MemberDropdown
                workspaceSlug={workspaceSlug}
                value={assigneeIds}
                onChange={(val) => updateProperty({ assignee_ids: val })}
                disabled={!isEditable}
                projectId={projectId}
                placeholder="Add assignees"
                multiple
                buttonVariant={assigneeIds.length > 0 ? "transparent-without-text" : "transparent-with-text"}
                className="group w-3/5 flex-grow"
                buttonContainerClassName="w-full text-left"
                buttonClassName={`text-13 justify-between ${assigneeIds.length > 0 ? "" : "text-placeholder"}`}
                hideIcon={assigneeIds.length === 0}
                dropdownArrow
                dropdownArrowClassName="h-3.5 w-3.5 hidden group-hover:inline"
              />
            </div>
            {/* Priority */}
            <div className="flex h-8 items-center gap-2">
              <div className="flex w-2/5 flex-shrink-0 items-center gap-1 text-13 text-tertiary">
                <PriorityPropertyIcon className="h-4 w-4 flex-shrink-0" />
                <span>Priority</span>
              </div>
              <PriorityDropdown
                value={issue.priority}
                onChange={(val) => updateProperty({ priority: val })}
                disabled={!isEditable}
                buttonVariant="border-with-text"
                className="w-3/5 flex-grow rounded-sm px-2 hover:bg-layer-1"
                buttonContainerClassName="w-full text-left"
                buttonClassName="w-min h-auto whitespace-nowrap"
              />
            </div>
          </div>
        </div>
        <div className={`mt-3 divide-y-2 divide-subtle-1 ${!isEditable ? "opacity-60" : ""}`}>
          <div className="flex flex-col gap-3">
            {/* Due Date */}
            <div className="flex h-8 items-center gap-2">
              <div className="flex w-2/5 flex-shrink-0 items-center gap-1 text-13 text-tertiary">
                <DueDatePropertyIcon className="h-4 w-4 flex-shrink-0" />
                <span>Due date</span>
              </div>
              <DateDropdown
                placeholder="Add due date"
                value={issue.target_date || null}
                onChange={(val) =>
                  updateProperty({
                    target_date: val ? renderFormattedPayloadDate(val) : null,
                  })
                }
                minDate={minDate ?? undefined}
                disabled={!isEditable}
                buttonVariant="transparent-with-text"
                className="group w-3/5 flex-grow"
                buttonContainerClassName="w-full text-left"
                buttonClassName={`text-13 ${issue.target_date ? "" : "text-placeholder"}`}
                hideIcon
                clearIconClassName="h-3 w-3 hidden group-hover:inline"
              />
            </div>
            {/* Labels */}
            <div className="flex min-h-8 items-center gap-2">
              <div className="flex w-2/5 flex-shrink-0 items-center gap-1 text-13 text-tertiary">
                <LabelPropertyIcon className="h-4 w-4 flex-shrink-0" />
                <span>Labels</span>
              </div>
              <div className="h-full min-h-8 w-3/5 flex-grow pt-1">
                <IssueLabel
                  workspaceSlug={workspaceSlug}
                  projectId={projectId}
                  issueId={issueId}
                  disabled={!isEditable}
                  isInboxIssue
                  labelIds={issue.label_ids ?? []}
                  onLabelUpdate={async (val: string[]) => {
                    await issueOperations.update(workspaceSlug, projectId, issueId, { label_ids: val });
                  }}
                />
              </div>
            </div>

            {duplicateIssueDetails && (
              <DuplicateWorkItemLink
                workspaceSlug={workspaceSlug}
                projectId={projectId}
                duplicateIssueDetails={duplicateIssueDetails}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

function DuplicateWorkItemLink({
  workspaceSlug,
  projectId,
  duplicateIssueDetails,
}: {
  workspaceSlug: string;
  projectId: string;
  duplicateIssueDetails: TInboxDuplicateIssueDetails;
}) {
  const router = useRouter();
  const { data: currentProjectDetails } = useProjectDetail(workspaceSlug, projectId);
  const duplicateWorkItemLink = generateWorkItemLink({
    workspaceSlug: workspaceSlug,
    projectId,
    issueId: duplicateIssueDetails.id,
    projectIdentifier: currentProjectDetails?.identifier,
    sequenceId: duplicateIssueDetails.sequence_id,
  });
  return (
    <div className="flex min-h-8 gap-2">
      <div className="flex w-2/5 flex-shrink-0 gap-1 pt-2 text-13 text-tertiary">
        <DuplicatePropertyIcon className="h-4 w-4 flex-shrink-0" />
        <span>Duplicate of</span>
      </div>

      <ControlLink
        href={duplicateWorkItemLink}
        onClick={() => {
          void router.navigate({ href: duplicateWorkItemLink });
        }}
        target="_self"
      >
        <Tooltip tooltipContent={`${duplicateIssueDetails.name}`}>
          <span className="flex cursor-pointer items-center gap-1 rounded-sm bg-layer-1 px-1.5 py-1 pb-0.5 text-11 text-secondary">
            {`${currentProjectDetails?.identifier}-${duplicateIssueDetails.sequence_id}`}
          </span>
        </Tooltip>
      </ControlLink>
    </div>
  );
}
