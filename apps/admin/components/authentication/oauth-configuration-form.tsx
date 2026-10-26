/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { Link } from "react-router";
import { useStore } from "@tanstack/react-form";
import { API_BASE_URL } from "@plane/constants";
import { getButtonStyling } from "@plane/propel/button";
import { ToggleSwitch } from "@plane/ui";
import type { IInstanceConfiguration } from "@plane/types";
import { ConfirmDiscardModal } from "@/components/common/confirm-discard-modal";
import { requiredField, SettingsForm } from "@/components/common/settings-form";
import { useConfigurationForm } from "@/components/common/use-configuration-form";
import { CopyField } from "@/components/common/copy-field";
import type { OAuthSettings, OAuthValues } from "./oauth-settings";

type Props = {
  settings: OAuthSettings;
  config: OAuthValues;
  save: (values: OAuthValues) => Promise<IInstanceConfiguration[]>;
};

export function OAuthConfigurationForm({ settings, config, save }: Props) {
  const [confirmDiscard, setConfirmDiscard] = useState(false);
  const defaults: OAuthValues = {};
  for (const field of settings.fields) defaults[field.key] = config[field.key] || settings.defaults[field.key] || "";
  defaults[settings.syncKey] = config[settings.syncKey] || "0";
  const form = useConfigurationForm({
    defaultValues: defaults,
    save,
    successMessage: `Your ${settings.name} authentication is configured. You should test it now.`,
  });
  const values = useStore(form.store, (state) => state.values);
  const isDefaultValue = useStore(form.store, (state) => state.isDefaultValue);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const originURL = API_BASE_URL || (typeof window !== "undefined" ? window.location.origin : "");
  const fields = settings.serviceFields(originURL, values);

  return (
    <>
      <ConfirmDiscardModal
        isOpen={confirmDiscard}
        onDiscardHref="/authentication"
        handleClose={() => setConfirmDiscard(false)}
      />
      <SettingsForm onSubmit={() => form.handleSubmit()} className="grid w-full grid-cols-2 gap-x-12 gap-y-8">
        <fieldset disabled={isSubmitting} className="col-span-2 flex flex-col gap-y-4 pt-1 md:col-span-1">
          <div className="pt-2.5 text-18 font-medium">{settings.name}-provided details for Vector</div>
          {settings.fields.map(({ key, ...input }) => (
            <form.AppField
              key={key}
              name={key}
              validators={{ onChange: input.required ? requiredField(input.label) : undefined }}
            >
              {(field) => <field.TextField {...input} />}
            </form.AppField>
          ))}
          <form.Field name={settings.syncKey}>
            {(field) => (
              <div className="flex items-center justify-between gap-4">
                <span className="text-13 text-tertiary">
                  Refresh user attributes from {settings.name} during sign in
                </span>
                <ToggleSwitch
                  label={`Refresh user attributes from ${settings.name} during sign in`}
                  value={field.state.value === "1"}
                  onChange={(enabled) => field.handleChange(enabled ? "1" : "0")}
                />
              </div>
            )}
          </form.Field>
          <div className="flex items-center gap-4 pt-4">
            <form.AppForm>
              <form.SaveButton />
            </form.AppForm>
            <Link
              to="/authentication"
              className={getButtonStyling("secondary", "lg")}
              onClick={(event) => {
                if (!isDefaultValue) {
                  event.preventDefault();
                  setConfirmDiscard(true);
                }
              }}
            >
              Go back
            </Link>
          </div>
        </fieldset>
        <div className="col-span-2 md:col-span-1">
          <div className="flex flex-col gap-y-4 rounded-lg bg-layer-1 px-6 pt-1.5 pb-4">
            <div className="pt-2 text-18 font-medium">Vector-provided details for {settings.name}</div>
            {fields.map((field) => (
              <CopyField key={field.key} label={field.label} url={field.url} description={field.description} />
            ))}
          </div>
        </div>
      </SettingsForm>
    </>
  );
}
