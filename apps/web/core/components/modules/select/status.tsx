/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

import { MODULE_STATUS } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { StatePropertyIcon, ModuleStatusIcon } from "@plane/propel/icons";
import type { IModule } from "@plane/types";
// ui
import { CustomSelect } from "@plane/ui";
// types
// constants

type Props = {
  value: IModule["status"];
  onChange: (value: NonNullable<IModule["status"]>) => void;
  disabled?: boolean;
  error?: boolean;
  tabIndex?: number;
  badge?: boolean;
};

export function ModuleStatusSelect({ value = "backlog", onChange, disabled, error, tabIndex, badge = false }: Props) {
  const { t } = useTranslation();
  const selectedValue = MODULE_STATUS.find((status) => status.value === value);
  return (
    <CustomSelect
      customButton={
        badge ? (
          <span
            className="flex h-6 w-20 items-center justify-center rounded-sm text-center text-11"
            style={{ color: selectedValue?.color, backgroundColor: `${selectedValue?.color}20` }}
          >
            {selectedValue ? t(selectedValue.i18n_label) : t("project_modules.status.backlog")}
          </span>
        ) : undefined
      }
      value={value}
      disabled={disabled}
      label={
        <div className={`flex items-center justify-center gap-2 py-0.5 text-11 ${error ? "text-danger-primary" : ""}`}>
          {value ? (
            <ModuleStatusIcon status={value} />
          ) : (
            <StatePropertyIcon className={`h-3 w-3 ${error ? "text-danger-primary" : "text-secondary"}`} />
          )}
          {(selectedValue && t(selectedValue?.i18n_label)) ?? (
            <span className={`${error ? "text-danger-primary" : "text-secondary"}`}>Status</span>
          )}
        </div>
      }
      onChange={onChange}
      tabIndex={tabIndex}
      noChevron
    >
      {MODULE_STATUS.map((status) => (
        <CustomSelect.Option key={status.value} value={status.value}>
          <div className="flex items-center gap-2">
            <ModuleStatusIcon status={status.value} />
            {t(status.i18n_label)}
          </div>
        </CustomSelect.Option>
      ))}
    </CustomSelect>
  );
}
