/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// i18n
import { useTranslation } from "@plane/i18n";
// ui icons
import {
  StatePropertyIcon,
  MembersPropertyIcon,
  PriorityPropertyIcon,
  StartDatePropertyIcon,
  DueDatePropertyIcon,
  LabelPropertyIcon,
  UserCirclePropertyIcon,
  ParentPropertyIcon,
} from "@plane/propel/icons";
import { cn } from "@plane/utils/common";
import { getDate, renderFormattedPayloadDate } from "@plane/utils/datetime";
import { getFileURL } from "@plane/utils/file";
import { shouldHighlightIssueDueDate } from "@plane/utils/work-item";
import { Avatar } from "@plane/ui";
// components
import { ItemCustomProperties } from "@/components/work-item-types/item-custom-properties";
import { DateDropdown } from "@/components/dropdowns/date";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { PriorityDropdown } from "@/components/dropdowns/priority";
import { StateDropdown } from "@/components/dropdowns/state/dropdown";
import { SidebarPropertyListItem } from "@/components/common/layout/sidebar/property-list-item";
// helpers
import { useIssuePropertyChange } from "@/hooks/use-issue-property-change";
import { useWorkItemDetailRecord } from "@/providers/work-item-detail-record";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkspaceStates } from "@/hooks/use-project-states";
// plane web components
import { IssueParentSelectRoot } from "@/components/issues/parent-select-root";
import type { TIssue, TStateGroups, IUserLite } from "@plane/types";
import type { TIssueOperations } from "../issue-detail";
import { IssueLabel } from "../issue-detail/label";
import { IssueProjectProperties as SharedIssueProjectProperties } from "../issue-detail/project-properties";

interface IPeekOverviewProperties {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  disabled: boolean;
  issueOperations: TIssueOperations;
}

type PeekPropertyGroupProps = {
  onUpdate: (data: Partial<TIssue>) => void;
  propertyDisabled: boolean;
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueOperations: TIssueOperations;
  disabled: boolean;
  issue: TIssue;
  createdByDetails?: IUserLite;
  minDate: Date | undefined;
  maxDate: Date | undefined;
  stateGroup: TStateGroups | undefined;
  isEstimateEnabled: string | null | undefined;
  moduleView: boolean | undefined;
  cycleView: boolean | undefined;
};

function PeekCreatedByProperty({ member }: { member: IUserLite }) {
  const { t } = useTranslation();
  const isIntake = member.display_name?.includes("-intake");
  return (
    <SidebarPropertyListItem icon={UserCirclePropertyIcon} label={t("common.created_by")} childrenClassName="px-2">
      {isIntake ? (
        <MembersPropertyIcon className="mx-[4px] h-3 w-3 flex-shrink-0" />
      ) : (
        <Avatar src={getFileURL(member.avatar_url ?? "")} name={member.display_name} size="md" showTooltip={false} />
      )}
      <span className="grow truncate text-body-xs-medium leading-5 text-secondary">
        {isIntake ? "Vector" : member.display_name}
      </span>
    </SidebarPropertyListItem>
  );
}

