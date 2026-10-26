/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
// plane imports
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import type { TFilterProperty, TSupportedOperators } from "@plane/types";
import { availableFilterConfigs, enabledFilterOperators, filterConfigByProperty } from "@plane/shared-state";
import { CustomSearchSelect } from "@plane/ui";
import { getOperatorForPayload } from "@plane/utils/rich-filters";
import type { TFilterExpressionEditor } from "../shared";

export type TAddFilterDropdownProps<P extends TFilterProperty> = {
  customButton: React.ReactNode;
  buttonConfig?: {
    className?: string;
    defaultOpen?: boolean;
    isDisabled?: boolean;
  };
  editor: TFilterExpressionEditor<P>;
  handleFilterSelect: (property: P, operator: TSupportedOperators, isNegation: boolean) => void;
};

export const AddFilterDropdown = function AddFilterDropdown<P extends TFilterProperty>(
  props: TAddFilterDropdownProps<P>
) {
  const { editor, customButton, buttonConfig } = props;
  const { className, defaultOpen = false, isDisabled = false } = buttonConfig || {};

  // Transform available filter configs to CustomSearchSelect options format
  const filterOptions = availableFilterConfigs(editor.configs, editor.conditions).map((config) => ({
    value: config.id,
    content: (
      <div className="flex items-center justify-between gap-2 text-secondary transition-all duration-200 ease-in-out">
        <div className="flex items-center gap-2">
          {config.icon && (
            <config.icon className="size-4 text-tertiary transition-transform duration-200 ease-in-out" />
          )}
          <span>{config.label}</span>
        </div>
        {config.rightContent}
      </div>
    ),
    query: config.label.toLowerCase(),
  }));

  // If all filters are applied, show disabled options
  const allFiltersApplied = filterOptions.length === 0;
  const displayOptions = allFiltersApplied
    ? [
        {
          value: "all_filters_applied",
          content: <div className="text-placeholder italic">All filters applied</div>,
          query: "all filters applied",
          disabled: true,
        },
      ]
    : filterOptions;

  const handleFilterSelect = (property: P) => {
    const config = filterConfigByProperty(editor.configs, property);
    const firstOperator = config ? enabledFilterOperators(config)[0] : undefined;
    if (firstOperator) {
      const { operator, isNegation } = getOperatorForPayload(firstOperator);
      props.handleFilterSelect(property, operator, isNegation);
    } else {
      setToast({
        title: "Filter configuration error",
        message: "This filter is not properly configured and cannot be applied",
        type: TOAST_TYPE.ERROR,
      });
    }
  };

  return (
    <div className="relative transition-all duration-200 ease-in-out">
      <CustomSearchSelect
        defaultOpen={defaultOpen}
        value={""}
        onChange={handleFilterSelect}
        options={displayOptions}
        optionsClassName="w-56"
        maxHeight="2xl"
        placement="bottom-start"
        disabled={isDisabled}
        customButtonClassName={className}
        customButton={customButton}
      />
    </div>
  );
};
