/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IFormattedInstanceConfiguration } from "@plane/types";
import { useSettingsForm, requiredField, SettingsForm } from "@/components/common/settings-form";
import { updateConfigurationsOptions } from "@/queries/instance";
import { SendTestEmailModal } from "./test-email-modal";

type Props = { config: Partial<IFormattedInstanceConfiguration> };
const emailFields = [
  { key: "EMAIL_HOST", label: "Host", placeholder: "smtp.example.com", type: "text", required: true },
  { key: "EMAIL_PORT", label: "Port", placeholder: "587", type: "text", required: true },
  {
    key: "EMAIL_FROM",
    label: "Sender's email address",
    placeholder: "no-reply@example.com",
    type: "text",
    required: true,
  },
  { key: "EMAIL_HOST_USER", label: "Username", placeholder: "support@example.com", type: "text", required: false },
  { key: "EMAIL_HOST_PASSWORD", label: "Password", placeholder: "Password", type: "password", required: false },
] as const;

function emailValues(config: Props["config"]) {
  return {
    EMAIL_HOST: config.EMAIL_HOST ?? "",
    EMAIL_PORT: config.EMAIL_PORT ?? "",
    EMAIL_FROM: config.EMAIL_FROM ?? "",
    EMAIL_HOST_USER: config.EMAIL_HOST_USER ?? "",
    EMAIL_HOST_PASSWORD: config.EMAIL_HOST_PASSWORD ?? "",
    security: config.EMAIL_USE_TLS === "1" ? "tls" : config.EMAIL_USE_SSL === "1" ? "ssl" : "none",
  };
}

function validatePort({ value }: { value: string }) {
  return /^\d+$/.test(value) && Number(value) >= 1 && Number(value) <= 65535
    ? undefined
    : "Enter a port between 1 and 65535.";
}

export function InstanceEmailForm({ config }: Props) {
  const [testEmailOpen, setTestEmailOpen] = useState(false);
  const client = useQueryClient();
  const update = useMutation(updateConfigurationsOptions(client));
  const form = useSettingsForm({
    defaultValues: emailValues(config),
    onSubmit: async ({ value, formApi }) => {
      const { security, ...credentials } = value;
      const payload = {
        ...credentials,
        ENABLE_SMTP: "1",
        EMAIL_USE_TLS: security === "tls" ? "1" : "0",
        EMAIL_USE_SSL: security === "ssl" ? "1" : "0",
      };
      try {
        const response = await update.mutateAsync(payload);
        const saved = { ...payload };
        for (const item of response) if (item.key in saved) Object.assign(saved, { [item.key]: item.value });
        formApi.reset(emailValues(saved));
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Settings saved",
          message: "Email settings updated successfully.",
        });
      } catch (error) {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Unable to save email settings",
          message: "Your changes are still available. Please try again.",
        });
        throw error;
      }
    },
  });
  return (
    <>
      {testEmailOpen && <SendTestEmailModal handleClose={() => setTestEmailOpen(false)} />}
      <SettingsForm onSubmit={() => form.handleSubmit()} className="space-y-8">
        <fieldset
          disabled={update.isPending}
          className="grid w-full max-w-4xl grid-cols-1 items-start gap-10 lg:grid-cols-2"
        >
          {emailFields.map(({ key, ...input }) => (
            <form.AppField
              key={key}
              name={key}
              validators={{
                onChange: key === "EMAIL_PORT" ? validatePort : input.required ? requiredField(input.label) : undefined,
              }}
            >
              {(field) => <field.TextField {...input} />}
            </form.AppField>
          ))}
          <form.Field name="security">
            {(field) => (
              <div className="flex flex-col gap-1">
                <label htmlFor="email-security" className="text-13 text-tertiary">
                  Email security
                </label>
                <select
                  id="email-security"
                  name={field.name}
                  value={field.state.value}
                  onBlur={field.handleBlur}
                  onChange={(event) => field.handleChange(event.target.value)}
                  className="rounded-md border border-subtle bg-surface-1 px-3 py-2 text-13"
                >
                  <option value="tls">TLS</option>
                  <option value="ssl">SSL</option>
                  <option value="none">No email security</option>
                </select>
              </div>
            )}
          </form.Field>
        </fieldset>
        <p className="text-13 text-tertiary">
          Save your settings before sending a test email. Username and password are optional.
        </p>
        <div className="flex max-w-4xl items-center gap-4 py-1">
          <form.AppForm>
            <form.SaveButton allowUnchanged={config.ENABLE_SMTP !== "1"} />
          </form.AppForm>
          <form.Subscribe selector={(state) => [state.isDefaultValue, state.isSubmitting]}>
            {([isDefaultValue, isSubmitting]) => (
              <Button
                type="button"
                variant="secondary"
                size="lg"
                onClick={() => setTestEmailOpen(true)}
                disabled={!isDefaultValue || isSubmitting || config.ENABLE_SMTP !== "1"}
              >
                Send test email
              </Button>
            )}
          </form.Subscribe>
        </div>
      </SettingsForm>
    </>
  );
}