const PeekBasicProperties = function PeekBasicProperties(props: PeekPropertyGroupProps) {
  const { workspaceSlug, projectId, onUpdate, propertyDisabled, issue, createdByDetails } = props;
  const { t } = useTranslation();
  return (
    <>
      <SidebarPropertyListItem icon={StatePropertyIcon} label={t("common.state")}>
        <StateDropdown
          value={issue.state_id}
          onChange={(val) => onUpdate({ state_id: val })}
          projectId={projectId}
          disabled={propertyDisabled}
          buttonVariant="transparent-with-text"
          className="group w-full grow"
          buttonContainerClassName="w-full text-left h-7.5"
          buttonClassName={`text-body-xs-medium ${issue.state_id ? "" : "text-placeholder"}`}
          dropdownArrow
          dropdownArrowClassName="h-3.5 w-3.5 hidden group-hover:inline"
        />
      </SidebarPropertyListItem>

      <SidebarPropertyListItem icon={MembersPropertyIcon} label={t("common.assignees")}>
        <MemberDropdown
          workspaceSlug={workspaceSlug}
          value={issue.assignee_ids ?? undefined}
          onChange={(val) => onUpdate({ assignee_ids: val })}
          disabled={propertyDisabled}
          projectId={projectId}
          placeholder={t("issue.add.assignee")}
          multiple
          buttonVariant={issue.assignee_ids?.length > 1 ? "transparent-without-text" : "transparent-with-text"}
          className="group w-full grow"
          buttonContainerClassName="w-full text-left h-7.5"
          buttonClassName={`text-body-xs-medium justify-between ${issue.assignee_ids?.length > 0 ? "" : "text-placeholder"}`}
          hideIcon={issue.assignee_ids?.length === 0}
          dropdownArrow
          dropdownArrowClassName="h-3.5 w-3.5 hidden group-hover:inline"
        />
      </SidebarPropertyListItem>

      <SidebarPropertyListItem icon={PriorityPropertyIcon} label={t("common.priority")}>
        <PriorityDropdown
          value={issue.priority}
          onChange={(val) => onUpdate({ priority: val })}
          disabled={propertyDisabled}
          buttonVariant="transparent-with-text"
          className="h-7.5 w-full grow rounded-sm"
          buttonContainerClassName="w-full text-left h-7.5"
          buttonClassName={`text-body-xs-medium whitespace-nowrap [&_svg]:size-3.5 ${!issue.priority || issue.priority === "none" ? "text-placeholder" : ""}`}
        />
      </SidebarPropertyListItem>

      {createdByDetails && <PeekCreatedByProperty member={createdByDetails} />}
    </>
  );
};

const PeekDateProperties = function PeekDateProperties(props: PeekPropertyGroupProps) {
  const { onUpdate, propertyDisabled, issue, minDate, maxDate, stateGroup } = props;
  const { t } = useTranslation();
  return (
    <>
      <SidebarPropertyListItem icon={StartDatePropertyIcon} label={t("common.order_by.start_date")}>
        <DateDropdown
          value={issue.start_date}
          onChange={(val) =>
            onUpdate({
              start_date: val ? renderFormattedPayloadDate(val) : null,
            })
          }
          placeholder={t("issue.add.start_date")}
          buttonVariant="transparent-with-text"
          maxDate={maxDate ?? undefined}
          disabled={propertyDisabled}
          className="group w-full grow"
          buttonContainerClassName="w-full text-left h-7.5"
          buttonClassName={`text-body-xs-medium ${issue.start_date ? "" : "text-placeholder"}`}
          hideIcon
          clearIconClassName="h-3 w-3 hidden group-hover:inline"
        />
      </SidebarPropertyListItem>

      <SidebarPropertyListItem icon={DueDatePropertyIcon} label={t("common.order_by.due_date")}>
        <div className="flex w-full items-center gap-2">
          <DateDropdown
            value={issue.target_date}
            onChange={(val) =>
              onUpdate({
                target_date: val ? renderFormattedPayloadDate(val) : null,
              })
            }
            placeholder={t("issue.add.due_date")}
            buttonVariant="transparent-with-text"
            minDate={minDate ?? undefined}
            disabled={propertyDisabled}
            className="group w-full grow"
            buttonContainerClassName="w-full text-left h-7.5"
            buttonClassName={cn("text-body-xs-medium", {
              "text-placeholder": !issue.target_date,
              "text-danger-primary": shouldHighlightIssueDueDate(issue.target_date, stateGroup),
            })}
            hideIcon
            clearIconClassName="h-3 w-3 hidden group-hover:inline text-primary"
          />
        </div>
      </SidebarPropertyListItem>
    </>
  );
};

