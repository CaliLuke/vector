/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useWorkItemTypeCatalog } from "@/hooks/use-work-item-type-catalog";
import { useCurrentWorkspace } from "@/hooks/use-workspaces";
import intersection from "lodash-es/intersection.js";

import { useForm, useStore } from "@tanstack/react-form";
import { EUserPermissions, EXPORTERS_LIST } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { CustomSearchSelect, CustomSelect, ToggleSwitch } from "@plane/ui";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useCreationPermissions } from "@/hooks/use-creation-permissions";
import { ProjectExportService } from "@/services/project/project-export.service";
// local imports
import { SettingsBoxedControlItem } from "../settings/boxed-control-item";

type Props = {
  workspaceSlug: string;
  provider: string | null;
  mutateServices: () => void;
};
type FormData = {
  provider: (typeof EXPORTERS_LIST)[0];
  project: string[];
  includeCustomProperties: boolean;
};

const projectExportService = new ProjectExportService();

export const ExportForm = function ExportForm(props: Props) {
  // props
  const { workspaceSlug, mutateServices } = props;

  // store hooks
  const { can: canWorkspace } = useWorkspacePermissions(workspaceSlug);
  const isMember = canWorkspace([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);
  const { identities } = useWorkItemTypeCatalog(workspaceSlug, undefined, isMember);
  const customPropertiesEnabled = identities?.enabled === true;
  const { data: user } = useCurrentUser();
  const { canPerformAnyCreateAction, projectsWithCreatePermissions } = useCreationPermissions();
  const { data: catalogueWorkspace } = useCurrentWorkspace();
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const workspaceProjectIds = projectRows
    ?.filter((project) => project.workspace === catalogueWorkspace?.id && !project.archived_at)
    .map((project) => project.id);
  const { t } = useTranslation();
  // form
  const defaultValues: FormData = {
    provider: EXPORTERS_LIST[0],
    project: [],
    includeCustomProperties: false,
  };
  const form = useForm({
    defaultValues,
    onSubmit: async ({ value }) => {
      await exportToMail(value);
    },
  });
  const exportLoading = useStore(form.store, (state) => state.isSubmitting);

  // derived values
  const hasProjects = workspaceProjectIds && workspaceProjectIds.length > 0;
  const wsProjectIdsWithCreatePermisisons = projectsWithCreatePermissions
    ? intersection(workspaceProjectIds, Object.keys(projectsWithCreatePermissions))
    : [];
  const options = wsProjectIdsWithCreatePermisisons?.map((projectId) => {
    const projectDetails = projectRows?.find((project) => project.id === projectId);

    return {
      value: projectDetails?.id,
      query: `${projectDetails?.name} ${projectDetails?.identifier}`,
      content: (
        <div className="flex items-center gap-2">
          <span className="flex-shrink-0 text-10 text-secondary">{projectDetails?.identifier}</span>
          <span className="truncate">{projectDetails?.name}</span>
        </div>
      ),
    };
  });

  // handlers
  async function exportToMail(formData: FormData) {
    if (workspaceSlug && user) {
      const payload = {
        provider: formData.provider.provider,
        project: formData.project,
        multiple: formData.project.length > 1,
        rich_filters: {},
        include_custom_properties: customPropertiesEnabled && formData.includeCustomProperties,
      };
      try {
        await projectExportService.csvExport(workspaceSlug, payload);
        mutateServices();

        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("workspace_settings.settings.exports.modal.toasts.success.title"),
          message: t("workspace_settings.settings.exports.modal.toasts.success.message", {
            entity:
              formData.provider.provider === "csv"
                ? "CSV"
                : formData.provider.provider === "xlsx"
                  ? "Excel"
                  : formData.provider.provider === "json"
                    ? "JSON"
                    : "",
          }),
        });
      } catch (_error) {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("error"),
          message: t("workspace_settings.settings.exports.modal.toasts.error.message"),
        });
      }
    }
  }

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        void form.handleSubmit();
      }}
      className="flex flex-col gap-5"
    >
      <div className="rounded-lg border border-subtle bg-layer-2">
        {/* Project Selector */}
        <SettingsBoxedControlItem
          className="rounded-none border-0 border-b"
          title={t("workspace_settings.settings.exports.exporting_projects")}
          control={
            <form.Field name="project">
              {(field) => (
                <CustomSearchSelect
                  disabled={exportLoading || (!isMember && (!hasProjects || !canPerformAnyCreateAction))}
                  value={field.state.value}
                  onChange={field.handleChange}
                  options={options}
                  input
                  label={
                    field.state.value.length > 0
                      ? field.state.value
                          .map((projectId) => {
                            const projectDetails = projectRows?.find((project) => project.id === projectId);

                            return projectDetails?.identifier;
                          })
                          .join(", ")
                      : "All projects"
                  }
                  optionsClassName="max-w-48 sm:max-w-[532px]"
                  placement="bottom-end"
                  multiple
                />
              )}
            </form.Field>
          }
        />
        {/* Format Selector */}
        <SettingsBoxedControlItem
          className="rounded-none border-0 border-b"
          title={t("workspace_settings.settings.exports.format")}
          control={
            <form.Field name="provider">
              {(field) => (
                <CustomSelect
                  disabled={exportLoading || (!isMember && (!hasProjects || !canPerformAnyCreateAction))}
                  value={field.state.value}
                  onChange={(value: FormData["provider"]) => field.handleChange(value)}
                  label={t(field.state.value.i18n_title)}
                  optionsClassName="max-w-48 sm:max-w-[532px]"
                  placement="bottom-end"
                  buttonClassName="py-2 text-13"
                >
                  {EXPORTERS_LIST.map((service) => (
                    <CustomSelect.Option key={service.provider} className="flex items-center gap-2" value={service}>
                      <span className="truncate">{t(service.i18n_title)}</span>
                    </CustomSelect.Option>
                  ))}
                </CustomSelect>
              )}
            </form.Field>
          }
        />
        {customPropertiesEnabled && (
          <SettingsBoxedControlItem
            className="rounded-none border-0 border-b"
            title="Include custom properties"
            description="Include work item types and custom property values, including formula results."
            control={
              <form.Field name="includeCustomProperties">
                {(field) => (
                  <ToggleSwitch
                    value={field.state.value}
                    onChange={field.handleChange}
                    disabled={exportLoading || !isMember}
                    aria-label="Include custom properties"
                  />
                )}
              </form.Field>
            }
          />
        )}
        <div className="px-4 py-3">
          <Button variant="primary" size="lg" type="submit" loading={exportLoading}>
            {exportLoading ? `${t("workspace_settings.settings.exports.exporting")}...` : t("export")}
          </Button>
        </div>
      </div>
    </form>
  );
};
