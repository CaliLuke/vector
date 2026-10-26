/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import cloneDeep from "lodash-es/cloneDeep.js";
import type {
  IFilterAdapter,
  TExternalFilter,
  TFilterConditionNode,
  TFilterExpression,
  TFilterProperty,
  TFilterValue,
} from "@plane/types";
import {
  extractConditions,
  getGroupChildren,
  hasValidValue,
  isConditionNode,
  sanitizeAndStabilizeExpression,
} from "@plane/utils/rich-filters";

export type TFilterReadResult<P extends TFilterProperty, E extends TExternalFilter> = {
  expression: TFilterExpression<P> | null;
  source: E | undefined;
  error: string | null;
  committedConditionIds: ReadonlySet<string>;
};

export type TFilterWriteResult<E extends TExternalFilter> = { ok: true; value: E } | { ok: false; error: string };

export function readFilterExpression<P extends TFilterProperty, E extends TExternalFilter>(
  adapter: IFilterAdapter<P, E>,
  source: E | undefined
): TFilterReadResult<P, E> {
  try {
    const expression = source === undefined ? null : adapter.toInternal(cloneDeep(source));
    return {
      expression: cloneDeep(expression),
      source: cloneDeep(source),
      error: null,
      committedConditionIds: new Set(expression ? extractConditions(expression).map(({ id }) => id) : []),
    };
  } catch {
    return {
      expression: null,
      source: cloneDeep(source),
      error: "These saved filters could not be loaded. Clear them to choose new filters.",
      committedConditionIds: new Set(),
    };
  }
}

export function filterConditionHasValue<P extends TFilterProperty>(
  condition: TFilterConditionNode<P, TFilterValue>,
  committedConditionIds: ReadonlySet<string>
): boolean {
  return hasValidValue(condition.value) || (committedConditionIds.has(condition.id) && condition.value !== undefined);
}

export function writeFilterExpression<P extends TFilterProperty, E extends TExternalFilter>(
  adapter: IFilterAdapter<P, E>,
  expression: TFilterExpression<P> | null,
  committedConditionIds: ReadonlySet<string>,
  currentError: string | null
): TFilterWriteResult<E> {
  if (currentError) return { ok: false, error: currentError };
  try {
    const value = adapter.toExternal(
      sanitizeAndStabilizeExpression(expression, (condition) =>
        filterConditionHasValue(condition, committedConditionIds)
      )
    );
    return { ok: true, value };
  } catch {
    return {
      ok: false,
      error: "These filters could not be saved. Your previous filters are still applied.",
    };
  }
}

export function acceptsFilterExpression<P extends TFilterProperty>(expression: TFilterExpression<P> | null): boolean {
  return expression === null || fitsDepth(expression, 1);
}

export function filterExpressionHasValidConditions<P extends TFilterProperty>(
  expression: TFilterExpression<P> | null,
  committedConditionIds: ReadonlySet<string>
): boolean {
  return Boolean(
    expression &&
    extractConditions(expression).some((condition) => filterConditionHasValue(condition, committedConditionIds))
  );
}

function fitsDepth<P extends TFilterProperty>(expression: TFilterExpression<P>, depth: number): boolean {
  if (depth > 5) return false;
  return isConditionNode(expression) || getGroupChildren(expression).every((child) => fitsDepth(child, depth + 1));
}
