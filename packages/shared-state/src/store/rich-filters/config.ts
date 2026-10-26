/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type {
  SingleOrArray,
  TAllAvailableOperatorsForDisplay,
  TFilterConfig,
  TFilterProperty,
  TFilterValue,
  TOperatorSpecificConfigs,
  TSupportedOperators,
} from "@plane/types";
import { EMPTY_OPERATOR_LABEL } from "@plane/constants";
import { FILTER_FIELD_TYPE } from "@plane/types";
import {
  getDateOperatorLabel,
  getOperatorForPayload,
  getOperatorLabel,
  isDateFilterOperator,
  isDateFilterType,
} from "@plane/utils/rich-filters";

export type TOperatorOptionForDisplay = {
  value: TAllAvailableOperatorsForDisplay;
  label: string;
};

export function enabledFilterOperators<P extends TFilterProperty>(config: TFilterConfig<P>): TSupportedOperators[] {
  return Array.from(config.supportedOperatorConfigsMap.entries())
    .filter(([, option]) => option.isOperatorEnabled)
    .map(([operator]) => operator);
}

export function operatorConfig<P extends TFilterProperty>(
  config: TFilterConfig<P> | undefined,
  operator: TAllAvailableOperatorsForDisplay
): TOperatorSpecificConfigs[keyof TOperatorSpecificConfigs] | undefined {
  return config?.supportedOperatorConfigsMap.get(getOperatorForPayload(operator).operator);
}

export function operatorLabel<P extends TFilterProperty>(
  config: TFilterConfig<P>,
  operator: TAllAvailableOperatorsForDisplay | undefined
): string {
  if (!operator) return EMPTY_OPERATOR_LABEL;
  const option = operatorConfig(config, operator);
  const { isNegation } = getOperatorForPayload(operator);
  if (isNegation && option?.allowNegative && option.negOperatorLabel) return option.negOperatorLabel;
  if (!isNegation && option?.operatorLabel) return option.operatorLabel;
  if (option?.type && isDateFilterType(option.type) && isDateFilterOperator(operator))
    return getDateOperatorLabel(operator);
  return getOperatorLabel(operator);
}

export function displayOperatorForValue<P extends TFilterProperty>(
  config: TFilterConfig<P>,
  operator: TAllAvailableOperatorsForDisplay,
  value: SingleOrArray<TFilterValue>
): TAllAvailableOperatorsForDisplay {
  const option = operatorConfig(config, operator);
  if (option?.type === FILTER_FIELD_TYPE.MULTI_SELECT && (Array.isArray(value) ? value.length : 0) <= 1) {
    const singleOperator = option.singleValueOperator;
    return getOperatorForPayload(operator).isNegation ? `not_${singleOperator}` : singleOperator;
  }
  return operator;
}

export function displayOperatorOptions<P extends TFilterProperty>(
  config: TFilterConfig<P>,
  value: SingleOrArray<TFilterValue>
): TOperatorOptionForDisplay[] {
  return enabledFilterOperators(config).flatMap((operator) => {
    const displayOperator = displayOperatorForValue(config, operator, value);
    const options: TOperatorOptionForDisplay[] = [{ value: operator, label: operatorLabel(config, displayOperator) }];
    if (operatorConfig(config, operator)?.allowNegative) {
      const negativeOperator = `not_${operator}` as const;
      options.push({
        value: negativeOperator,
        label: operatorLabel(config, displayOperatorForValue(config, negativeOperator, value)),
      });
    }
    return options;
  });
}
