/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { updateInstanceOptions } from "@/queries/instance";
// plane imports
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IInstance, IInstanceAdmin } from "@plane/types";
import { Input } from "@plane/ui";
import { TelemetryStatus } from "./telemetry-status";
// components
import { useSettingsForm, requiredField, SettingsForm } from "@/components/common/settings-form";
// hooks

export interface IGeneralConfigurationForm {
  instance: IInstance;
  instanceAdmins: IInstanceAdmin[];
}

export const GeneralConfigurationForm = function GeneralConfigurationForm(props: IGeneralConfigurationForm) {
  const { instance, instanceAdmins } = props;
  // hooks
  const queryClient = useQueryClient();
  const { mutateAsync: updateInstanceInfo } = useMutation(updateInstanceOptions(queryClient));

  const form = useSettingsForm({
    defaultValues: {
      instance_name: instance.instance_name ?? "",
    },
    onSubmit: async ({ value, formApi }) => {
      try {
        const saved = await updateInstanceInfo(value);
        formApi.reset({
          instance_name: saved.instance_name ?? value.instance_name,
        });
        setToast({ type: TOAST_TYPE.SUCCESS, title: "Settings saved", message: "Settings updated successfully." });
      } catch (error) {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Unable to save settings",
          message: "Your changes are still available. Please try again.",
        });
        throw error;
      }
    },
  });

  return (
    <SettingsForm onSubmit={() => form.handleSubmit()} className="space-y-8">
      <div className="space-y-4">
        <div className="text-16 font-medium text-primary">Instance details</div>
        <div className="grid-col grid w-full grid-cols-1 items-center justify-between gap-8 md:grid-cols-2 lg:grid-cols-3">
          <form.AppField name="instance_name" validators={{ onChange: requiredField("Name of instance") }}>
            {(field) => <field.TextField type="text" label="Name of instance" placeholder="Instance name" required />}
          </form.AppField>

          <div className="flex flex-col gap-1">
            <h4 className="text-13 text-tertiary">Email</h4>
            <Input
              id="email"
              name="email"
              type="email"
              value={instanceAdmins[0]?.user_detail?.email ?? ""}
              placeholder="Admin email"
              className="w-full cursor-not-allowed !text-placeholder"
              autoComplete="on"
              disabled
            />
          </div>

          <div className="flex flex-col gap-1">
            <h4 className="text-13 text-tertiary">Instance ID</h4>
            <Input
              id="instance_id"
              name="instance_id"
              type="text"
              value={instance.instance_id}
              className="w-full cursor-not-allowed rounded-md font-medium !text-placeholder"
              disabled
            />
          </div>
        </div>
      </div>

      <TelemetryStatus />

      <div>
        <form.AppForm>
          <form.SaveButton />
        </form.AppForm>
      </div>
    </SettingsForm>
  );
};
