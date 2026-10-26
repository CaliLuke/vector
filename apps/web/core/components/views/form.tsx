/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo, useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";

import { ETabIndices, ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { EmojiPicker, EmojiIconPickerTypes, Logo } from "@plane/propel/emoji-icon-picker";
import { ViewsIcon } from "@plane/propel/icons";
import type { IProjectView, IIssueFilters } from "@plane/types";
import { EViewAccess, EIssuesStoreType, EIssueLayoutTypes } from "@plane/types";
import { TextArea, ModalForm, EModalPosition, EModalWidth } from "@plane/ui";
import { getComputedDisplayFilters, getComputedDisplayProperties } from "@plane/utils/work-item";
import { getTabIndex } from "@plane/utils/tab-indices";
import { EntityTitleField } from "@/components/common/forms/entity-title-field";
import { DisplayFiltersSelection, FiltersDropdown } from "@/components/issues/issue-layouts/filters";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
import { ProjectLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/project-level";
import { WorkspaceLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/workspace-level";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { LayoutDropDown } from "../dropdowns/layout";
import { createFilterDraft, filterDraftExpression } from "./filter-draft";
import type { FilterDraftNode } from "./filter-draft";

type ViewDraft = Pick<
  IProjectView,
  "name" | "description" | "access" | "display_filters" | "display_properties" | "logo_props" | "rich_filters"
>;
type Props = {
  data?: (Partial<ViewDraft> & { id?: string }) | null;
  preLoadedData?: (Partial<ViewDraft> & { id?: string }) | null;
  projectId?: string;
  workspaceSlug: string;
  handleClose: () => void;
  handleFormSubmit: (values: ViewDraft) => Promise<void>;
};

function initialView({ data, preLoadedData, projectId }: Props): ViewDraft {
  const initial = { ...preLoadedData, ...data };
  return {
    name: initial.name ?? "",
    description: initial.description ?? "",
    access: initial.access ?? EViewAccess.PUBLIC,
    logo_props: initial.logo_props,
    rich_filters: initial.rich_filters ?? {},
    display_properties: initial.display_properties ?? getComputedDisplayProperties(),
    display_filters:
      initial.display_filters ??
      getComputedDisplayFilters(
        projectId ? { group_by: "state" } : { layout: EIssueLayoutTypes.SPREADSHEET, order_by: "-created_at" }
      ),
  };
}

export const ViewForm = function ViewForm(props: Props) {
  const { data, projectId, workspaceSlug, handleClose, handleFormSubmit } = props;
  const { t } = useTranslation();
  const projectCatalogue = useWorkspaceProjectDetails(props.workspaceSlug);
  const projectRows = projectCatalogue.projects;
  const { isMobile } = usePlatformOS();
  const { rich_filters, ...metadata } = initialView(props);
  const form = useForm({
    defaultValues: { ...metadata, filter_nodes: createFilterDraft(rich_filters) },
    onSubmit: async ({ value: { filter_nodes, ...value } }) => {
      form.setErrorMap({ onSubmit: undefined });
      try {
        await handleFormSubmit({ ...value, rich_filters: filterDraftExpression(filter_nodes) });
      } catch (cause) {
        form.setErrorMap({ onSubmit: { form: "Could not save the view. Please try again.", fields: {} } });
        throw cause;
      }
    },
  });
  const saveError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const values = useStore(form.store, (state) => state.values);
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const project = projectId ? projectRows?.find((candidate) => candidate.id === projectId) : undefined;
  const { getIndex } = getTabIndex(ETabIndices.PROJECT_VIEW, isMobile);
  const submitViewForm = () => {
    form.setErrorMap({ onSubmit: undefined });
    return form.handleSubmit().catch(() => {});
  };
  return (
    <ModalForm
      busy={busy}
      onSubmit={submitViewForm}
      handleClose={handleClose}
      position={EModalPosition.TOP}
      width={EModalWidth.XXL}
    >
      <fieldset disabled={busy} inert={busy} className="space-y-5 p-5">
        <h3 className="text-18 font-medium text-secondary">{data ? t("view.update.label") : t("view.create.label")}</h3>
        <div className="space-y-3">
          <div className="flex w-full items-start gap-2">
            {projectId && (
              <ViewIconPicker
                logo={values.logo_props}
                disabled={busy}
                onChange={(value) => form.setFieldValue("logo_props", value)}
              />
            )}
            <form.Field
              name="name"
              validators={{
                onChange: ({ value }) =>
                  !value.trim()
                    ? t("form.title.required")
                    : value.length > 255
                      ? t("form.title.max_length", { length: 255 })
                      : undefined,
              }}
            >
              {(field) => (
                <EntityTitleField
                  id="name"
                  name={field.name}
                  value={field.state.value}
                  onChange={(value) => field.handleChange(value)}
                  onBlur={field.handleBlur}
                  disabled={busy}
                  hasError={!field.state.meta.isValid}
                  placeholder={t("common.title")}
                  tabIndex={getIndex("name")}
                  errors={field.state.meta.errors}
                  className="w-full space-y-1"
                />
              )}
            </form.Field>
          </div>
          <TextArea
            id="description"
            name="description"
            value={values.description}
            onChange={(event) => form.setFieldValue("description", event.target.value)}
            placeholder={t("common.description")}
            className="min-h-24 w-full resize-none text-14"
            tabIndex={getIndex("descriptions")}
          />
          <div className="flex gap-2">
            {projectId && (
              <LayoutDropDown
                value={values.display_filters.layout}
                onChange={(layout) => form.setFieldValue("display_filters", (current) => ({ ...current, layout }))}
              />
            )}
            <FiltersDropdown title={t("common.display")}>
              <DisplayFiltersSelection
                layoutDisplayFiltersOptions={
                  projectId
                    ? ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.layoutOptions[values.display_filters.layout]
                    : ISSUE_DISPLAY_FILTERS_BY_PAGE.my_issues.layoutOptions.spreadsheet
                }
                displayFilters={values.display_filters}
                handleDisplayFiltersUpdate={(update) =>
                  form.setFieldValue("display_filters", (current) => ({ ...current, ...update }))
                }
                displayProperties={values.display_properties}
                handleDisplayPropertiesUpdate={(update) =>
                  form.setFieldValue("display_properties", (current) => ({ ...current, ...update }))
                }
                cycleViewDisabled={projectId ? !project?.cycle_view : undefined}
                moduleViewDisabled={projectId ? !project?.module_view : undefined}
              />
            </FiltersDropdown>
          </div>
          <form.Field name="filter_nodes">
            {(field) => (
              <ViewFilters
                entityId={data?.id}
                projectId={projectId}
                workspaceSlug={workspaceSlug}
                nodes={field.state.value}
                onChange={field.handleChange}
                displayFilters={values.display_filters}
                displayProperties={values.display_properties}
              />
            )}
          </form.Field>
        </div>
        {saveError && (
          <p role="alert" className="text-13 text-danger-primary">
            {saveError}
          </p>
        )}
      </fieldset>
      <div className="flex items-center justify-end gap-2 border-t-[0.5px] border-subtle px-5 py-4">
        <Button
          type="button"
          variant="secondary"
          size="lg"
          disabled={busy}
          onClick={handleClose}
          tabIndex={getIndex("cancel")}
        >
          {t("common.cancel")}
        </Button>
        <Button variant="primary" size="lg" type="submit" loading={busy} tabIndex={getIndex("submit")}>
          {data ? t("view.update.label") : t("view.create.label")}
        </Button>
      </div>
    </ModalForm>
  );
};

function ViewFilters({
  entityId,
  projectId,
  workspaceSlug,
  nodes,
  onChange,
  displayFilters,
  displayProperties,
}: {
  entityId: string | undefined;
  projectId: string | undefined;
  workspaceSlug: string;
  nodes: FilterDraftNode[];
  onChange: (nodes: FilterDraftNode[]) => void;
  displayFilters: IIssueFilters["displayFilters"];
  displayProperties: IIssueFilters["displayProperties"];
}) {
  const richFilters = useMemo(() => filterDraftExpression(nodes), [nodes]);
  const filterProps = {
    entityId,
    initialWorkItemFilters: { richFilters, displayFilters, displayProperties, kanbanFilters: undefined },
    isTemporary: true as const,
    showOnMount: true,
    workspaceSlug,
    updateFilters: (expression: IProjectView["rich_filters"]) => onChange(createFilterDraft(expression)),
  };
  return projectId ? (
    <ProjectLevelWorkItemFiltersHOC
      {...filterProps}
      projectId={projectId}
      entityType={EIssuesStoreType.PROJECT_VIEW}
      filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.filters}
    >
      {({ editor }) => editor && <WorkItemFiltersRow editor={editor} variant="modal" />}
    </ProjectLevelWorkItemFiltersHOC>
  ) : (
    <WorkspaceLevelWorkItemFiltersHOC
      {...filterProps}
      entityType={EIssuesStoreType.GLOBAL}
      filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE.my_issues.filters}
    >
      {({ editor }) => editor && <WorkItemFiltersRow editor={editor} variant="modal" />}
    </WorkspaceLevelWorkItemFiltersHOC>
  );
}

function ViewIconPicker({
  logo,
  disabled,
  onChange,
}: {
  logo: IProjectView["logo_props"];
  disabled: boolean;
  onChange: (value: IProjectView["logo_props"]) => void;
}) {
  const [isOpen, setIsOpen] = useState(false);
  return (
    <EmojiPicker
      iconType="lucide"
      isOpen={isOpen}
      handleToggle={setIsOpen}
      disabled={disabled}
      label={
        <span className="grid h-9 w-9 place-items-center rounded-md bg-surface-2">
          {logo?.in_use ? (
            <Logo logo={logo} size={18} type="lucide" />
          ) : (
            <ViewsIcon className="h-4 w-4 text-tertiary" />
          )}
        </span>
      }
      onChange={(value) => {
        onChange(
          value.type === "emoji"
            ? { in_use: "emoji", emoji: { value: value.value } }
            : { in_use: "icon", icon: value.value }
        );
        setIsOpen(false);
      }}
      defaultIconColor={logo?.icon?.color}
      defaultOpen={logo?.in_use === "emoji" ? EmojiIconPickerTypes.EMOJI : EmojiIconPickerTypes.ICON}
    />
  );
}
