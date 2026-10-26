/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type {
  TAllAvailableOperatorsForDisplay,
  TFilterExpression,
  TFilterProperty,
  TSupportedOperators,
} from "@plane/types";

import { isNotGroupNode } from "../../types/core";
import { findParentChain } from "./core";

/**
 * Helper function to get the display operator for a condition.
 * This checks for NOT group context and applies negation if needed.
 * @param operator - The original operator
 * @param expression - The filter expression
 * @param conditionId - The ID of the condition
 * @returns The display operator (possibly negated)
 */
export const getDisplayOperator = <P extends TFilterProperty>(
  operator: TSupportedOperators,
  expression: TFilterExpression<P>,
  conditionId: string
): TAllAvailableOperatorsForDisplay => {
  const parents = findParentChain(expression, conditionId) ?? [];
  // Group negation is rendered on the group itself, not distributed to its leaves.
  const boundary = parents.findIndex((parent) => !isNotGroupNode(parent));
  const isNegated = (boundary === -1 ? parents.length : boundary) % 2 === 1;
  return isNegated ? `not_${operator}` : operator;
};
