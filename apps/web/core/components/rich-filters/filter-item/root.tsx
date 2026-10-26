/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type {
  SingleOrArray,
  TFilterProperty,
  TFilterValue,
  TFilterConditionNodeForDisplay,
  TAllAvailableOperatorsForDisplay,
} from "@plane/types";
import {
  displayOperatorForValue,
  displayOperatorOptions,
  filterConfigByProperty,
  operatorConfig,
  operatorLabel,
} from "@plane/shared-state";
import { CustomSearchSelect } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { getOperatorForPayload } from "@plane/utils/rich-filters";
// local imports
import { FilterValueInput } from "../filter-value-input/root";
import { COMMON_FILTER_ITEM_BORDER_CLASSNAME } from "../shared";
import { FilterItemCloseButton } from "./close-button";
import { FilterItemContainer } from "./container";
import { InvalidFilterItem } from "./invalid";
import { FilterItemLoader } from "./loader";
import { FilterItemProperty } from "./property";
import type { TFilterExpressionEditor } from "../shared";

export interface IFilterItemProps<P extends TFilterProperty> {
  condition: TFilterConditionNodeForDisplay<P, TFilterValue>;
  editor: TFilterExpressionEditor<P>;
  isDisabled?: boolean;
  showTransition?: boolean;
}

export const FilterItem = function FilterItem<P extends TFilterProperty>(props: IFilterItemProps<P>) {
  const { condition, editor, isDisabled = false, showTransition = true } = props;
  const { filterConfig, operatorOptions, selectedOperatorFieldConfig, selectedOperatorOption } = getFilterItemData(
    editor,
    condition
  );
  // Disable operator selection when filter is disabled or only one operator option is available and selected
  const isOperatorSelectionDisabled = operatorSelectionDisabled(isDisabled, operatorOptions, condition.operator);

  const handleOperatorChange = (operator: TAllAvailableOperatorsForDisplay) =>
    updateFilterItemOperator(editor, condition.id, operator);

  const handleValueChange = (values: SingleOrArray<TFilterValue>) => {
    editor.onUpdateConditionValue(condition.id, values);
  };

  if (!editor.isConfigReady) {
    return <FilterItemLoader />;
  }

  if (!filterConfig?.isEnabled || !selectedOperatorFieldConfig?.isOperatorEnabled) {
    return (
      <InvalidFilterItem
        condition={condition}
        editor={editor}
        isDisabled={isDisabled}
        showTransition={showTransition}
      />
    );
  }

  return (
    <FilterItemContainer conditionValue={condition.value} showTransition={showTransition}>
      {/* Property section */}
      <FilterItemProperty
        conditionId={condition.id}
        editor={editor}
        icon={filterConfig.icon}
        isDisabled={isDisabled}
        label={filterConfig.label}
        tooltipContent={filterConfig.tooltipContent}
      />

      {/* Operator section */}
      <CustomSearchSelect
        value={condition.operator}
        onChange={handleOperatorChange}
        options={operatorOptions}
        className={COMMON_FILTER_ITEM_BORDER_CLASSNAME}
        customButtonClassName={cn(
          "h-full px-2 text-13 font-regular",
          isOperatorSelectionDisabled && "hover:bg-layer-2-hover"
        )}
        optionsClassName="w-48"
        maxHeight="2xl"
        disabled={isOperatorSelectionDisabled}
        customButton={
          <div className="flex h-full items-center" aria-disabled={isOperatorSelectionDisabled}>
            {filterConfig ? operatorLabel(filterConfig, selectedOperatorOption) : undefined}
          </div>
        }
      />

      {/* Value section */}
      {selectedOperatorFieldConfig && (
        <FilterValueInput
          filterFieldConfig={selectedOperatorFieldConfig}
          condition={condition}
          onChange={handleValueChange}
          isDisabled={isDisabled}
        />
      )}

      {/* Remove button */}
      {!isDisabled && <FilterItemCloseButton conditionId={condition.id} editor={editor} />}
    </FilterItemContainer>
  );
};

function getFilterItemData<P extends TFilterProperty>(
  editor: TFilterExpressionEditor<P>,
  condition: IFilterItemProps<P>["condition"]
) {
  const filterConfig = filterConfigByProperty(editor.configs, condition.property);
  const operatorOptions = filterConfig
    ? displayOperatorOptions(filterConfig, condition.value).map((option) => ({
        value: option.value,
        content: option.label,
        query: option.label.toLowerCase(),
      }))
    : undefined;
  const selectedOperatorFieldConfig = filterConfig ? operatorConfig(filterConfig, condition.operator) : undefined;
  const selectedOperatorOption = filterConfig
    ? displayOperatorForValue(filterConfig, condition.operator, condition.value)
    : undefined;
  return { filterConfig, operatorOptions, selectedOperatorFieldConfig, selectedOperatorOption };
}

function updateFilterItemOperator<P extends TFilterProperty>(
  editor: TFilterExpressionEditor<P>,
  conditionId: string,
  operator: TAllAvailableOperatorsForDisplay
) {
  if (!operator) return;
  const { operator: positiveOperator, isNegation } = getOperatorForPayload(operator);
  editor.onUpdateConditionOperator(conditionId, positiveOperator, isNegation);
}

function operatorSelectionDisabled(
  isDisabled: boolean,
  options: { value: TAllAvailableOperatorsForDisplay }[] | undefined,
  selectedOperator: TAllAvailableOperatorsForDisplay
) {
  return isDisabled || (options?.length === 1 && options[0]?.value === selectedOperator);
}
