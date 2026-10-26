/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

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
  config: TMultiSelectFilterFieldConfig<string>;
  condition: TFilterConditionNodeForDisplay<P, string>;
  isDisabled?: boolean;
  onChange: (values: SingleOrArray<string>) => void;
};

export const MultiSelectFilterValueInput = function MultiSelectFilterValueInput<P extends TFilterProperty>(
  props: TMultiSelectFilterValueInputProps<P>
) {
  const { config, condition, isDisabled, onChange } = props;
  const { options, loading, error } = useFilterSelectOptions(config);
  const formattedOptions = useMemo(() => getFormattedOptions<string>(options), [options]);

  const handleSelectChange = (values: string[]) => {
    onChange(values);
  };

  return (
    <CustomSearchSelect
      {...getCommonCustomSearchSelectProps(isDisabled)}
      value={toFilterArray(condition.value)}
      onChange={handleSelectChange}
      options={formattedOptions}
      multiple
      disabled={loading || error || isDisabled}
      customButton={<SelectedOptionsDisplay<string> selectedValue={condition.value} options={options} />}
      defaultOpen={getFilterValueLength(condition.value) === 0}
    />
  );
};
