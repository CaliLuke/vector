/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file in the project root for details.
 */

import React from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { WORKSPACE_SETTINGS_TRACKER_ELEMENTS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import type { TWebhookEventTypes } from "@plane/types";
import type { IWebhookRecord } from "@/queries/webhooks";
import { WebhookIndividualEventOptions, type WebhookEvents } from "./individual-event-options";
import { WebhookInput } from "./input";
import { WebhookOptions } from "./event-types";
import { WebhookSecretKey } from "./secret-key";
import { WebhookToggle } from "./toggle";

type Props = {
  data?: Partial<IWebhookRecord>;
  secretKey?: string | null;
  onRegenerate?: () => Promise<boolean | "stale" | void>;
  onSubmit: (data: Partial<IWebhookRecord>) => Promise<IWebhookRecord | void>;
  handleClose?: () => void;
};

type WebhookValues = WebhookEvents & { url: string; is_active: boolean; eventType: TWebhookEventTypes };

function webhookDefaults({
  project = true,
  cycle = true,
  module = true,
  issue = true,
  issue_comment = true,
  url = "",
  is_active = true,
}: Partial<IWebhookRecord> = {}): WebhookValues {
  const events = { project, cycle, module, issue, issue_comment };
  return { ...events, url, is_active, eventType: Object.values(events).every(Boolean) ? "all" : "individual" };
}

function errorProperty(error: unknown, key: string): unknown {
  return error && typeof error === "object" && key in error ? Reflect.get(error, key) : undefined;
}

function webhookSubmissionError(error: unknown, fallback: string) {
  const urlValue = errorProperty(error, "url");
  const urlError = Array.isArray(urlValue) ? urlValue[0] : urlValue;
  return {
    urlError,
    message:
      [urlError, errorProperty(error, "error"), errorProperty(error, "detail")].find(
        (value): value is string => typeof value === "string"
      ) ?? fallback,
  };
}

export function WebhookForm({ data, onSubmit, handleClose, secretKey, onRegenerate }: Props) {
  const { t } = useTranslation();
  const form = useForm({
    defaultValues: webhookDefaults(data),
    onSubmit: async ({ value, formApi }) => {
      const { eventType, ...payload } = value;
      if (eventType === "all") {
        payload.project = payload.cycle = payload.module = payload.issue = payload.issue_comment = true;
      }
      try {
        const savedWebhook = await onSubmit(payload);
        formApi.reset(webhookDefaults(savedWebhook ?? payload));
      } catch (error) {
        const { urlError, message } = webhookSubmissionError(
          error,
          t("workspace_settings.settings.webhooks.toasts.not_created.message")
        );
        if (urlError)
          formApi.setFieldMeta("url", (meta) => ({ ...meta, errorMap: { ...meta.errorMap, onChange: message } }));
        else formApi.setErrorMap({ onSubmit: { form: message, fields: {} } });
        throw error;
      }
    },
  });
  const values = useStore(form.store, (state) => state.values);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        form.setErrorMap({ onSubmit: undefined });
        void form.handleSubmit().catch(() => undefined);
      }}
    >
      <div className="space-y-5">
        <div className="text-18 font-medium text-secondary">
          {data
            ? t("workspace_settings.settings.webhooks.modal.details")
            : t("workspace_settings.settings.webhooks.modal.title")}
        </div>
        <div className="space-y-3">
          <div className="space-y-1">
            <form.Field
              name="url"
              validators={{
                onChange: ({ value }) =>
                  value.trim() ? undefined : t("workspace_settings.settings.webhooks.modal.error"),
              }}
            >
              {(field) => (
                <>
                  <WebhookInput
                    value={field.state.value}
                    onChange={field.handleChange}
                    hasError={!field.state.meta.isValid}
                    disabled={isSubmitting}
                  />
                  {!field.state.meta.isValid && (
                    <div className="text-11 text-danger-primary" role="alert">
                      {field.state.meta.errors.join(" ")}
                    </div>
                  )}
                </>
              )}
            </form.Field>
          </div>
          {data && (
            <WebhookToggle
              value={values.is_active}
              onChange={(value) => form.setFieldValue("is_active", value)}
              disabled={isSubmitting}
            />
          )}
          <WebhookOptions
            value={values.eventType}
            onChange={(value) => form.setFieldValue("eventType", value)}
            disabled={isSubmitting}
          />
        </div>
        <div className="mt-4">
          {values.eventType === "individual" && (
            <WebhookIndividualEventOptions
              value={values}
              onChange={(key, value) => form.setFieldValue(key, value)}
              disabled={isSubmitting}
            />
          )}
        </div>
        {formError && (
          <div className="text-11 text-danger-primary" role="alert">
            {typeof formError === "string"
              ? formError
              : t("workspace_settings.settings.webhooks.toasts.not_created.message")}
          </div>
        )}
      </div>
      {data ? (
        <div className="space-y-5 pt-0">
          <WebhookSecretKey data={data} secretKey={secretKey ?? null} onRegenerate={onRegenerate} />
          <Button
            size="lg"
            type="submit"
            loading={isSubmitting}
            disabled={isSubmitting}
            data-ph-element={WORKSPACE_SETTINGS_TRACKER_ELEMENTS.WEBHOOK_UPDATE_BUTTON}
          >
            {isSubmitting ? t("updating") : t("update")}
          </Button>
        </div>
      ) : (
        <div className="flex items-center justify-end gap-2 border-t-[0.5px] border-subtle px-5 py-4">
          <Button variant="secondary" type="button" size="lg" onClick={handleClose} disabled={isSubmitting}>
            {t("cancel")}
          </Button>
          <Button
            type="submit"
            variant="primary"
            size="lg"
            loading={isSubmitting}
            disabled={isSubmitting}
            className="capitalize"
          >
            {isSubmitting ? t("common.creating") : t("common.create")}
          </Button>
        </div>
      )}
    </form>
  );
}
