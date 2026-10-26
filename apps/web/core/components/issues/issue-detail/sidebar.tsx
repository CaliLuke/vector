/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// i18n
import { useTranslation } from "@plane/i18n";
// ui
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
import type { TIssue, TStateGroups, IUserLite } from "@plane/types";
import { Avatar } from "@plane/ui";
// components
import { ItemCustomProperties } from "@/components/work-item-types/item-custom-properties";
import { DateDropdown } from "@/components/dropdowns/date";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { PriorityDropdown } from "@/components/dropdowns/priority";
import { StateDropdown } from "@/components/dropdowns/state/dropdown";
// hooks
import { useProjectEstimates } from "@/hooks/use-estimates";
import { useWorkItemDetailRecord } from "@/providers/work-item-detail-record";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useProjectStates } from "@/hooks/use-project-states";
// components
import { IssueParentSelectRoot } from "@/components/issues/parent-select-root";
import { SidebarPropertyListItem } from "@/components/common/layout/sidebar/property-list-item";
import { IssueLabel } from "./label";
import { IssueProjectProperties as SharedIssueProjectProperties } from "./project-properties";
import type { TIssueOperations } from "./root";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueOperations: TIssueOperations;
  isEditable: boolean;
};

type SidebarPropertyGroupsProps = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueOperations: TIssueOperations;
  isEditable: boolean;
  issue: TIssue;
  createdByDetails?: IUserLite;
  minDate: ReturnType<typeof getDate> | null;
  maxDate: ReturnType<typeof getDate> | null;
  stateGroup: TStateGroups | undefined;
  estimateEnabled: boolean;
  moduleView: boolean | undefined;
  cycleView: boolean | undefined;
};

const IssueIdentityProperties = function IssueIdentityProperties(props: SidebarPropertyGroupsProps) {
  const { workspaceSlug, projectId, issueId, issueOperations, isEditable, issue, createdByDetails } = props;
  const { t } = useTranslation();
  return (
    <>
      <SidebarPropertyListItem icon={StatePropertyIcon} label={t("common.state")}>
        <StateDropdown
          value={issue.state_id}
          onChange={(value) => void issueOperations.update(workspaceSlug, projectId, issueId, { state_id: value })}
          projectId={projectId}
          disabled={!isEditable}
          buttonVariant="transparent-with-text"
          className="group w-full grow"
          buttonContainerClassName="w-full text-left h-7.5"
          buttonClassName="text-body-xs-regular"
          dropdownArrow
          dropdownArrowClassName="h-3.5 w-3.5 hidden group-hover:inline"
        />
      </SidebarPropertyListItem>
      <SidebarPropertyListItem icon={MembersPropertyIcon} label={t("common.assignees")}>
        <MemberDropdown
          workspaceSlug={workspaceSlug}
          value={issue.assignee_ids ?? undefined}
          onChange={(value) => void issueOperations.update(workspaceSlug, projectId, issueId, { assignee_ids: value })}
          disabled={!isEditable}
          projectId={projectId}
          placeholder={t("issue.add.assignee")}
          multiple
          buttonVariant={issue.assignee_ids?.length > 1 ? "transparent-without-text" : "transparent-with-text"}
          className="group w-full grow"
          buttonContainerClassName="w-full text-left h-7.5"
          buttonClassName={`text-body-xs-regular justify-between ${issue.assignee_ids?.length > 0 ? "" : "text-placeholder"}`}
          hideIcon={issue.assignee_ids?.length === 0}
          dropdownArrow
          dropdownArrowClassName="h-3.5 w-3.5 hidden group-hover:inline"
        />
      </SidebarPropertyListItem>
      <SidebarPropertyListItem icon={PriorityPropertyIcon} label={t("common.priority")}>
        <PriorityDropdown
          value={issue.priority}
          onChange={(value) => void issueOperations.update(workspaceSlug, projectId, issueId, { priority: value })}
          disabled={!isEditable}
          buttonVariant="transparent-with-text"
          className="h-7.5 w-full grow rounded-sm"
          buttonContainerClassName="size-full text-left"
          buttonClassName="size-full px-2 py-0.5 whitespace-nowrap [&_svg]:size-3.5"
        />
      </SidebarPropertyListItem>
      {createdByDetails && (
        <SidebarPropertyListItem icon={UserCirclePropertyIcon} label={t("common.created_by")}>
          <div className="flex gap-2 px-2">
            <Avatar
              src={getFileURL(createdByDetails.avatar_url ?? "")}
              name={createdByDetails.display_name}
              size="md"
              showTooltip={false}
            />
            <span className="grow truncate text-body-xs-regular leading-5">{createdByDetails.display_name}</span>
          </div>
        </SidebarPropertyListItem>
      )}
    </>
  );
};

