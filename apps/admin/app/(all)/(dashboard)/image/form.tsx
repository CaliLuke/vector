/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { updateConfigurationsOptions } from "@/queries/instance";
import type { IFormattedInstanceConfiguration } from "@plane/types";
// components
import { requiredField, SettingsForm } from "@/components/common/settings-form";
import { useConfigurationForm } from "@/components/common/use-configuration-form";
// hooks

type IInstanceImageConfigForm = {
  config: Partial<IFormattedInstanceConfiguration>;
};

export function InstanceImageConfigForm(props: IInstanceImageConfigForm) {
  const { config } = props;
  const queryClient = useQueryClient();
  const { mutateAsync: updateInstanceConfigurations } = useMutation(updateConfigurationsOptions(queryClient));
  const form = useConfigurationForm({
    defaultValues: { UNSPLASH_ACCESS_KEY: config.UNSPLASH_ACCESS_KEY ?? "" },
    save: updateInstanceConfigurations,
    successMessage: "Settings updated successfully.",
  });

  return (
    <SettingsForm onSubmit={() => form.handleSubmit()} className="space-y-8">
      <div className="grid-col grid w-full grid-cols-1 items-center justify-between gap-x-16 gap-y-8 lg:grid-cols-2">
        <form.AppField name="UNSPLASH_ACCESS_KEY" validators={{ onChange: requiredField("Access key") }}>
          {(field) => (
            <field.TextField
              type="password"
              label="Access key from your Unsplash account"
              description={
                <>
                  You will find your access key in your Unsplash developer console.&nbsp;
                  <a
                    href="https://unsplash.com/documentation#creating-a-developer-account"
                    target="_blank"
                    className="text-accent-primary hover:underline"
                    rel="noreferrer"
                    aria-label="Unsplash developer account documentation"
                  >
                    Learn more.
                  </a>
                </>
              }
              placeholder="oXgq-sdfadsaeweqasdfasdf3234234rassd"
              required
            />
          )}
        </form.AppField>
      </div>

      <div>
        <form.AppForm>
          <form.SaveButton />
        </form.AppForm>
      </div>
    </SettingsForm>
  );
}
