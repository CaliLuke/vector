/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TAllAvailableOperatorsForDisplay, TSupportedOperators } from "@plane/types";

/**
 * Result type for operator conversion
 */
export type TOperatorForPayload = {
  operator: TSupportedOperators;
  isNegation: boolean;
};

/**
 * Converts a display operator to the format needed for supported by filter expression condition.
 * @param displayOperator - The operator from the UI
 * @returns Object with supported operator and negation flag
 */
export const getOperatorForPayload = (displayOperator: TAllAvailableOperatorsForDisplay): TOperatorForPayload => {
  switch (displayOperator) {
    case "not_exact":
      return { operator: "exact", isNegation: true };
    case "not_in":
      return { operator: "in", isNegation: true };
    case "not_range":
      return { operator: "range", isNegation: true };
    case "not_contains":
      return { operator: "contains", isNegation: true };
    case "not_isnull":
      return { operator: "isnull", isNegation: true };
    case "not_lt":
      return { operator: "lt", isNegation: true };
    case "not_lte":
      return { operator: "lte", isNegation: true };
    case "not_gt":
      return { operator: "gt", isNegation: true };
    case "not_gte":
      return { operator: "gte", isNegation: true };
    default:
      return { operator: displayOperator, isNegation: false };
  }
};
