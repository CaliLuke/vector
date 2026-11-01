/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@plane/i18n";
import { CheckIcon } from "@plane/propel/icons";
import { CustomMenu } from "@plane/ui";

type OrderOption<T extends string> = { key: T; descendingKey?: T; i18n_label: string };

export function OrderByMenuItems<T extends string>({
  options,
  value,
  onChange,
  hideManualDirections = false,
}: {
  options: readonly OrderOption<T>[];
  value: T | undefined;
  onChange: (value: T) => void;
  hideManualDirections?: boolean;
}) {
  const { t } = useTranslation();
  const selected = options.find(
    (option) => option.key === value || (option.descendingKey !== undefined && option.descendingKey === value)
  );
  const descending = selected?.descendingKey !== undefined && selected.descendingKey === value;
  const directionDisabled = selected?.descendingKey === undefined;
  return (
    <>
      {options.map((option) => (
        <CustomMenu.MenuItem
          key={option.key}
          className="flex items-center justify-between gap-2"
          onClick={() => onChange(descending ? (option.descendingKey ?? option.key) : option.key)}
        >
          {t(option.i18n_label)}
          {selected === option && <CheckIcon className="h-3 w-3" />}
        </CustomMenu.MenuItem>
      ))}
      {!(hideManualDirections && directionDisabled) && (
        <>
          <hr className="my-2 border-subtle" />
          <CustomMenu.MenuItem
            className="flex items-center justify-between gap-2"
            disabled={directionDisabled}
            onClick={() => {
              if (selected && descending) onChange(selected.key);
            }}
          >
            Ascending
            {!directionDisabled && !descending && <CheckIcon className="h-3 w-3" />}
          </CustomMenu.MenuItem>
          <CustomMenu.MenuItem
            className="flex items-center justify-between gap-2"
            disabled={directionDisabled}
            onClick={() => {
              if (selected?.descendingKey !== undefined && !descending) onChange(selected.descendingKey);
            }}
          >
            Descending
            {!directionDisabled && descending && <CheckIcon className="h-3 w-3" />}
          </CustomMenu.MenuItem>
        </>
      )}
    </>
  );
}
