/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

// plane imports
import type {
  TFilterConditionNode,
  TFilterValue,
  TFilterProperty,
  TMultiSelectFilterFieldConfig,
  TDateFilterFieldConfig,
  TDateRangeFilterFieldConfig,
  TFilterConditionNodeForDisplay,
} from "@plane/types";
import { FILTER_FIELD_TYPE } from "@plane/types";
import type { TFilterValueInputProps } from "../shared";
import { DateRangeFilterValueInput } from "./date/range";
import { SingleDateFilterValueInput } from "./date/single";
import { MultiSelectFilterValueInput } from "./select/multi";
import { SingleSelectFilterValueInput } from "./select/single";
import { LiteralFilterValueInput } from "./literal/input";

export const FilterValueInput = function FilterValueInput<P extends TFilterProperty>(
  props: TFilterValueInputProps<P, TFilterValue>
) {
  const { condition, filterFieldConfig, isDisabled = false, onChange } = props;

  // Single select input
  if (filterFieldConfig?.type === FILTER_FIELD_TYPE.SINGLE_SELECT) {
    return (
      <SingleSelectFilterValueInput<P>
        config={filterFieldConfig}
        condition={condition}
        isDisabled={isDisabled}
        onChange={onChange}
      />
    );
  }

  // Multi select input
  if (filterFieldConfig?.type === FILTER_FIELD_TYPE.MULTI_SELECT) {
    return (
      <MultiSelectFilterValueInput<P>
        config={filterFieldConfig as TMultiSelectFilterFieldConfig<string>}
        condition={condition as TFilterConditionNode<P, string>}
        isDisabled={isDisabled}
        onChange={onChange}
      />
    );
  }

  // Date filter input
  if (filterFieldConfig?.type === FILTER_FIELD_TYPE.DATE) {
    return (
      <SingleDateFilterValueInput<P>
        config={filterFieldConfig as TDateFilterFieldConfig<string>}
        condition={condition as TFilterConditionNodeForDisplay<P, string>}
        isDisabled={isDisabled}
        onChange={onChange}
      />
    );
  }

  // Date range filter input
  if (filterFieldConfig?.type === FILTER_FIELD_TYPE.DATE_RANGE) {
    return (
      <DateRangeFilterValueInput<P>
        config={filterFieldConfig as TDateRangeFilterFieldConfig<string>}
        condition={condition as TFilterConditionNodeForDisplay<P, string>}
        isDisabled={isDisabled}
        onChange={onChange}
      />
    );
  }

  if (
    filterFieldConfig.type === FILTER_FIELD_TYPE.TEXT ||
    filterFieldConfig.type === FILTER_FIELD_TYPE.NUMBER ||
    filterFieldConfig.type === FILTER_FIELD_TYPE.NUMBER_RANGE
  ) {
    return (
      <LiteralFilterValueInput
        condition={condition}
        config={filterFieldConfig}
        isDisabled={isDisabled}
        onChange={onChange}
      />
    );
  }

  return <AdditionalFilterValueInput {...props} />;
};

const AdditionalFilterValueInput = function AdditionalFilterValueInput<
  P extends TFilterProperty,
  V extends TFilterValue,
>(_props: TFilterValueInputProps<P, V>) {
  return (
    // Fallback
    <div className="flex h-full cursor-not-allowed items-center px-4 text-11 text-placeholder transition-opacity duration-200">
      Filter type not supported
    </div>
  );
};
