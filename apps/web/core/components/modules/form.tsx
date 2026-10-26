/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { validateTitle } from "@/helpers/form-validation";
import { useForm, useStore } from "@tanstack/react-form";
import { useEffect, useRef } from "react";
import { ETabIndices } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import type { IModule } from "@plane/types";
import { TextArea, ModalCore, EModalPosition, EModalWidth } from "@plane/ui";
import { getDate, renderFormattedPayloadDate } from "@plane/utils/datetime";
import { getTabIndex } from "@plane/utils/tab-indices";
import { EntityTitleField } from "@/components/common/forms/entity-title-field";
import { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { ProjectDropdown } from "@/components/dropdowns/project/dropdown";
import { ModuleStatusSelect } from "./select/status";
import { useCreationPermissions } from "@/hooks/use-creation-permissions";

type Props = {
  handleFormSubmit: (value: Partial<IModule>) => Promise<void>;
  handleClose: () => void;
  projectId: string;
  workspaceSlug: string;
  data?: IModule;
  isMobile?: boolean;
};

function moduleDefaults(projectId: string, data?: IModule) {
  const {
    project_id = projectId,
    name = "",
    description = "",
    status = "backlog",
    lead_id = null,
    member_ids = [],
    start_date = null,
    target_date = null,
  } = data ?? {};
  return { project_id, name, description, status, lead_id, member_ids, start_date, target_date };
}

export function ModuleForm({ handleFormSubmit, handleClose, projectId, workspaceSlug, data, isMobile = false }: Props) {
  const nameInputRef = useRef<HTMLInputElement>(null);
  useEffect(() => {
    nameInputRef.current?.focus();
  }, []);
  const { projectsWithCreatePermissions } = useCreationPermissions();
  const { t } = useTranslation();
  const { getIndex } = getTabIndex(ETabIndices.PROJECT_MODULE, isMobile);
  const form = useForm({
    defaultValues: moduleDefaults(projectId, data),
    validators: {
      onChange: ({ value }) =>
        value.start_date && value.target_date && value.start_date > value.target_date
          ? "End date must be on or after the start date."
          : undefined,
    },
    onSubmit: async ({ value }) => {
      await handleFormSubmit(value);
    },
  });
  const { isSubmitting, values, errors } = useStore(form.store, (state) => ({
    isSubmitting: state.isSubmitting,
    values: state.values,
    errors: state.errors,
  }));
  const close = () => {
    if (!isSubmitting) handleClose();
  };
  return (
    <ModalCore isOpen handleClose={close} position={EModalPosition.TOP} width={EModalWidth.XXL}>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => {});
        }}
      >
        <div className="space-y-5 p-5">
          <div className="flex items-center gap-x-3">
            {!data && (
              <form.Field
                name="project_id"
                validators={{ onChange: ({ value }) => (value ? undefined : "Select a project.") }}
              >
                {(field) => (
                  <div className="h-7">
                    <ProjectDropdown
                      value={field.state.value}
                      onChange={(value) => {
                        if (Array.isArray(value) || !value) return;
                        field.handleChange(value);
                        form.setFieldValue("lead_id", null);
                        form.setFieldValue("member_ids", []);
                      }}
                      disabled={isSubmitting}
                      multiple={false}
                      buttonVariant="border-with-text"
                      renderCondition={(id) => Boolean(projectsWithCreatePermissions?.[id])}
                      tabIndex={getIndex("cover_image")}
                    />
                  </div>
                )}
              </form.Field>
            )}
            <h3 className="text-18 font-medium text-secondary">
              {t(data ? "common.update" : "common.create")} {t("common.module").toLowerCase()}
            </h3>
          </div>
          <div className="space-y-3">
            <form.Field
              name="name"
              validators={{
                onChange: ({ value }) => {
                  const error = validateTitle(value);
                  return error ? t(error) : undefined;
                },
              }}
            >
              {(field) => (
                <EntityTitleField
                  id={field.name}
                  name={field.name}
                  value={field.state.value}
                  onChange={(value) => field.handleChange(value)}
                  onBlur={field.handleBlur}
                  disabled={isSubmitting}
                  hasError={field.state.meta.errors.length > 0}
                  placeholder={t("title")}
                  tabIndex={getIndex("name")}
                  errors={field.state.meta.errors}
                  inputRef={nameInputRef}
                />
              )}
            </form.Field>
            <form.Field name="description">
              {(field) => (
                <TextArea
                  id={field.name}
                  name={field.name}
                  value={field.state.value ?? ""}
                  onChange={(event) => field.handleChange(event.target.value)}
                  onBlur={field.handleBlur}
                  disabled={isSubmitting}
                  placeholder={t("description")}
                  className="min-h-24 w-full resize-none text-14"
                  tabIndex={getIndex("description")}
                />
              )}
            </form.Field>
            <div className="flex flex-wrap items-center gap-2">
              <DateRangeDropdown
                buttonVariant="border-with-text"
                className="h-7"
                value={{ from: getDate(values.start_date), to: getDate(values.target_date) }}
                onSelect={(value) => {
                  form.setFieldValue(
                    "start_date",
                    value?.from ? (renderFormattedPayloadDate(value.from) ?? null) : null
                  );
                  form.setFieldValue("target_date", value?.to ? (renderFormattedPayloadDate(value.to) ?? null) : null);
                }}
                placeholder={{ from: t("start_date"), to: t("end_date") }}
                hideIcon={{ to: true }}
                tabIndex={getIndex("date_range")}
                disabled={isSubmitting}
              />
              <form.Field name="status">
                {(field) => (
                  <div className="h-7">
                    <ModuleStatusSelect
                      value={field.state.value}
                      onChange={field.handleChange}
                      disabled={isSubmitting}
                      tabIndex={getIndex("status")}
                    />
                  </div>
                )}
              </form.Field>
              <form.Field name="lead_id">
                {(field) => (
                  <div className="h-7">
                    <MemberDropdown
                      workspaceSlug={workspaceSlug}
                      value={field.state.value}
                      onChange={field.handleChange}
                      projectId={values.project_id}
                      multiple={false}
                      disabled={isSubmitting}
                      buttonVariant="border-with-text"
                      placeholder={t("lead")}
                      tabIndex={getIndex("lead")}
                    />
                  </div>
                )}
              </form.Field>
              <form.Field name="member_ids">
                {(field) => (
                  <div className="h-7">
                    <MemberDropdown
                      workspaceSlug={workspaceSlug}
                      value={field.state.value}
                      onChange={field.handleChange}
                      projectId={values.project_id}
                      multiple
                      disabled={isSubmitting}
                      buttonVariant={field.state.value.length > 0 ? "transparent-without-text" : "border-with-text"}
                      placeholder={t("members")}
                      tabIndex={getIndex("member_ids")}
                    />
                  </div>
                )}
              </form.Field>
            </div>
            {errors.map((error) => (
              <p key={error} role="alert" className="text-11 text-danger-primary">
                {error}
              </p>
            ))}
          </div>
        </div>
        <div className="flex items-center justify-end gap-2 border-t border-subtle px-5 py-4">
          <Button
            type="button"
            variant="secondary"
            size="lg"
            onClick={close}
            disabled={isSubmitting}
            tabIndex={getIndex("cancel")}
          >
            {t("cancel")}
          </Button>
          <Button
            variant="primary"
            size="lg"
            type="submit"
            loading={isSubmitting}
            disabled={isSubmitting}
            tabIndex={getIndex("submit")}
          >
            {t(data ? "project_module.update_module" : "project_module.create_module")}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
}
