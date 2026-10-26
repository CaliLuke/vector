/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { add } from "date-fns/add";
import { useForm, useStore } from "@tanstack/react-form";
import { Calendar } from "lucide-react";
// types
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import type { IApiToken } from "@plane/types";
// ui
import { CustomSelect, Input, TextArea, ToggleSwitch } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { renderFormattedDate, renderFormattedTime } from "@plane/utils/datetime";
// components
import { DateDropdown } from "@/components/dropdowns/date";
// helpers
type Props = {
  handleClose: () => void;
  onSubmit: (data: Partial<IApiToken>) => Promise<void>;
};

const getSubmissionErrorMessage = (error: unknown): string => {
  if (error instanceof Error) return error.message;
  if (typeof error === "object" && error !== null) {
    if ("detail" in error && typeof error.detail === "string") return error.detail;
    if ("message" in error && typeof error.message === "string") return error.message;
  }
  return "Could not create API token. Please try again.";
};

const EXPIRY_DATE_OPTIONS = [
  {
    key: "1_week",
    label: "1 week",
    value: { weeks: 1 },
  },
  {
    key: "1_month",
    label: "1 month",
    value: { months: 1 },
  },
  {
    key: "3_months",
    label: "3 months",
    value: { months: 3 },
  },
  {
    key: "1_year",
    label: "1 year",
    value: { years: 1 },
  },
];

const defaultValues = {
  label: "",
  description: "",
  expired_at: "",
  customDate: null as Date | null,
  neverExpires: false,
};

const getExpiryDate = (val: string, today: Date): Date | null | undefined => {
  const dateToAdd = EXPIRY_DATE_OPTIONS.find((option) => option.key === val)?.value;
  if (dateToAdd) return add(today, dateToAdd);
  return null;
};

const getFormattedDate = (date: Date, now: Date): Date => {
  const expiry = new Date(date);
  expiry.setHours(now.getHours(), now.getMinutes(), now.getSeconds(), 0);
  return expiry;
};