const IssueDateProperties = function IssueDateProperties(props: SidebarPropertyGroupsProps) {
  const { workspaceSlug, projectId, issueId, issueOperations, isEditable, issue, minDate, maxDate, stateGroup } = props;
  const { t } = useTranslation();
  return (
    <>
      <SidebarPropertyListItem icon={StartDatePropertyIcon} label={t("common.order_by.start_date")}>
        <DateDropdown
          placeholder={t("issue.add.start_date")}
          value={issue.start_date}
          onChange={(value) =>
            void issueOperations.update(workspaceSlug, projectId, issueId, {
              start_date: value ? renderFormattedPayloadDate(value) : null,
            })
          }
          maxDate={maxDate ?? undefined}
          disabled={!isEditable}
          buttonVariant="transparent-with-text"
          className="group w-full grow"
          buttonContainerClassName="w-full text-left h-7.5"
          buttonClassName={`text-body-xs-regular ${issue.start_date ? "" : "text-placeholder"}`}
          hideIcon
          clearIconClassName="h-3 w-3 hidden group-hover:inline"
        />
      </SidebarPropertyListItem>
      <SidebarPropertyListItem icon={DueDatePropertyIcon} label={t("common.order_by.due_date")}>
        <DateDropdown
          placeholder={t("issue.add.due_date")}
          value={issue.target_date}
          onChange={(value) =>
            void issueOperations.update(workspaceSlug, projectId, issueId, {
              target_date: value ? renderFormattedPayloadDate(value) : null,
            })
          }
          minDate={minDate ?? undefined}
          disabled={!isEditable}
          buttonVariant="transparent-with-text"
          className="group w-full grow"
          buttonContainerClassName="w-full text-left h-7.5"
          buttonClassName={cn("text-body-xs-regular", {
            "text-placeholder": !issue.target_date,
            "text-danger-primary": shouldHighlightIssueDueDate(issue.target_date, stateGroup),
          })}
          hideIcon
          clearIconClassName="h-3 w-3 hidden group-hover:inline text-primary"
        />
      </SidebarPropertyListItem>
    </>
  );
};

const IssueAdditionalProperties = function IssueAdditionalProperties(props: SidebarPropertyGroupsProps) {
  const { workspaceSlug, projectId, issueId, issueOperations, isEditable } = props;
  const { t } = useTranslation();
  return (
    <>
      <SharedIssueProjectProperties
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
        issueOperations={issueOperations}
        issue={props.issue}
        disabled={!isEditable}
        estimateEnabled={props.estimateEnabled}
        estimateButtonClassName={`text-body-xs-regular ${props.issue.estimate_point !== null ? "" : "text-placeholder"}`}
        estimatePlaceholder={t("common.none")}
        moduleView={props.moduleView}
        cycleView={props.cycleView}
      />
      <SidebarPropertyListItem icon={ParentPropertyIcon} label={t("common.parent")}>
        <IssueParentSelectRoot
          className="h-7.5 w-full grow"
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          issueOperations={issueOperations}
          disabled={!isEditable}
        />
      </SidebarPropertyListItem>
      <SidebarPropertyListItem icon={LabelPropertyIcon} label={t("common.labels")}>
        <IssueLabel
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          disabled={!isEditable}
          onLabelUpdate={(labelIds) =>
            issueOperations.update(workspaceSlug, projectId, issueId, { label_ids: labelIds })
          }
        />
      </SidebarPropertyListItem>
    </>
  );
};

export const IssueDetailsSidebar = function IssueDetailsSidebar(props: Props) {
  const { t } = useTranslation();
  const { workspaceSlug, projectId, issueId, issueOperations, isEditable } = props;
  // store hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const { isEnabled: estimateEnabled } = useProjectEstimates(workspaceSlug, projectId);
  const record = useWorkItemDetailRecord();
  const { memberById } = useWorkspaceMemberRoster(workspaceSlug);
  const { getStateById } = useProjectStates(workspaceSlug, projectId);
  const issue = issueId === record?.issueId ? record.issue : undefined;
  if (!issue) return <></>;

  const createdByDetails = memberById(issue.created_by)?.member;

  // derived values
  const projectDetails = projectRows?.find((project) => project.id === issue.project_id);
  const stateDetails = getStateById(issue.state_id);

  const minDate = issue.start_date ? getDate(issue.start_date) : null;
  minDate?.setDate(minDate.getDate());

  const maxDate = issue.target_date ? getDate(issue.target_date) : null;
  maxDate?.setDate(maxDate.getDate());

  const groupProps: SidebarPropertyGroupsProps = {
    workspaceSlug,
    projectId,
    issueId,
    issueOperations,
    isEditable,
    issue,
    createdByDetails,
    minDate,
    maxDate,
    stateGroup: stateDetails?.group,
    estimateEnabled,
    moduleView: projectDetails?.module_view,
    cycleView: projectDetails?.cycle_view,
  };

  return (
    <div className="flex h-full w-full flex-col items-center divide-y-2 divide-subtle-1 overflow-hidden">
      <div className="h-full w-full overflow-y-auto px-6">
        <h5 className="mt-5 text-body-xs-medium">{t("common.properties")}</h5>
        <div className={`mt-4 mb-2 space-y-2.5 truncate ${!isEditable ? "opacity-60" : ""}`}>
          <IssueIdentityProperties {...groupProps} />
          <IssueDateProperties {...groupProps} />
          <IssueAdditionalProperties {...groupProps} />
        </div>
        <ItemCustomProperties
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          disabled={!isEditable}
        />
      </div>
    </div>
  );
};
