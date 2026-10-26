/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import { Tooltip } from "@plane/propel/tooltip";
import { cn } from "@plane/propel/utils";
import type { TFilterProperty, TSupportedOperators } from "@plane/types";
// local imports
import { AddFilterDropdown } from "../add-filters/dropdown";
import { COMMON_FILTER_ITEM_BORDER_CLASSNAME } from "../shared";
import type { TFilterExpressionEditor } from "../shared";

interface IFilterItemPropertyProps<P extends TFilterProperty> {
  conditionId: string;
  icon: React.FC<React.SVGAttributes<SVGElement>> | undefined;
  isDisabled?: boolean;
  editor: TFilterExpressionEditor<P>;
  label: string;
  tooltipContent?: React.ReactNode | undefined;
}

export const FilterItemProperty = function FilterItemProperty<P extends TFilterProperty>(
  props: IFilterItemPropertyProps<P>
) {
  const { conditionId, editor, isDisabled } = props;

  if (isDisabled) {
    return <PropertyButton {...props} />;
  }

  const handleFilterSelect = (property: P, operator: TSupportedOperators, isNegation: boolean) => {
    editor.onUpdateConditionProperty(conditionId, property, operator, isNegation);
  };

  return (
    <AddFilterDropdown
      {...props}
      handleFilterSelect={handleFilterSelect}
      customButton={<PropertyButton {...props} />}
    />
  );
};

type TPropertyButtonProps<P extends TFilterProperty> = IFilterItemPropertyProps<P> & {
  className?: string;
};

function PropertyButton<P extends TFilterProperty>(props: TPropertyButtonProps<P>) {
  const { icon: Icon, label, tooltipContent, className } = props;

  return (
    <Tooltip tooltipContent={tooltipContent} position="bottom-start" disabled={!tooltipContent}>
      <div
        className={cn(
          "flex h-full min-w-0 items-center gap-1 px-2 py-[5px] text-11 text-tertiary",
          COMMON_FILTER_ITEM_BORDER_CLASSNAME,
          className
        )}
      >
        {Icon && (
          <div className="flex-shrink-0 transition-transform duration-200 ease-in-out">
            <Icon className="size-3.5" />
          </div>
        )}
        <span className="truncate">{label}</span>
      </div>
    </Tooltip>
  );
}
