/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type {
  SingleOrArray,
  TFilterExpression,
  TFilterValue,
  TSupportedOperators,
  TWorkItemFilterExpression,
  TWorkItemFilterExpressionData,
  TWorkItemFilterProperty,
} from "@plane/types";
import type { IFilterAdapter } from "@plane/types";
import {
  CORE_OPERATORS,
  EXTENDED_OPERATORS,
  MULTI_VALUE_OPERATORS,
  WORK_ITEM_FILTER_PROPERTY_KEYS,
} from "@plane/types";
import {
  createConditionNode,
  createAndGroupNode,
  createOrGroupNode,
  createNotGroupNode,
  isNotGroupNode,
  isConditionNode,
} from "@plane/utils/rich-filters";

const properties = new Set<string>(WORK_ITEM_FILTER_PROPERTY_KEYS);
const legacyOperators = new Set<string>(Object.values(CORE_OPERATORS));
const operators = new Set<string>(Object.values({ ...CORE_OPERATORS, ...EXTENDED_OPERATORS }));
const logicalOperators = new Set(["and", "or", "not"]);

const toInternal = (external: TWorkItemFilterExpression): TFilterExpression<TWorkItemFilterProperty> | null => {
  if (external == null) return null;
  const entries = expressionEntries(external);
  return entries.length === 0 ? null : parseExpression(external);
};

const toExternal = (internal: TFilterExpression<TWorkItemFilterProperty> | null): TWorkItemFilterExpression =>
  internal ? serializeExpression(internal) : {};

function expressionEntries(expression: unknown): [string, unknown][] {
  if (!expression || typeof expression !== "object" || Array.isArray(expression))
    throw new Error("Expected a filter object");
  return Object.entries(expression);
}

function parseExpression(expression: unknown, depth = 1): TFilterExpression<TWorkItemFilterProperty> {
  if (depth > 5) throw new Error("Filter nesting exceeds the API limit of five levels");
  const entries = expressionEntries(expression);
  if (entries.length !== 1) throw new Error("Each filter node must contain exactly one field or logical operator");
  const [key, value] = entries[0];
  const logical = key.toLowerCase();
  if (logical === "not") return createNotGroupNode(parseExpression(value, depth + 1));
  if (logical === "and" || logical === "or") {
    if (!Array.isArray(value) || value.length === 0)
      throw new Error("Filter groups must contain at least one condition");
    const children = value.map((child) => parseExpression(child, depth + 1));
    return logical === "and" ? createAndGroupNode(children) : createOrGroupNode(children);
  }
  const [property, operator] = parseConditionKey(key);
  if (!isSerializedFilterValue(value)) throw new Error("Invalid filter value");
  return {
    ...createConditionNode({
      property,
      operator,
      value: MULTI_VALUE_OPERATORS.includes(operator) ? parseValue(value) : value,
    }),
    ...(Array.isArray(value) ? { valueFormat: "array" as const } : {}),
  };
}

function parseConditionKey(key: string): [TWorkItemFilterProperty, TSupportedOperators] {
  const separator = key.lastIndexOf("__");
  const property = separator < 0 ? key : key.slice(0, separator);
  const operator = separator < 0 ? "exact" : key.slice(separator + 2);
  if (logicalOperators.has(property.toLowerCase()) || !isFilterProperty(property) || !isFilterOperator(operator)) {
    throw new Error("Unsupported filter field or operator");
  }
  if (!property.startsWith("customproperty_") && !legacyOperators.has(operator))
    throw new Error("Unsupported standard property operator");
  if (property === "type_id" && operator !== "exact" && operator !== "in")
    throw new Error("Unsupported work item type operator");
  return [property, operator];
}

function serializeExpression(
  expression: TFilterExpression<TWorkItemFilterProperty>,
  depth = 1
): TWorkItemFilterExpressionData {
  if (depth > 5) throw new Error("Filter nesting exceeds the API limit of five levels");
  if (isConditionNode(expression)) {
    const key = `${expression.property}__${expression.operator}`;
    parseConditionKey(key);
    if (!isSerializedFilterValue(expression.value)) throw new Error("Invalid filter value");
    const value =
      expression.valueFormat !== "array" && Array.isArray(expression.value) && expression.value.length > 0
        ? expression.value.join(",")
        : expression.value;
    return { [key]: value };
  }
  if (isNotGroupNode(expression)) return { not: serializeExpression(expression.child, depth + 1) };
  if (expression.children.length === 0) throw new Error("Filter groups must contain at least one condition");
  const children = expression.children.map((child) => serializeExpression(child, depth + 1));
  return expression.logicalOperator === "or" ? { or: children } : { and: children };
}

function parseValue(value: SingleOrArray<TFilterValue>): SingleOrArray<TFilterValue> {
  if (typeof value !== "string" || !value.includes(",")) return value;
  return value.split(",");
}

type SerializedFilterValue = string | number | boolean | null;

function isSerializedFilterValue(value: unknown): value is SingleOrArray<SerializedFilterValue> {
  if (Array.isArray(value))
    return value.every((entry) => entry !== null && typeof entry === typeof value[0] && isSerializedScalar(entry));
  return isSerializedScalar(value);
}

function isSerializedScalar(value: unknown): boolean {
  return (
    value === null ||
    typeof value === "string" ||
    typeof value === "boolean" ||
    (typeof value === "number" && Number.isFinite(value))
  );
}

export const workItemFiltersAdapter: IFilterAdapter<TWorkItemFilterProperty, TWorkItemFilterExpression> = {
  toInternal,
  toExternal,
};

function isFilterProperty(value: string): value is TWorkItemFilterProperty {
  return properties.has(value) || (value.startsWith("customproperty_") && value.length > "customproperty_".length);
}
function isFilterOperator(value: string): value is TSupportedOperators {
  return operators.has(value);
}