const PeekOtherProperties = function PeekOtherProperties(props: PeekPropertyGroupProps) {
  const { workspaceSlug, projectId, issueId, issueOperations, disabled } = props;
  const { t } = useTranslation();
  return (
    <>
      <SidebarPropertyListItem icon={ParentPropertyIcon} label={t("common.parent")}>
        <IssueParentSelectRoot
          className="h-7.5 w-full grow"
          disabled={disabled}
          issueId={issueId}
          issueOperations={issueOperations}
          projectId={projectId}
          workspaceSlug={workspaceSlug}
        />
      </SidebarPropertyListItem>
      <SidebarPropertyListItem icon={LabelPropertyIcon} label={t("common.labels")}>
        <IssueLabel
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          disabled={disabled}
          onLabelUpdate={(labelIds) =>
            issueOperations.update(workspaceSlug, projectId, issueId, { label_ids: labelIds })
          }
        />
      </SidebarPropertyListItem>
    </>
  );
};

export const PeekOverviewProperties = function PeekOverviewProperties(props: IPeekOverviewProperties) {
  const { workspaceSlug, projectId, issueId, issueOperations, disabled } = props;
  // store hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const record = useWorkItemDetailRecord();
  const { getStateById } = useWorkspaceStates(workspaceSlug);
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  // derived values
  const issue = issueId === record?.issueId ? record.issue : undefined;
  const propertyChange = useIssuePropertyChange(
    { workspaceSlug, projectId, issueId, context: "detail" },
    !disabled && !!issue
  );
  const onUpdate = (data: Partial<TIssue>) =>
    propertyChange.run(() => issueOperations.update(workspaceSlug, projectId, issueId, data));
  if (!issue) return <></>;
  const createdByDetails = memberById(issue?.created_by)?.member;
  const projectDetails = projectRows?.find((project) => project.id === issue.project_id);
  const isEstimateEnabled = projectDetails?.estimate;
  const stateDetails = getStateById(issue.state_id);

  const minDate = getDate(issue.start_date);
  minDate?.setDate(minDate.getDate());

  const maxDate = getDate(issue.target_date);
  maxDate?.setDate(maxDate.getDate());

  const groupProps: PeekPropertyGroupProps = {
    onUpdate,
    propertyDisabled: propertyChange.disabled,
    workspaceSlug,
    projectId,
    issueId,
    issueOperations,
    disabled,
    issue,
    createdByDetails,
    minDate,
    maxDate,
    stateGroup: stateDetails?.group,
    isEstimateEnabled,
    moduleView: projectDetails?.module_view,
    cycleView: projectDetails?.cycle_view,
  };

  return (
    <PeekOverviewPropertyGroups groupProps={groupProps} issue={issue} estimateEnabled={Boolean(isEstimateEnabled)} />
  );
};

function PeekOverviewPropertyGroups({
  groupProps,
  issue,
  estimateEnabled,
}: {
  groupProps: PeekPropertyGroupProps;
  issue: TIssue;
  estimateEnabled: boolean;
}) {
  const { workspaceSlug, projectId, issueId, issueOperations, disabled } = groupProps;
  const { t } = useTranslation();

  return (
    <div>
      <h6 className="text-body-xs-medium">{t("common.properties")}</h6>
      <div className={`mt-3 w-full space-y-3 ${disabled ? "opacity-60" : ""}`}>
        <PeekBasicProperties {...groupProps} />
        <PeekDateProperties {...groupProps} />
        <SharedIssueProjectProperties
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          issueOperations={issueOperations}
          issue={issue}
          disabled={disabled}
          estimateEnabled={estimateEnabled}
          estimateButtonClassName={`text-body-xs-medium ${issue.estimate_point !== undefined ? "" : "text-placeholder"}`}
          estimatePlaceholder="None"
          moduleView={groupProps.moduleView}
          cycleView={groupProps.cycleView}
        />
        <PeekOtherProperties {...groupProps} />
      </div>
      <ItemCustomProperties workspaceSlug={workspaceSlug} projectId={projectId} issueId={issueId} disabled={disabled} />
    </div>
  );
}
