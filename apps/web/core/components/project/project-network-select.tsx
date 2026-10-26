/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { NETWORK_CHOICES } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { CustomSelect } from "@plane/ui";
import { ProjectNetworkIcon } from "./project-network-icon";

type Props = {
  value: number;
  onChange: (value: number) => void;
  disabled?: boolean;
  tabIndex?: number;
  input?: boolean;
};
export function ProjectNetworkSelect({ value, onChange, disabled, tabIndex, input }: Props) {
  const { t } = useTranslation();
  const selected = NETWORK_CHOICES.find((choice) => choice.key === value);
  return (
    <CustomSelect
      value={value}
      onChange={onChange}
      disabled={disabled}
      tabIndex={tabIndex}
      input={input}
      label={
        <span className="flex items-center gap-1">
          {selected && <ProjectNetworkIcon iconKey={selected.iconKey} />}
          {selected ? t(selected.i18n_label) : t("select_network")}
        </span>
      }
    >
      {NETWORK_CHOICES.map((choice) => (
        <CustomSelect.Option key={choice.key} value={choice.key}>
          <div className="flex items-start gap-2">
            <ProjectNetworkIcon iconKey={choice.iconKey} className="size-3.5" />
            <div>
              <p>{t(choice.i18n_label)}</p>
              <p className="text-11 text-placeholder">{t(choice.description)}</p>
            </div>
          </div>
        </CustomSelect.Option>
      ))}
    </CustomSelect>
  );
}