export function CreateApiTokenForm({ handleClose, onSubmit }: Props) {
  const { t } = useTranslation();
  const form = useForm({
    defaultValues,
    validators: {
      onSubmit: ({ value }): string | undefined => {
        const expiration = value.neverExpires
          ? null
          : value.expired_at === "custom"
            ? value.customDate && getFormattedDate(value.customDate, new Date())
            : getExpiryDate(value.expired_at, new Date());
        return !value.neverExpires && (!expiration || expiration.getTime() <= Date.now())
          ? "Please select a future expiration date."
          : undefined;
      },
    },
    onSubmit: async ({ value, formApi }) => {
      const now = new Date();
      const expiration = value.neverExpires
        ? null
        : value.expired_at === "custom"
          ? value.customDate && getFormattedDate(value.customDate, now)
          : getExpiryDate(value.expired_at, now);
      await onSubmit({
        label: value.label.trim(),
        description: value.description,
        expired_at: expiration?.toISOString() ?? null,
      });
      formApi.reset();
    },
  });
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const { customDate, neverExpires, expired_at: expiredAt } = useStore(form.store, (state) => state.values);
  const [today] = useState(() => new Date());
  const tomorrow = add(today, { days: 1 });
  const expiryDate = getExpiryDate(expiredAt ?? "", today);
  const customDateFormatted = customDate && getFormattedDate(customDate, today);

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        form.setErrorMap({ onSubmit: undefined });
        void form.handleSubmit().catch((error: unknown) => {
          form.setErrorMap({
            onSubmit: getSubmissionErrorMessage(error),
          });
        });
      }}
    >
      <div className="space-y-5 p-5">
        <h3 className="text-18 font-medium text-secondary">
          {t("workspace_settings.settings.api_tokens.create_token")}
        </h3>
        <div className="space-y-3">
          {formError && (
            <p role="alert" className="text-11 text-danger-primary">
              {formError}
            </p>
          )}
          <div className="space-y-1">
            <form.Field
              name="label"
              validators={{
                onChange: ({ value }) =>
                  !value.trim()
                    ? t("title_is_required")
                    : value.length > 255
                      ? t("title_should_be_less_than_255_characters")
                      : undefined,
              }}
            >
              {(field) => (
                <>
                  <Input
                    name={field.name}
                    type="text"
                    value={field.state.value}
                    onChange={(event) => field.handleChange(event.target.value)}
                    onBlur={field.handleBlur}
                    disabled={isSubmitting}
                    hasError={field.state.meta.errors.length > 0}
                    placeholder={t("title")}
                    className="w-full text-14"
                  />
                  {field.state.meta.errors.map((error) => (
                    <span key={error} role="alert" className="text-11 text-danger-primary">
                      {error}
                    </span>
                  ))}
                </>
              )}
            </form.Field>
          </div>
          <form.Field name="description">
            {(field) => (
              <TextArea
                name={field.name}
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                onBlur={field.handleBlur}
                disabled={isSubmitting}
                placeholder={t("description")}
                className="min-h-24 w-full resize-none text-14"
              />
            )}
          </form.Field>
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <form.Field name="expired_at">
                {(field) => {
                  const value = field.state.value;
                  const selectedOption = EXPIRY_DATE_OPTIONS.find((option) => option.key === value);

                  return (
                    <CustomSelect
                      customButton={
                        <div
                          className={cn(
                            "flex h-7 items-center gap-2 rounded-sm border-[0.5px] border-strong px-2 py-0.5",
                            {
                              "text-placeholder": neverExpires,
                            }
                          )}
                        >
                          <Calendar className="h-3 w-3" />
                          {value === "custom"
                            ? "Custom date"
                            : selectedOption
                              ? selectedOption.label
                              : "Set expiration date"}
                        </div>
                      }
                      value={value}
                      onChange={(selection: string | null) => field.handleChange(selection ?? "")}
                      disabled={neverExpires || isSubmitting}
                    >
                      {EXPIRY_DATE_OPTIONS.map((option) => (
                        <CustomSelect.Option key={option.key} value={option.key}>
                          {option.label}
                        </CustomSelect.Option>
                      ))}
                      <CustomSelect.Option value="custom">Custom</CustomSelect.Option>
                    </CustomSelect>
                  );
                }}
              </form.Field>
              {expiredAt === "custom" && (
                <div className="h-7">
                  <DateDropdown
                    value={customDate}
                    onChange={(date) => form.setFieldValue("customDate", date)}
                    minDate={tomorrow}
                    icon={<Calendar className="h-3 w-3" />}
                    buttonVariant="border-with-text"
                    placeholder="Set date"
                    disabled={neverExpires || isSubmitting}
                  />
                </div>
              )}
            </div>
            {!neverExpires && (
              <span className="text-11 text-placeholder">
                {expiredAt === "custom"
                  ? customDate
                    ? `Expires ${renderFormattedDate(customDateFormatted ?? "")} at ${renderFormattedTime(customDateFormatted ?? "")}`
                    : null
                  : expiredAt
                    ? `Expires ${renderFormattedDate(expiryDate ?? "")} at ${renderFormattedTime(expiryDate ?? "")}`
                    : null}
              </span>
            )}
          </div>
        </div>
      </div>
      <div className="flex items-center justify-between gap-2 border-t-[0.5px] border-subtle px-5 py-4">
        <label className="flex cursor-pointer items-center gap-1.5">
          <div className="flex cursor-pointer items-center justify-center">
            <ToggleSwitch
              value={neverExpires}
              onChange={() => {
                if (!isSubmitting) form.setFieldValue("neverExpires", !neverExpires);
              }}
              disabled={isSubmitting}
              size="sm"
            />
          </div>
          <span className="text-11">{t("workspace_settings.settings.api_tokens.never_expires")}</span>
        </label>
        <div className="flex items-center gap-2">
          <Button type="button" variant="secondary" disabled={isSubmitting} onClick={handleClose}>
            {t("cancel")}
          </Button>
          <Button variant="primary" type="submit" loading={isSubmitting}>
            {isSubmitting
              ? t("workspace_settings.settings.api_tokens.generating")
              : t("workspace_settings.settings.api_tokens.generate_token")}
          </Button>
        </div>
      </div>
    </form>
  );
}
