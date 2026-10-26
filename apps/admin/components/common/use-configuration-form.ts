/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IInstanceConfiguration, IFormattedInstanceConfiguration } from "@plane/types";
import { useSettingsForm } from "./settings-form";

export function useConfigurationForm<T extends Partial<IFormattedInstanceConfiguration>>({
  defaultValues,
  save,
  successMessage,
}: {
  defaultValues: T;
  save: (values: T) => Promise<IInstanceConfiguration[]>;
  successMessage: string;
}) {
  return useSettingsForm({
    defaultValues,
    onSubmit: async ({ value, formApi }) => {
      try {
        const response = await save(value);
        const saved = { ...value };
        for (const item of response) if (item.key in saved) Object.assign(saved, { [item.key]: item.value });
        formApi.reset(saved);
        setToast({ type: TOAST_TYPE.SUCCESS, title: "Settings saved", message: successMessage });
      } catch (error) {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Unable to save configuration",
          message: "Your changes are still available. Please try again.",
        });
        throw error;
      }
    },
  });
}
