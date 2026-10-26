/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useMemo } from "react";

// plane imports
import type {
  TFilterProperty,
  TFilterValue,
  TSingleSelectFilterFieldConfig,
  TFilterConditionNodeForDisplay,
} from "@plane/types";
import { CustomSearchSelect } from "@plane/ui";
// local imports
import { useFilterSelectOptions } from "./use-options";
import { SelectedOptionsDisplay } from "./selected-options-display";
import { getCommonCustomSearchSelectProps, getFormattedOptions } from "./shared";

type TSingleSelectFilterValueInputProps<P extends TFilterProperty> = {
  config: TSingleSelectFilterFieldConfig<TFilterValue>;
  condition: TFilterConditionNodeForDisplay<P, TFilterValue>;
  isDisabled?: boolean;
  onChange: (value: TFilterValue) => void;
};

export const SingleSelectFilterValueInput = function SingleSelectFilterValueInput<P extends TFilterProperty>(
  props: TSingleSelectFilterValueInputProps<P>
) {
  const { config, condition, onChange, isDisabled } = props;
  const { options, loading, error } = useFilterSelectOptions(config);
  const formattedOptions = useMemo(() => getFormattedOptions(options), [options]);

  const handleSelectChange = (value: TFilterValue) => {
    if (value === condition.value) {
      onChange(null);
    } else {
      onChange(value);
    }
  };

  return (
    <CustomSearchSelect
      {...getCommonCustomSearchSelectProps(isDisabled)}
      value={condition.value ?? null}
      onChange={handleSelectChange}
      options={formattedOptions}
      multiple={false}
      disabled={loading || error || isDisabled}
      customButton={<SelectedOptionsDisplay selectedValue={condition.value} options={options} displayCount={1} />}
      defaultOpen={condition.value === null || condition.value === undefined}
    />
  );
};
