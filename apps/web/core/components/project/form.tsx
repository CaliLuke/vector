/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useForm, useStore } from "@tanstack/react-form";
import { useMutation } from "@tanstack/react-query";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EFileAssetType } from "@plane/types";
import type { IProject } from "@plane/types";
import { Input, TextArea } from "@plane/ui";
import { renderFormattedDate } from "@plane/utils/datetime";
import { projectIdentifierSanitizer } from "@plane/utils/project";
import { CoverImage } from "@/components/common/cover-image";
import { ImagePickerPopover } from "@/components/core/image-picker-popover";
import { TimezoneSelect } from "@/components/global/timezone-select";
import { handleCoverImageChange } from "@/helpers/cover-image.helper";
import { useProjectMutations } from "@/hooks/use-projects";
import { ProjectService } from "@/services/project";
import { ProjectLogoPicker } from "./project-logo-picker";
import { ProjectNetworkSelect } from "./project-network-select";

type Props = { project: IProject; workspaceSlug: string; projectId: string; isAdmin: boolean };
const service = new ProjectService();
function defaults(project: IProject) {
  return {
    name: project.name,
    identifier: project.identifier,
    description: project.description ?? "",
    logo_props: project.logo_props,
    network: project.network ?? 2,
    timezone: project.timezone ?? "UTC",
    cover_image_url: project.cover_image_url ?? "",
  };
}
export function ProjectDetailsForm(props: Props) {
  return <ProjectSettingsForm key={`${props.workspaceSlug}/${props.projectId}`} {...props} />;
}
function ProjectSettingsForm({ project, workspaceSlug, isAdmin }: Props) {
  const { t } = useTranslation();
  const projectMutation = useProjectMutations(workspaceSlug, project.id);
  const cover = useMutation({
    mutationFn: async (url: string) => ({
      url,
      payload: await handleCoverImageChange(project.cover_image_url, url, {
        workspaceSlug,
        entityIdentifier: project.id,
        entityType: EFileAssetType.PROJECT_COVER,
        isUserAsset: false,
      }),
    }),
  });
  const form = useForm({
    defaultValues: defaults(project),
    validators: {
      onSubmitAsync: async ({ value }) => {
        if (!isAdmin || value.identifier === project.identifier) return undefined;
        try {
          const availability = await service.checkProjectIdentifierAvailability(workspaceSlug, value.identifier);
          return availability.exists ? { fields: { identifier: t("common.identifier_already_exists") } } : undefined;
        } catch {
          return { fields: { identifier: "Could not verify this project ID. Please try again." } };
        }
      },
    },
    onSubmit: async ({ value, formApi }) => {
      if (!isAdmin) return;
      const image =
        cover.data?.url === value.cover_image_url ? cover.data : await cover.mutateAsync(value.cover_image_url);
      const { cover_image_url: _coverUrl, ...fields } = value;
      const result = await projectMutation.mutateAsync({
        type: "update",
        projectId: project.id,
        data: { ...fields, ...image.payload },
      });
      const saved = result.project;
      if (!saved) throw new Error("The server did not return the updated project.");
      formApi.reset(defaults(saved));
      cover.reset();
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("toast.success"),
        message: t("project_settings.general.toast.success"),
      });
    },
  });
  const values = useStore(form.store, (state) => state.values);
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const disabled = !isAdmin || busy;
  return (
    <form
      noValidate
      onSubmit={(event) => {
        event.preventDefault();
        form.setErrorMap({ onSubmit: undefined });
        void form.handleSubmit().catch(() => {
          form.setErrorMap({ onSubmit: { form: t("something_went_wrong"), fields: {} } });
        });
      }}
    >
      <fieldset disabled={disabled} inert={busy}>
        <div className="relative h-44 w-full">
          <CoverImage src={values.cover_image_url} alt="Project cover image" className="h-44 w-full rounded-md" />
          <div className="absolute bottom-4 flex w-full items-end justify-between gap-3 px-4">
            <div className="flex items-center gap-3">
              <ProjectLogoPicker
                value={values.logo_props}
                onChange={(value) => form.setFieldValue("logo_props", value)}
                disabled={disabled}
              />
              <div className="flex flex-col text-on-color">
                <span className="text-16 font-semibold">{values.name}</span>
                <span className="text-13">{values.identifier}</span>
              </div>
            </div>
            <ImagePickerPopover
              workspaceSlug={workspaceSlug}
              label={t("change_cover")}
              value={values.cover_image_url}
              onChange={(value) => form.setFieldValue("cover_image_url", value)}
              disabled={disabled}
              projectId={project.id}
            />
          </div>
        </div>
        <div className="mt-8 flex flex-col gap-8">
          <form.Field
            name="name"
            validators={{
              onChange: ({ value }) =>
                !value.trim()
                  ? t("name_is_required")
                  : value.length > 255
                    ? t("title_should_be_less_than_255_characters")
                    : undefined,
            }}
          >
            {(field) => (
              <label className="flex flex-col gap-1 text-13">
                {t("common.project_name")}
                <Input
                  name={field.name}
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                  onBlur={field.handleBlur}
                  hasError={!field.state.meta.isValid}
                  className="rounded-md !p-3 font-medium"
                />
                {field.state.meta.errors.map((message) => (
                  <span key={message} role="alert" className="text-11 text-danger-primary">
                    {message}
                  </span>
                ))}
              </label>
            )}
          </form.Field>
          <label className="flex flex-col gap-1 text-13">
            {t("description")}
            <TextArea
              name="description"
              value={values.description}
              onChange={(event) => form.setFieldValue("description", event.target.value)}
              className="min-h-[102px] text-13 font-medium"
            />
          </label>
          <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
            <form.Field
              name="identifier"
              validators={{
                onChange: ({ value }) =>
                  !value ? t("project_id_is_required") : value.length > 10 ? t("project_id_max_char") : undefined,
              }}
            >
              {(field) => (
                <label className="flex flex-col gap-1 text-13">
                  Project ID
                  <Input
                    name={field.name}
                    value={field.state.value}
                    onChange={(event) =>
                      field.handleChange(projectIdentifierSanitizer(event.target.value).toUpperCase())
                    }
                    onBlur={field.handleBlur}
                    hasError={!field.state.meta.isValid}
                  />
                  {field.state.meta.errors.map((message) => (
                    <span key={message} role="alert" className="text-11 text-danger-primary">
                      {message}
                    </span>
                  ))}
                </label>
              )}
            </form.Field>
            <div className="flex flex-col gap-1">
              <h4 className="text-13">{t("workspace_projects.network.label")}</h4>
              <ProjectNetworkSelect
                value={values.network}
                onChange={(value) => form.setFieldValue("network", value)}
                disabled={disabled}
                input
              />
            </div>
            <form.Field
              name="timezone"
              validators={{
                onChange: ({ value }) => (value ? undefined : t("project_settings.general.please_select_a_timezone")),
              }}
            >
              {(field) => (
                <div className="flex flex-col gap-1">
                  <h4 className="text-13">{t("common.project_timezone")}</h4>
                  <TimezoneSelect
                    value={field.state.value}
                    onChange={field.handleChange}
                    error={!field.state.meta.isValid}
                    disabled={disabled}
                  />
                  {field.state.meta.errors.map((message) => (
                    <span key={message} role="alert" className="text-11 text-danger-primary">
                      {message}
                    </span>
                  ))}
                </div>
              )}
            </form.Field>
          </div>
          {typeof formError === "string" && (
            <p role="alert" className="text-13 text-danger-primary">
              {formError}
            </p>
          )}
          <div className="flex items-center justify-between py-2">
            <Button type="submit" variant="primary" loading={busy} disabled={disabled}>
              {t("common.update_project")}
            </Button>
            <span className="text-13 text-placeholder italic">
              {t("common.created_on")} {renderFormattedDate(project.created_at)}
            </span>
          </div>
        </div>
      </fieldset>
    </form>
  );
}
