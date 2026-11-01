/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TFilterValue } from "@plane/types";
import React, { useMemo } from "react";

// plane imports
import type {
  SingleOrArray,
  TFilterProperty,
  TMultiSelectFilterFieldConfig,
  TFilterConditionNodeForDisplay,
} from "@plane/types";
import { CustomSearchSelect } from "@plane/ui";
import { toFilterArray, getFilterValueLength } from "@plane/utils/rich-filters";
// local imports
import { useFilterSelectOptions } from "./use-options";
import { SelectedOptionsDisplay } from "./selected-options-display";
import { getCommonCustomSearchSelectProps, getFormattedOptions } from "./shared";

type TMultiSelectFilterValueInputProps<P extends TFilterProperty> = {
  config: TMultiSelectFilterFieldConfig<TFilterValue>;
  condition: TFilterConditionNodeForDisplay<P, TFilterValue>;
  isDisabled?: boolean;
  onChange: (values: SingleOrArray<TFilterValue>) => void;
};

export const MultiSelectFilterValueInput = function MultiSelectFilterValueInput<P extends TFilterProperty>(
  props: TMultiSelectFilterValueInputProps<P>
) {
  const { config, condition, isDisabled, onChange } = props;
  const { options, loading, error } = useFilterSelectOptions(config);
  const formattedOptions = useMemo(() => getFormattedOptions<TFilterValue>(options), [options]);

  const handleSelectChange = (values: TFilterValue[]) => {
    onChange(values.filter((value) => value !== null && value !== undefined));
  };

  return (
    <CustomSearchSelect<TFilterValue>
      {...getCommonCustomSearchSelectProps(isDisabled)}
      value={toFilterArray<TFilterValue>(condition.value)}
      onChange={handleSelectChange}
      options={formattedOptions}
      multiple
      disabled={loading || error || isDisabled}
      customButton={<SelectedOptionsDisplay<TFilterValue> selectedValue={condition.value} options={options} />}
      defaultOpen={getFilterValueLength(condition.value) === 0}
    />
  );
};
