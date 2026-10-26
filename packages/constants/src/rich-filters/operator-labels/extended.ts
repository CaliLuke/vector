/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type {
  TExtendedSupportedOperators,
  TExtendedSupportedDateFilterOperators,
  TNegatedOperator,
  TSupportedDateFilterOperators,
} from "@plane/types";

/**
 * Extended operator labels
 */
export const EXTENDED_OPERATOR_LABELS_MAP: Record<TExtendedSupportedOperators, string> = {
  contains: "contains",
  isnull: "presence is",
  lt: "less than",
  lte: "at most",
  gt: "greater than",
  gte: "at least",
} as const;

/**
 * Extended date-specific operator labels
 */
export const EXTENDED_DATE_OPERATOR_LABELS_MAP: Record<TExtendedSupportedDateFilterOperators, string> = {
  lt: "before",
  lte: "on or before",
  gt: "after",
  gte: "on or after",
} as const;

/**
 * Negated operator labels for all operators
 */
export const NEGATED_OPERATOR_LABELS_MAP: Record<TNegatedOperator, string> = {
  not_exact: "is not",
  not_in: "is none of",
  not_range: "not between",
  not_lt: "not less than",
  not_lte: "more than",
  not_gt: "not greater than",
  not_gte: "less than",
  not_contains: "does not contain",
  not_isnull: "presence is not",
};

/**
 * Negated date operator labels for all date operators
 */
export const NEGATED_DATE_OPERATOR_LABELS_MAP: Record<`not_${TSupportedDateFilterOperators}`, string> = {
  not_exact: "is not",
  not_range: "not between",
  not_lt: "on or after",
  not_lte: "after",
  not_gt: "on or before",
  not_gte: "before",
};
