/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { validateTitle } from "@/helpers/form-validation";
import { useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
// plane imports
import { ETabIndices } from "@plane/constants";
// types
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import type { ICycle } from "@plane/types";
// ui
import { Input, TextArea, ModalForm, EModalPosition, EModalWidth } from "@plane/ui";
import { getDate, renderFormattedPayloadDate } from "@plane/utils/datetime";
import { getTabIndex } from "@plane/utils/tab-indices";
// components
import { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { ProjectDropdown } from "@/components/dropdowns/project/dropdown";
// hooks
import { useCreationPermissions } from "@/hooks/use-creation-permissions";

type Props = {
  handleFormSubmit: (values: Partial<ICycle>) => Promise<void>;
  handleClose: () => void;
  projectId: string;
  data?: ICycle | null;
  isMobile?: boolean;
};

function cycleDefaults(data: ICycle | null | undefined, projectId: string) {
  return {
    project_id: data?.project_id ?? projectId,
    name: data?.name ?? "",
    description: data?.description ?? "",
    start_date: data?.start_date ?? null,
    end_date: data?.end_date ?? null,
  };
}

export function CycleForm(props: Props) {
  const { handleFormSubmit, handleClose, projectId, data, isMobile = false } = props;
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const { projectsWithCreatePermissions } = useCreationPermissions();
  const [today] = useState(() => new Date());
  const form = useForm({
    defaultValues: cycleDefaults(data, projectId),
    onSubmit: async ({ value }) => handleFormSubmit(value),
  });
  const values = useStore(form.store, (state) => state.values);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);

  const { getIndex } = getTabIndex(ETabIndices.PROJECT_CYCLE, isMobile);

  return (
    <ModalForm
      busy={isSubmitting}
      onSubmit={() => {
        form.setErrorMap({ onSubmit: undefined });
        return form.handleSubmit().catch((error: unknown) => {
          form.setErrorMap({
            onSubmit: {
              form: error instanceof Error ? error.message : "Could not save the cycle. Please try again.",
              fields: {},
            },
          });
        });
      }}
      handleClose={handleClose}
      position={EModalPosition.TOP}
      width={EModalWidth.XXL}
    >
      <fieldset disabled={isSubmitting} inert={isSubmitting} className="space-y-5 p-5">
        <div className="flex items-center gap-x-3">
          {!data && (
            <div className="h-7">
              <ProjectDropdown
                value={values.project_id}
                onChange={(value) => {
                  if (!Array.isArray(value)) form.setFieldValue("project_id", value);
                }}
                multiple={false}
                disabled={isSubmitting}
                buttonVariant="border-with-text"
                renderCondition={(id) => !!projectsWithCreatePermissions?.[id]}
                tabIndex={getIndex("cover_image")}
              />
            </div>
          )}
          <h3 className="text-18 font-medium text-secondary">
            {data ? t("project_cycles.update_cycle") : t("project_cycles.create_cycle")}
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
              <div className="space-y-1">
                <Input
                  name={field.name}
                  value={field.state.value}
                  onChange={(event) => field.handleChange(event.target.value)}
                  onBlur={field.handleBlur}
                  placeholder={t("title")}
                  className="w-full text-14"
                  inputSize="md"
                  hasError={!field.state.meta.isValid}
                  tabIndex={getIndex("description")}
                />
                {field.state.meta.errors.map((error) => (
                  <p role="alert" className="text-11 text-danger-primary" key={error}>
                    {error}
                  </p>
                ))}
              </div>
            )}
          </form.Field>
          <TextArea
            name="description"
            placeholder={t("description")}
            className="min-h-24 w-full resize-none text-14"
            value={values.description}
            onChange={(event) => form.setFieldValue("description", event.target.value)}
            tabIndex={getIndex("description")}
          />
          <div className="flex flex-wrap items-center gap-2">
            <DateRangeDropdown
              buttonVariant="border-with-text"
              className="h-7"
              minDate={today}
              value={{
                from: getDate(values.start_date),
                to: getDate(values.end_date),
              }}
              onSelect={(val) => {
                form.setFieldValue("start_date", renderFormattedPayloadDate(val?.from) ?? null);
                form.setFieldValue("end_date", renderFormattedPayloadDate(val?.to) ?? null);
              }}
              placeholder={{
                from: "Start date",
                to: "End date",
              }}
              hideIcon={{
                to: true,
              }}
              tabIndex={getIndex("date_range")}
            />
          </div>
        </div>
        {formError && (
          <p role="alert" className="text-13 text-danger-primary">
            {String(formError)}
          </p>
        )}
      </fieldset>
      <div className="flex items-center justify-end gap-2 border-t-[0.5px] border-subtle px-5 py-4">
        <Button
          type="button"
          disabled={isSubmitting}
          variant="secondary"
          size="lg"
          onClick={handleClose}
          tabIndex={getIndex("cancel")}
        >
          {t("common.cancel")}
        </Button>
        <Button variant="primary" size="lg" type="submit" loading={isSubmitting} tabIndex={getIndex("submit")}>
          {data
            ? isSubmitting
              ? t("common.updating")
              : t("project_cycles.update_cycle")
            : isSubmitting
              ? t("common.creating")
              : t("project_cycles.create_cycle")}
        </Button>
      </div>
    </ModalForm>
  );
}
