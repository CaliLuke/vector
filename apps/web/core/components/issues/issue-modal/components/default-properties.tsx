/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDraftForm } from "../form-state";
import { useState } from "react";
import { useStore } from "@tanstack/react-form";

import { ETabIndices, EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { ParentPropertyIcon } from "@plane/propel/icons";
// types
import type { ISearchIssueResponse } from "@plane/types";
// ui
import { CustomMenu } from "@plane/ui/dropdowns";
import { FormErrors } from "@plane/ui/form-fields";
import { getDate, renderFormattedPayloadDate } from "@plane/utils/datetime";
import { getTabIndex } from "@plane/utils/tab-indices";
// components
import { CycleDropdown } from "@/components/dropdowns/cycle";
import { DateDropdown } from "@/components/dropdowns/date";
import { EstimateDropdown } from "@/components/dropdowns/estimate";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { ModuleDropdown } from "@/components/dropdowns/module/dropdown";
import { PriorityDropdown } from "@/components/dropdowns/priority";
import { StateDropdown } from "@/components/dropdowns/state/dropdown";
import { WorkItemSelectionModal } from "@/components/issues/work-item-selection-modal";
import { IssueLabelSelect } from "@/components/issues/select";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { usePlatformOS } from "@/hooks/use-platform-os";

type TIssueDefaultPropertiesProps = {
  form: IssueDraftForm;
  id: string | undefined;
  projectId: string | null;
  workspaceSlug: string;
  selectedParentIssue: ISearchIssueResponse | null;
  startDate: string | null;
  targetDate: string | null;
  parentId: string | null;
  isDraft: boolean;
  setSelectedParentIssue: (issue: ISearchIssueResponse) => void;
};

export const IssueDefaultProperties = function IssueDefaultProperties(props: TIssueDefaultPropertiesProps) {
  const { form, projectId, workspaceSlug, startDate, targetDate } = props;
  // states
  // store hooks
  const { isMobile } = usePlatformOS();
  const { getIndex } = getTabIndex(ETabIndices.ISSUE_FORM, isMobile);

  return (
    <div className="flex flex-wrap items-center gap-2">
      <form.Field name="priority">
        {({ state: { value, meta }, handleChange: onChange }) => (
          <div data-issue-field="priority" tabIndex={-1}>
            <div className="h-7">
              <PriorityDropdown
                value={value}
                onChange={(priority) => {
                  onChange(priority);
                }}
                buttonVariant="border-with-text"
                tabIndex={getIndex("priority")}
              />
            </div>
            <FormErrors errors={meta.errors} className="mt-1 max-w-48 text-12 text-danger-primary" />
          </div>
        )}
      </form.Field>
      <IssueDateProperties form={form} startDate={startDate} targetDate={targetDate} getIndex={getIndex} />
      <IssueProjectProperties form={form} projectId={projectId} workspaceSlug={workspaceSlug} getIndex={getIndex} />
      {projectId && <IssueParentProperty {...props} projectId={projectId} tabIndex={getIndex("parent_id")} />}
    </div>
  );
};

function IssueProjectProperties({
  form,
  projectId,
  workspaceSlug,
  getIndex,
}: Pick<TIssueDefaultPropertiesProps, "form" | "projectId" | "workspaceSlug"> & {
  getIndex: (key: string) => number | undefined;
}) {
  const { t } = useTranslation();
  const projectCatalogue = useWorkspaceProjectDetails();
  const project = projectCatalogue.projects?.find((projectRow) => projectRow.id === projectId);
  const { can } = useProjectPermissions(workspaceSlug, projectId ?? undefined);
  if (!projectId) return null;

  return (
    <>
      <form.Field name="state_id">
        {({ state: { value, meta }, handleChange }) => (
          <div data-issue-field="state_id" tabIndex={-1}>
            <div className="h-7">
              <StateDropdown
                value={value}
                onChange={handleChange}
                projectId={projectId}
                buttonVariant="border-with-text"
                tabIndex={getIndex("state_id")}
              />
            </div>
            <FormErrors errors={meta.errors} className="mt-1 max-w-48 text-12 text-danger-primary" />
          </div>
        )}
      </form.Field>
      <form.Field name="assignee_ids">
        {({ state: { value, meta }, handleChange }) => (
          <div data-issue-field="assignee_ids" tabIndex={-1}>
            <div className="h-7">
              <MemberDropdown
                workspaceSlug={workspaceSlug}
                projectId={projectId}
                value={value}
                onChange={handleChange}
                buttonVariant={value?.length ? "transparent-without-text" : "border-with-text"}
                buttonClassName={value?.length ? "hover:bg-transparent" : ""}
                placeholder={t("assignees")}
                multiple
                tabIndex={getIndex("assignee_ids")}
              />
            </div>
            <FormErrors errors={meta.errors} className="mt-1 max-w-48 text-12 text-danger-primary" />
          </div>
        )}
      </form.Field>
      <form.Field name="label_ids">
        {({ state: { value, meta }, handleChange }) => (
          <div data-issue-field="label_ids" tabIndex={-1}>
            <div className="h-7">
              <IssueLabelSelect
                value={value}
                onChange={handleChange}
                projectId={projectId}
                tabIndex={getIndex("label_ids")}
                createLabelEnabled={can([EUserPermissions.ADMIN])}
              />
            </div>
            <FormErrors errors={meta.errors} className="mt-1 max-w-48 text-12 text-danger-primary" />
          </div>
        )}
      </form.Field>
      {project?.cycle_view && (
        <form.Field name="cycle_id">
          {({ state: { value, meta }, handleChange }) => (
            <div data-issue-field="cycle_id" tabIndex={-1}>
              <div className="h-7">
                <CycleDropdown
                  projectId={projectId}
                  onChange={handleChange}
                  placeholder={t("cycle.label", { count: 1 })}
                  value={value}
                  buttonVariant="border-with-text"
                  tabIndex={getIndex("cycle_id")}
                />
              </div>
              <FormErrors errors={meta.errors} className="mt-1 max-w-48 text-12 text-danger-primary" />
            </div>
          )}
        </form.Field>
      )}
      {project?.module_view && (
        <form.Field name="module_ids">
          {({ state: { value, meta }, handleChange }) => (
            <div data-issue-field="module_ids" tabIndex={-1}>
              <div className="h-7">
                <ModuleDropdown
                  projectId={projectId}
                  value={value ?? []}
                  onChange={handleChange}
                  placeholder={t("modules")}
                  buttonVariant="border-with-text"
                  tabIndex={getIndex("module_ids")}
                  multiple
                  showCount
                />
              </div>
              <FormErrors errors={meta.errors} className="mt-1 max-w-48 text-12 text-danger-primary" />
            </div>
          )}
        </form.Field>
      )}
      {project?.estimate && (
        <form.Field name="estimate_point">
          {({ state: { value, meta }, handleChange }) => (
            <div data-issue-field="estimate_point" tabIndex={-1}>
              <div className="h-7">
                <EstimateDropdown
                  value={value || undefined}
                  onChange={(estimatePoint) => handleChange(estimatePoint ?? null)}
                  projectId={projectId}
                  buttonVariant="border-with-text"
                  tabIndex={getIndex("estimate_point")}
                  placeholder={t("estimate")}
                />
              </div>
              <FormErrors errors={meta.errors} className="mt-1 max-w-48 text-12 text-danger-primary" />
            </div>
          )}
        </form.Field>
      )}
    </>
  );
}

function IssueDateProperties({
  form,
  startDate,
  targetDate,
  getIndex,
}: Pick<TIssueDefaultPropertiesProps, "form" | "startDate" | "targetDate"> & {
  getIndex: (key: string) => number | undefined;
}) {
  const { t } = useTranslation();
  const minDate = getDate(startDate);
  const maxDate = getDate(targetDate);

  return (
    <>
      <form.Field name="start_date">
        {({ state: { value, meta }, handleChange: onChange }) => (
          <div data-issue-field="start_date" tabIndex={-1}>
            <div className="h-7">
              <DateDropdown
                value={value}
                onChange={(date) => onChange(renderFormattedPayloadDate(date) ?? null)}
                buttonVariant="border-with-text"
                maxDate={maxDate ?? undefined}
                placeholder={t("start_date")}
                tabIndex={getIndex("start_date")}
              />
            </div>
            <FormErrors errors={meta.errors} className="mt-1 max-w-48 text-12 text-danger-primary" />
          </div>
        )}
      </form.Field>
      <form.Field name="target_date">
        {({ state: { value, meta }, handleChange: onChange }) => (
          <div data-issue-field="target_date" tabIndex={-1}>
            <div className="h-7">
              <DateDropdown
                value={value}
                onChange={(date) => onChange(renderFormattedPayloadDate(date) ?? null)}
                buttonVariant="border-with-text"
                minDate={minDate ?? undefined}
                placeholder={t("due_date")}
                tabIndex={getIndex("target_date")}
              />
            </div>
            <FormErrors errors={meta.errors} className="mt-1 max-w-48 text-12 text-danger-primary" />
          </div>
        )}
      </form.Field>
    </>
  );
}

function IssueParentProperty({
  form,
  id,
  projectId,
  workspaceSlug,
  selectedParentIssue,
  parentId,
  isDraft,
  setSelectedParentIssue,
  tabIndex,
}: Pick<
  TIssueDefaultPropertiesProps,
  "form" | "id" | "workspaceSlug" | "selectedParentIssue" | "parentId" | "isDraft" | "setSelectedParentIssue"
> & { projectId: string; tabIndex: number | undefined }) {
  const { t } = useTranslation();
  const [parentIssueListModalOpen, setParentIssueListModalOpen] = useState(false);
  const childTypeId = useStore(form.store, (state) => state.values.type_id);
  return (
    <>
      <div className="h-7" data-issue-field="parent_id" tabIndex={-1}>
        {parentId ? (
          <CustomMenu
            customButtonAsChild
            customButton={
              <button
                type="button"
                className="flex h-full cursor-pointer items-center justify-between gap-1 rounded-sm border-[0.5px] border-strong px-2 py-0.5 text-caption-sm-regular hover:bg-layer-1"
              >
                {selectedParentIssue?.project_id && (
                  <IssueIdentifier
                    projectId={selectedParentIssue.project_id}
                    issueTypeId={selectedParentIssue.type_id}
                    projectIdentifier={selectedParentIssue?.project__identifier}
                    issueSequenceId={selectedParentIssue.sequence_id}
                    size="xs"
                  />
                )}
              </button>
            }
            placement="bottom-start"
            className="h-full w-full"
            customButtonClassName="h-full"
            tabIndex={tabIndex}
          >
            <>
              <CustomMenu.MenuItem className="!p-1" onClick={() => setParentIssueListModalOpen(true)}>
                {t("change_parent_issue")}
              </CustomMenu.MenuItem>
              <form.Field name="parent_id">
                {({ handleChange: onChange }) => (
                  <CustomMenu.MenuItem
                    className="!p-1"
                    onClick={() => {
                      onChange(null);
                    }}
                  >
                    {t("remove_parent_issue")}
                  </CustomMenu.MenuItem>
                )}
              </form.Field>
            </>
          </CustomMenu>
        ) : (
          <button
            type="button"
            className="flex h-full cursor-pointer items-center justify-between gap-1 rounded-sm border-[0.5px] border-strong px-2 py-0.5 text-caption-sm-regular hover:bg-layer-1"
            onClick={() => setParentIssueListModalOpen(true)}
          >
            <ParentPropertyIcon className="h-3 w-3 flex-shrink-0" />
            <span className="whitespace-nowrap">{t("add_parent")}</span>
          </button>
        )}
      </div>
      <form.Field name="parent_id">
        {({ state: { meta }, handleChange: onChange }) => (
          <>
            <FormErrors errors={meta.errors} className="text-12 text-danger-primary" />
            <WorkItemSelectionModal
              workspaceSlug={workspaceSlug}
              filters={{ parent: true, issue_id: isDraft ? undefined : id, child_type_id: childTypeId ?? undefined }}
              isOpen={parentIssueListModalOpen}
              onClose={() => setParentIssueListModalOpen(false)}
              onSelect={(issue) => {
                onChange(issue.id);
                setSelectedParentIssue(issue);
              }}
              projectId={projectId ?? undefined}
              issueId={isDraft ? undefined : id}
            />
          </>
        )}
      </form.Field>
    </>
  );
}
