/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type {
  TBuildFilterExpressionParams,
  TExternalFilter,
  TFilterExpression,
  TFilterProperty,
  TFilterValue,
} from "@plane/types";
import {
  createAndGroupNode,
  createConditionNode,
  createNotGroupNode,
  getOperatorForPayload,
  hasValidValue,
  sanitizeAndStabilizeExpression,
} from "@plane/utils/rich-filters";

/**
 * Builds a temporary filter expression from conditions.
 * @param params.conditions - The conditions for building the filter expression.
 * @param params.adapter - The adapter for building the filter expression.
 * @returns The temporary filter expression.
 */
export const buildTempFilterExpressionFromConditions = <
  P extends TFilterProperty,
  V extends TFilterValue,
  E extends TExternalFilter,
>(
  params: TBuildFilterExpressionParams<P, V, E>
): E | undefined => {
  const { conditions, adapter } = params;
  let expression: TFilterExpression<P> | null = null;
  for (const condition of conditions) {
    const { operator, isNegation } = getOperatorForPayload(condition.operator);
    const node = createConditionNode({ property: condition.property, operator, value: condition.value });
    const next = isNegation ? createNotGroupNode(node) : node;
    expression = expression ? createAndGroupNode([expression, next]) : next;
  }
  if (!expression) return undefined;
  const hasAppliedCondition = conditions.some((condition) => hasValidValue(condition.value));
  return hasAppliedCondition
    ? adapter.toExternal(sanitizeAndStabilizeExpression(expression, (condition) => hasValidValue(condition.value)))
    : undefined;
};
