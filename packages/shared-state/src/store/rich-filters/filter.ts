/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import cloneDeep from "lodash-es/cloneDeep.js";
import type {
  TFilterConditionNode,
  TFilterConditionNodeForDisplay,
  TFilterExpression,
  TFilterProperty,
  TFilterValue,
} from "@plane/types";
import {
  deepCompareFilterExpressions,
  extractConditions,
  extractConditionsWithDisplayOperators,
  findConditionsByPropertyAndOperator,
  findConjunctiveCondition,
  hasValidValue,
} from "@plane/utils/rich-filters";

export type TFilterExpressionSnapshot<P extends TFilterProperty> = {
  expression: TFilterExpression<P> | null;
  initialExpression: TFilterExpression<P> | null;
  committedConditionIds: ReadonlySet<string>;
  conversionError: string | null;
};

export function filterConditions<P extends TFilterProperty>(
  expression: TFilterExpression<P> | null
): TFilterConditionNode<P, TFilterValue>[] {
  return expression ? extractConditions(expression) : [];
}

export function displayFilterConditions<P extends TFilterProperty>(
  expression: TFilterExpression<P> | null
): TFilterConditionNodeForDisplay<P, TFilterValue>[] {
  return expression ? extractConditionsWithDisplayOperators(expression) : [];
}

export function findFilterConditions<P extends TFilterProperty>(
  expression: TFilterExpression<P> | null,
  property: P,
  operator: Parameters<typeof findConditionsByPropertyAndOperator<P>>[2]
): TFilterConditionNodeForDisplay<P, TFilterValue>[] {
  return expression ? findConditionsByPropertyAndOperator(expression, property, operator) : [];
}

export function firstFilterCondition<P extends TFilterProperty>(
  expression: TFilterExpression<P> | null,
  property: P,
  operator: Parameters<typeof findConditionsByPropertyAndOperator<P>>[2]
): TFilterConditionNodeForDisplay<P, TFilterValue> | undefined {
  return findFilterConditions(expression, property, operator)[0];
}

export function conjunctiveFilterCondition<P extends TFilterProperty>(
  expression: TFilterExpression<P> | null,
  property: P,
  operator: Parameters<typeof findConjunctiveCondition<P>>[2]
): TFilterConditionNodeForDisplay<P, TFilterValue> | undefined {
  return expression ? findConjunctiveCondition(expression, property, operator) : undefined;
}

export function filterHasActiveValues<P extends TFilterProperty>(
  expression: TFilterExpression<P> | null,
  committedConditionIds: ReadonlySet<string>,
  conversionError: string | null
): boolean {
  if (conversionError) return true;
  return filterConditions(expression).some(
    (condition) =>
      hasValidValue(condition.value) || (committedConditionIds.has(condition.id) && condition.value !== undefined)
  );
}

export function filterExpressionChanged<P extends TFilterProperty>(
  initialExpression: TFilterExpression<P> | null,
  expression: TFilterExpression<P> | null,
  hasValue: (condition: TFilterConditionNode<P, TFilterValue>) => boolean
): boolean {
  return !deepCompareFilterExpressions(initialExpression, expression, hasValue);
}

export function cloneFilterExpression<P extends TFilterProperty>(
  expression: TFilterExpression<P> | null
): TFilterExpression<P> | null {
  return cloneDeep(expression);
}
