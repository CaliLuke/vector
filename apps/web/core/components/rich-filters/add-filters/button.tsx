/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { ListFilter } from "lucide-react";
// plane imports
import type { TButtonSize, TButtonVariant } from "@plane/propel/button";
import { getButtonStyling } from "@plane/propel/button";
import type { TFilterProperty, TSupportedOperators } from "@plane/types";
import { cn } from "@plane/utils/common";
// local imports
import { AddFilterDropdown } from "./dropdown";
import type { TFilterExpressionEditor } from "../shared";

export type TAddFilterButtonProps<P extends TFilterProperty> = {
  buttonConfig?: {
    label: string | null;
    variant?: TButtonVariant;
    size?: TButtonSize;
    className?: string;
    defaultOpen?: boolean;
    iconConfig?: {
      shouldShowIcon: boolean;
      iconComponent?: React.ElementType;
    };
    isDisabled?: boolean;
  };
  editor: TFilterExpressionEditor<P>;
  onFilterSelect?: (id: string) => void;
};

export const AddFilterButton = function AddFilterButton<P extends TFilterProperty>(props: TAddFilterButtonProps<P>) {
  const { editor, buttonConfig, onFilterSelect } = props;
  const {
    variant = "secondary",
    size = "base",
    className,
    label,
    iconConfig = { shouldShowIcon: true },
    isDisabled = false,
  } = buttonConfig || {};
  // derived values
  const FilterIcon = iconConfig.iconComponent || ListFilter;

  const handleFilterSelect = (property: P, operator: TSupportedOperators, isNegation: boolean) => {
    editor.onAddCondition(property, operator, isNegation);
    onFilterSelect?.(property);
  };

  if (isDisabled) return null;
  return (
    <AddFilterDropdown
      {...props}
      buttonConfig={{
        ...buttonConfig,
        className: cn(getButtonStyling(variant, size), "py-[5px]", className),
      }}
      handleFilterSelect={handleFilterSelect}
      customButton={
        <div className="flex items-center gap-1">
          {iconConfig.shouldShowIcon && <FilterIcon className="size-4 text-secondary" />}
          {label}
        </div>
      }
    />
  );
};
